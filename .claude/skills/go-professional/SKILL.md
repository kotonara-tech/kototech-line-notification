---
name: go-professional
description: Go 言語のベストプラクティスと、このプロジェクト(backend/)の DDD 構成・ROLE 切り替え・テスト・ログ・エラー処理の作法。backend/ 配下の Go コードを書く・直す・レビューする・テストを書く・Dockerfile を書くときは、ユーザーが「Go」と明示しなくても必ず使うこと。ユースケース、ドメインモデル、LINE / Sheets 連携、エントリポイント、go.mod、golangci-lint、make test / lint / build に触れる作業も対象。
---

# go-professional

`backend/` の Go コードを、読みやすく・テストしやすく・壊れにくく書くための指針。
迷ったら「標準ライブラリで足りるか」「依存の向きが内側へ向いているか」「時刻や外部 API を差し替えられるか」を確認する。

設計の正本は `docs/engineering/03-backend.md`。この文書と食い違う場合は、先にそちらを確認し、変えるなら設計ドキュメントも同じ変更で更新する。

## このプロジェクトの前提

- Cloud Run **Job** として動く。HTTP サーバではない。将来 API を足せるよう、エントリポイントで環境変数 `ROLE`(`scan` / 将来 `api`)により役割を切り替える
- ビルドする Go バイナリ(イメージ)は **1 つだけ**
- 構成は DDD。依存は `domain` ← `application` ← `infrastructure` / `interfaces` の一方向。`domain` は標準ライブラリのみに依存する
- 秘匿情報(LINE のトークンなど)は Secret Manager から**環境変数として注入**される。アプリは Secret Manager の API を呼ばない
- LINE への送信は **broadcast**(全友だち)。送信先の概念をドメインに持たない

## 構成と依存の向き

```
backend/
├── cmd/notifier/main.go        # エントリポイント。config 読込 → ROLE の Runner 実行 → 終了コード
└── internal/
    ├── config/                 # 環境変数の読込・検証
    ├── bootstrap/              # 依存の組み立て(Composition Root)
    ├── domain/{program,notice} # ビジネスルール。外部に依存しない
    ├── application/{port,scan} # ユースケースとポート(インターフェース)
    ├── infrastructure/{sheets,line,clock}
    └── interfaces/{job,api}    # 入口のアダプタ
```

- `domain` は他の層を import しない。違反は `depguard` で検出する(`.golangci.yml` に設定する。未作成)
- インターフェースは**使う側**で定義する。リポジトリは `domain`、外部サービスのポートは `application/port`
- インターフェースは小さく保つ(メソッド 1〜3 個)。実装を 1 つしか持たない大きなインターフェースを先回りして作らない
- 具体実装の選択と注入は `bootstrap` に集める。ユースケースやドメインの中で `New...` して外部クライアントを作らない
- `internal/` 配下に置く。公開 API を持たないため、外から import されない方がよい

## コーディング指針

### 命名とパッケージ

- パッケージ名は短い小文字の名詞 1 語。`util` `common` `helper` のような受け皿を作らない
- 呼び出し側で `program.Repository` と読めるように、パッケージ名との重複を避ける(`program.ProgramRepository` ではなく `program.Repository`)
- 略語は大文字で統一する(`ID`、`URL`、`HTTP`)
- エクスポートする識別子にだけコメントを書く。コメントは識別子名で始め、**なぜ**を書く(何をしているかはコードで分かる)

### エラー

- 握りつぶさない。返すときは文脈を足して `fmt.Errorf("工程表の取得: %w", err)` のようにラップする(`%w` で原因を保つ)
- 判定には `errors.Is` / `errors.As` を使い、文字列比較をしない
- ドメインの失敗は、意味のあるエラー値(`var ErrAlreadyNotified = errors.New(...)`)や型で表す
- `panic` は起動時の致命的な不整合に限る。リクエスト処理や通常のフローでは使わない
- 複数の失敗を集めるときは `errors.Join` を使う(例: `scan` で 1 件の失敗で全体を止めず、最後にまとめて返す)
- エラーメッセージは小文字始まり・句点なし(ログで連結されるため)。ただし日本語メッセージの場合は簡潔さを優先し、プロジェクト内で統一する

### context

- I/O を伴う関数は第一引数に `ctx context.Context` を取る。構造体に context を保持しない
- 外部 API 呼び出しにはタイムアウトを付ける(`context.WithTimeout`)。無期限の待ちを作らない
- Cloud Run Job はタイムアウトやキャンセルで `SIGTERM` を受ける。`main` で `signal.NotifyContext` を使い、キャンセルを下位へ伝える

### 時刻

- **`time.Now()` をドメインやユースケースの中で直接呼ばない**。`port.Clock` を注入するか、時刻を引数で受け取る(テストで固定できる)
- タイムゾーンは JST(`Asia/Tokyo`)で解釈する。最小イメージには tzdata がないため、`import _ "time/tzdata"` で埋め込む
- 日付またぎ・境界値(ちょうど n 分前、開始時刻ちょうど)をテストに含める

### 並行処理

- 必要になるまで goroutine を足さない。MVP のスキャンは直列で十分
- 使う場合は、起動した goroutine の終了を必ず保証する(`errgroup` など)。`ctx` のキャンセルで止まるようにする
- 共有状態をロックで守るより、所有者を 1 つにして値を渡す

### ログ

- 標準の `log/slog` で JSON を標準出力へ出す(Cloud Logging が構造化して取り込む)
- 全ログに `role` と `app_env` を付ける。メッセージは固定文にし、可変の値は属性(`slog.String("program_id", id)`)に入れる
- **秘匿情報をログに出さない**。config では `Secret` 型(`String()` と JSON 出力をマスクする)で保持する
- 同じエラーを、返す側とログに出す側の両方で記録しない(二重ログを避ける)。ログは境界(`main` やランナー)で 1 回

### 設定(環境変数)

- 読み込みと検証は `config` に集め、他のパッケージは `os.Getenv` を呼ばない
- `ROLE` ごとに必須項目を検証し、足りないものは**まとめて**報告して終了コード 2 で終わる
- 既定値で曖昧に動かさない。`ROLE` が未設定なら、既定の役割で動かさずエラーにする

### 終了コード

| コード | 意味 |
| --- | --- |
| 0 | 正常終了 |
| 1 | 処理の失敗(送信失敗、外部 API エラーなど) |
| 2 | 設定エラー(`ROLE` が未設定・未知、必須の環境変数の欠落) |

`os.Exit` は `main` の 1 か所だけで呼ぶ(`defer` が走らなくなるため、`run() int` のような関数に処理を寄せる)。

### 冪等性と外部連携

- Scheduler は毎分起動し、実行が重なる可能性がある。同じプログラムへ二重送信しない設計にする
  - 通知済みの記録を信頼の源にする
  - LINE broadcast には `X-Line-Retry-Key`(UUID)がある。プログラム ID と通知の種類から決まる安定した UUID(v5 など)を付与し、二重の防止にする。有効期間などの仕様は公式を確認する
- 外部 API のレスポンスやシートの値は**信頼しない**。境界(`infrastructure`)で検証してからドメインの型に変換する
- 工程表の行 ⇔ `Program` の対応(防腐層)は `infrastructure/sheets` に閉じ込める。シートの列が変わっても、ドメインを変えずに済むようにする

## テスト

| 対象 | 方法 |
| --- | --- |
| `domain` | 外部依存なしの純粋な単体テスト。時刻を固定する |
| `application` | `Repository` / `Notifier` / `Clock` のフェイクで検証する |
| `config` | `ROLE` の未設定・未知・必須項目の欠落が終了コード 2 になること |
| `infrastructure` | `//go:build integration` を付けたインテグレーションテスト。dev 環境の実 API のみ |

- テーブル駆動テスト + `t.Run` でケース名を付ける。ケース名で「何を確認しているか」が読めるようにする
- 環境変数のテストは `t.Setenv` を使う(`t.Parallel` とは併用できない点に注意)
- 後片付けは `t.Cleanup`。テスト用ヘルパーには `t.Helper()` を付ける
- 比較は `cmp.Diff`(`go-cmp`)などで差分を出す。依存を増やしたくなければ、標準の `reflect.DeepEqual` と明確なメッセージでよい
- **モックライブラリより手書きのフェイク**を優先する。インターフェースが小さければ数行で書ける
- 実装の詳細ではなく、振る舞い(入力と出力、副作用)を検証する
- バグを直すときは、先に再現するテストを書く

## 品質ゲート

`make` のコマンドをそのまま CI でも使う。コミット前に手元でも実行する。

| コマンド | 内容 |
| --- | --- |
| `make test` | `go test -race -cover ./...`(外部 API は叩かない) |
| `make lint` | `go vet` と `golangci-lint` |
| `make integration-test` | `-tags=integration`。dev 環境の実 API を使う。prod には実行しない |
| `make build` | `bin/notifier` を 1 つビルド(`CGO_ENABLED=0 -trimpath`) |

- `golangci-lint` は v2 系。設定ファイル(`.golangci.yml`)を置くときは v2 の形式で書く
- 有効にする候補: `errcheck` `govet` `staticcheck` `unused` `ineffassign` `errorlint` `depguard` `gocritic`。ルールは増やしすぎず、実際に出た指摘を見て調整する
- `gofmt` / `goimports` で整形する。整形の議論はしない

## Dockerfile

- マルチステージにする。ビルドは `golang` イメージ、実行は最小のイメージ(`gcr.io/distroless/static` など)
- `CGO_ENABLED=0 go build -trimpath -ldflags "-s -w"` で静的ビルドし、バイナリ 1 つだけをコピーする
- 非 root で実行する(distroless の `nonroot` タグ)
- `go.mod` / `go.sum` を先にコピーして `go mod download` し、依存のレイヤーをキャッシュさせる
- `ENTRYPOINT` はバイナリのみ。`ROLE` などは環境変数(Terraform)で与え、イメージには焼き込まない

## 依存ライブラリ

- 標準ライブラリで足りるなら足す。依存が増えるほど、更新と脆弱性対応のコストが増える
- 追加するときは理由を PR に書く。候補: Sheets は `google.golang.org/api/sheets/v4`、LINE は公式 SDK か標準 `net/http`(未決: Q12)
- 認証は Application Default Credentials に任せ、鍵ファイルを扱うコードを書かない

## レビューの観点(自己チェック)

- [ ] 依存の向きが内側へ向いているか(`domain` が外を import していないか)
- [ ] `time.Now()` を直接呼んでいないか
- [ ] エラーを握りつぶさず、`%w` で文脈を付けているか
- [ ] 外部呼び出しに `ctx` とタイムアウトがあるか
- [ ] 二重送信しない設計になっているか(冪等性)
- [ ] 秘匿情報がログ・エラーメッセージに出ていないか
- [ ] 設計ドキュメント(`03-backend.md`、未決事項)と整合しているか
- [ ] `make test` と `make lint` が通るか
