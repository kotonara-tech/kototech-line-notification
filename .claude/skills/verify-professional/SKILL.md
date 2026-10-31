---
name: verify-professional
description: 作業の完了前に、必ずローカルで検証するためのスキル。`make verify` を実行して問題を洗い出し、直して、再実行する。GitHub Actions はローカルで動かせないため、これまでの失敗に基づく論理検証のチェックリストを持つ。コード・Terraform・ワークフロー(.github/)・ドキュメント・Dockerfile のいずれかを変更したあと、「完了」「できた」「直した」と報告する前、コミット・push の前には、ユーザーが「検証して」と言わなくても必ず使うこと。CI が失敗した原因の調査や、push 後に実環境で確かめるときも対象。
---

# verify-professional

**作業が終わったら、報告する前に、ローカルで検証する。** 「書いた」ことは「動く」ことの証拠ではない。
このプロジェクトでは、検証を省いたことで、次の失敗が実際に起きた(いずれも、push して CI が落ちて初めて気づいた)。

| 起きた失敗 | 防げた検証 |
| --- | --- |
| シートが読めず 403(CI のテストが、本番と別の主体で動いていた) | 「各ステップは、誰として動くか」を、論理で確認する |
| インテグレーションテストが、権限を作るインフラの適用より前に動いた | 「このジョブが前提とするものは、どのジョブが作るか」を確認する |
| `working-directory: backend` なのに、`checkout` の前で `run` した | `make verify`(自動で検出) |
| `golangci-lint` の固定版が、Go 1.27 を解析できなかった | CI と手元のバージョンの食い違いを、論理で確認する(②のF) |
| 削除した Q の番号や、廃止した環境変数名が、ドキュメントに残った | `make verify`(自動で検出) |
| GitHub の変数の先頭に空白が入っていた(`REGION`) | `gh variable list` で、実際の値を見る |
| state バケットへの権限が、CI のサービスアカウントになかった | 「CI の主体が、必要なリソースに触れるか」を確認する |

## 手順

1. **変更を終える**(コードを書く、ドキュメントを直す、など)
2. **`make verify` を実行する**。出力を**最後まで読む**
3. 失敗は、すべて直す
4. **すべて通るまで、2 → 3 を繰り返す**。直したら、必ず再実行する(直したことで、別の検証が壊れることがある)
5. 変更の種類に応じた**追加の検証**を行う(後述)
6. ローカルで確認できないこと(GitHub Actions の実行など)は、**論理検証のチェックリスト**で確認し、確認できなかったことを、正直に報告する
7. 報告する(「報告のしかた」を参照)

`make verify` が通る前に、「完了」と言わない。通らないまま、コミットや push をしない。

## `make verify`

既存のテストと lint を呼ぶだけ。独自の複雑な検査は持たない。

```
make verify
```

| 実行されるもの | 内容 |
| --- | --- |
| `make -C backend test` | ユニットテスト(`-race`) |
| `make -C backend lint` | `go vet`(`integration` タグ付きも)と `golangci-lint` |
| `make -C backend build` | 静的ビルド |
| `terraform fmt -check` / `validate` | `bootstrap`・`env/dev`・`env/prod`(`init -backend=false`) |
| `node scripts/verify.mjs` | ドキュメントと GitHub Actions(yml)の簡単な検査(下記)。**1 ファイル、依存なし** |

- どれか 1 つが失敗したら、そこで止まる。直して、最初から再実行する
- 検証の対象は、設計のドキュメント(`docs/engineering/02-infra.md` の変数表、`00-openquestion.md`)を**正本**としている。正本が古いと、検証が誤る
- Docker は、`make verify` に含めない。`Dockerfile` / `go.mod` / 起動処理を変えたときだけ、手で確認する(「変更の種類ごとの追加の検証」)

### `scripts/verify.mjs` の検査

実際に起きた失敗に絞った、**正規表現で見られる範囲だけ**の検査。ルールを増やすときも、単純さを保つ。

| 対象 | ルール |
| --- | --- |
| ドキュメント | 相対リンクが存在する / 参照する `Q` `D` の番号が `00-openquestion.md` にある / 廃止した用語(`TASK`、`LINE_CHANNEL_ACCESS_TOKEN` など)が残っていない / `docs/README.md` が 100 行以内 |
| yml | タブ文字がない / 呼ぶ reusable workflow が存在する / **呼び出し先が `id-token: write` を要求するとき、呼び出し側も許可している** / `needs` が存在するジョブを指す / **`working-directory` があるジョブで、`checkout` より前に `run` していない** / 使う `vars.*` が `02-infra.md` の表にある / `terraform apply` するワークフローが必須の変数を `TF_VAR_*` で渡す / `apply` を `cancel-in-progress: true` で中断しない |

## GitHub Actions の論理検証

Actions はローカルで動かせない(`act` も、WIF・Environment・承認・`vars` を再現できない)。
そのため、**① 静的に検出できるものは `make verify` に任せ、② 残りは、下の問いかけで論理的に確認し、③ push したあとに、実際の実行結果で確かめる**。

### ① `make verify` が自動で検出するもの

上の `scripts/verify.mjs` の yml のルール。ほかに、次のものは**自動では見ない**ので、②で確認する。

- `with` が呼び出し先の `inputs` に合っているか、必須の `inputs` を渡しているか
- アプリが読む環境変数と、Terraform が Job に渡す環境変数が合っているか
- CI と手元の、ツールのバージョン(`golangci-lint`、Terraform、Go、ベースイメージ)

### ② 論理で確認する(自動では検出できない)

ワークフローを変えたら、次を**一つずつ、自分に問う**。「たぶん大丈夫」で済ませない。

**A. 誰として動くか(主体と権限)** — 403 の原因の大半はこれ
- 各ステップは、**どのアカウントとして**動くか。`google-github-actions/auth` の後は、WIF 経由で**デプロイ用サービスアカウント**。Cloud Run Job は**実行用サービスアカウント**。この 2 つは別物
- そのアカウントが、必要なリソース(state バケット、Artifact Registry、Cloud Run、シート、Secret)に**実際に権限を持つか**。Terraform に書いてあるか。書いてあっても、**適用済みか**
- 本番と違う主体で確認していないか。CI のテストが、本番の主体と別だと、本番の設定漏れ(共有し忘れ)を検知できない。**確認したい主体として動かす**(借用など)
- 強い権限を足すとき、**どの環境に**、なぜ必要かを問う(prod には足さない、など)

**B. 順序と、前提の作成元(鶏と卵)**
- このジョブが前提とするもの(権限、リソース、変数)は、**どのジョブ・どの操作が作る**か。それは、このジョブより**前**に動くか
- 最初の 1 回だけ手作業が必要なもの(例: OIDC ができる前の最初の `apply`、state バケットへの権限付与)が、手順書にあるか
- 新しい環境・新しい変数を足したとき、**初回の実行**で何が起きるかを、頭の中で最初から追う

**C. 変数・Environment・コンテキスト**
- 使う `vars.*` は、GitHub に**実際に登録されているか**。値は正しいか(空、**先頭・末尾の空白**、別環境の値)。→ `gh variable list --env <env>` で確かめる
- `${{ vars.X || '既定' }}` は、未設定と空文字の両方で既定になる。空白だけの値は、既定にならない
- Environment の承認(Required reviewers)は、**ジョブごと**に求められる。承認が何回入るか
- PR のとき `github.ref` は `refs/pull/N/merge`。WIF の `attribute_condition`(prod は `refs/heads/main` のみ)に、**合うか**

**D. 条件・トリガー・スキップ**
- `if:` の条件(`github.ref == 'refs/heads/main'` など)が、**意図した場面だけ**で真になるか。逆に、**走るべきときに走らない**条件になっていないか
- 先行ジョブが**スキップ**されると、`needs` の後続もスキップされる(`always()` を付けない限り)。チェーンの途中が飛んで、デプロイだけが走る/走らない、という事故がないか
- `push` のブランチ、`workflow_dispatch` の入力(`boolean` は、式の中で型が変わる点に注意)

**E. 実行の安全性**
- `plan` の結果を、**そのまま** `apply` するか(保存したプラン)。`destroy` / `replace` が含まれるときは、**何が消えるか**を読む
- `apply` の前に、**削除のガード**があるか(`reusable-infra-deploy.yml`: `delete` / `replace` を含む計画を、`allow_destroy` が false なら止める)。Environment の承認は `plan` の前に求められ、承認では守れない。ガードを外していないか、`deploy-to-*` が `allow_destroy` を true で渡していないか。ガードの `jq` を変えたら、削除・replace・変更なし・`resource_changes` が無い計画の、サンプルの JSON で確かめる
- `deletion_protection` が有効なリソースが `tainted` になると、作り直しで止まる。実体が健全なら `untaint`
- 失敗して途中で止まったとき、**再実行しても安全か**(冪等か)。中途半端な状態から、続きを実行できるか
- 同じ環境の `apply` が並列にならないか(`concurrency`)

**F. 環境の違い**
- ツールのバージョン固定(`golangci-lint` が、`go.mod` の Go を解析できるか。Terraform、Go、ベースイメージ)。CI で固定している版(`reusable-qa.yml` の `GOLANGCI_LINT_VERSION` など)と、手元の `golangci-lint version` を見比べる
- **アーキテクチャ**: 手元(Apple Silicon)は `arm64`、CI は `amd64`、Cloud Run は `amd64`
- ランナーの更新(`ubuntu-latest` の移行、Node のバージョンの非推奨の警告)

**G. 秘匿情報**
- ログ・エラーメッセージ・`echo` に、シークレットやトークンが出ないか。state に平文で残らないか
- 「ステートレスなトークン」など、**保管しない**設計を、崩していないか

**H. 外部サービスの仕様に依存する主張**
- LINE・Google Sheets・GCP の挙動(上限、ステータスコード、スコープ、反映の遅延)を、**記憶で断定しない**。公式のドキュメントか、実際のログで確かめる。確かめられなければ、「未確認」と書く
- IAM の反映には**遅延**がある(作成直後に、別のリソースが参照すると失敗する。`depends_on` で順序を付ける)

### ③ push したあとに、実環境で確かめる

ワークフローは、**push して初めて本当に動く**。push したら、結果を見に行く。「push した」で終わりにしない。

```
gh run list --limit 3                       # 結果の概要
gh run view <ID>                            # どのジョブで止まったか
gh run view <ID> --log-failed               # 失敗したステップのログ
gh variable list --env dev                  # 実際に登録されている値
```

- 失敗したら、**ログの該当箇所を読んで**、原因を特定する。推測で直さない(今回の 403 も、ログで主体を確かめてから、設計の矛盾が分かった)
- 原因を直したら、**同じ種類の失敗が二度起きないよう、`make verify` に検証を足す**(後述)
- 秘匿情報を含みうるログは、必要な部分だけを抜き出す

## 変更の種類ごとの追加の検証

**Terraform(`infra/`)**
- `make verify` の `fmt` / `validate` は、**構文**の検証。`plan` ではない。`validate` が通っても、`plan` で失敗することがある
- 実環境に影響する変更は、**読み取りの `plan`** で差分を確認する(`destroy` / `replace` を探す)。認証は `GOOGLE_OAUTH_ACCESS_TOKEN=$(gcloud auth print-access-token)` を、そのコマンドだけに渡す
- **`terraform apply` は、ユーザーが明示的に指示するまで実行しない**
- 実体は `gcloud ... describe` / `list` で、読み取りで確かめる(リソースの状態、IAM、Secret の version)

**backend(Go)**
- `make verify` で、テスト・lint・ビルドが通ること。新しい振る舞いには、**先にテスト**を書く
- `Dockerfile` / `go.mod` / 起動処理を変えたら、イメージをビルドして、終了コードを確認する(`docker build -t t backend`、`docker run --rm t` が 2、`-e ROLE=api -e APP_ENV=dev` が 2、`-e ROLE=scan` に必要な環境変数を足して認証情報なしが 1)。手元は `arm64`(Cloud Run は `amd64`)。確認後は `docker rmi t`
- 実際の API(Sheets・LINE)を使うインテグレーションテストは、手元に ADC がないと動かない。動かせないなら、**未実施と報告する**(CI で確認する)

**ドキュメント**
- `make verify` の `docs` で、リンク・参照・廃止した用語を確認する
- 構成図(`docs/infra/architecture.png`)は、`generate_architecture.py` を直して再生成し、**画像を実際に開いて目で確認**する(文字のはみ出し、矢印がラベルを貫通していないか、古い表記)
- 設計を変えたら、正本(`01`〜`04`)と `00-openquestion.md` を、同じ変更で更新したか

## 検証を育てる

検証で見つけた失敗、または push 後に CI で見つけた失敗は、**同じ種類のものが二度起きないよう、検証に足す**。ただし、単純さを保つ。

1. 失敗の原因を、一般化する(「`checkout` の前に `run` した」→「`working-directory` のあるジョブで、`checkout` より前に `run` がある」)
2. **正規表現で見られるなら**、`scripts/verify.mjs` に、短いルールとして足す。見られない(論理の判断が要る)なら、上の「論理で確認する」の問いかけに足す
3. 足したルールが、**実際に失敗を検出する**ことを確かめる(失敗を再現した状態にして、`node scripts/verify.mjs` が NG を出すか)
4. `make verify` が通ることを確認する
5. このスキルの、失敗の表に反映する

## 報告のしかた

完了の報告には、**実行したことと、その結果**を書く。事実だけを、簡潔に。

```
検証: make verify … 通った(test / lint / build / Terraform / ドキュメントと yml)
追加の検証: (terraform plan の結果、docker の確認、など)
未確認: (CI でしか確認できないこと。push 後に gh run で確認する)
```

- 失敗が残っているなら、「完了」と言わず、何が残っているかを書く
- ローカルで確認できなかったことを、「問題ないはず」とぼかさない。**未確認と書く**
- 検証で、**ユーザーの依頼の範囲外の問題**(既存の不整合など)を見つけたら、勝手に広げて直さず、報告する

## やってはいけないこと

- 検証せずに、「できました」「直しました」と報告する
- `make test` だけ(または `validate` だけ)で、全体を検証したことにする
- 検証が失敗したとき、検証側を緩めて通す(誤検知と確かめたときだけ、検証を直す。直した理由を、報告する)
- ローカルで再現できないからと、論理検証を省く(ワークフローこそ、論理で追う)
- `terraform apply`、本番へのデプロイ、シークレットの読み出しを、検証の名目で実行する
