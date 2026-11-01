# 00. Open Questions

未決事項の一覧。決まったら該当する設計ドキュメントに反映し、ここでは「決定」に移す。

## 未決

| ID | 質問 | 備考 |
| --- | --- | --- |
| Q7 | 実行履歴・エラーの記録先と、失敗時の検知・通知方法 | Cloud Logging / アラート など |
| Q8 | LINE 公式アカウントのプランと月間メッセージ数の上限 | broadcast は友だち数 × 配信回数で消費される。1 プログラム 1 回の配信でも、友だち数が増えると上限に近づく |
| Q9 | Webhook の要否。必要な場合、HTTP サービスを別に追加するか | broadcast なら友だちのユーザー ID の収集は不要。返信などの双方向機能を持たせる場合のみ。受ける場合は署名検証が必要で、未認証公開になる |
| Q11 | Job の実行が重なった場合の排他 | 前回の実行が長引くと次の実行と重なる。冪等性で防ぐか、排他を入れるか |
| Q14 | dev / prod 用のテストスプレッドシート・LINE チャネル(チャネル ID・チャネルシークレット)の用意 | 環境は分ける(D8) |
| Q17 | 失敗時のアラート(Cloud Monitoring)の要否と通知先 | Q7 と関連 |
| Q20 | Environment `prod` の承認者(Required reviewers)を設定するか。設定するなら誰か。しないなら、ドキュメントの「承認あり」を直すか | 2026-11-01 に `gh api` で確かめたところ、`prod` も `dev` も `protection_rules` が空(`deployment_branch_policy` は `null`)。D16・D22、[01-operations.md](../operations/01-operations.md)、[02-infra.md](02-infra.md) の「CD」は、承認ありを前提にしている。未設定のままだと、リリース PR のマージの直後に、承認なしで prod へ反映される。D18(PR の Approve)と関連。[03-release.md](../operations/03-release.md) |
| Q21 | リリース PR(`develop` → `main`)のタイトル・本文の書き方。D34 の「必ず Issue と関連付ける」「1 PR で閉じるのは 1 Issue まで」を適用するか、リリース PR は例外にするか | リリース PR は複数の変更を含むため、1 Issue に紐付けにくい。v1.0.0 のリリース PR(#1)は、プレフィックスも `Closes` もなしでマージされた。決まったら、[01-operations.md](../operations/01-operations.md) の PR の節に例外を書き、[03-release.md](../operations/03-release.md) の手順 1 は、リンクだけにする |
| Q22 | バージョン番号(`vMAJOR.MINOR.PATCH`)を、どの変更で上げるか。Milestone 名(`v1.0.1` など)との対応 | D17。SemVer とだけ書いていて、上げる基準がない。hotfix は PATCH を上げる([01-operations.md](../operations/01-operations.md) の「リリースとバージョニング」) |
| Q23 | prod を前の状態に戻す(ロールバック)手順 | 専用の手順はない。[03-release.md](../operations/03-release.md) は、問題の変更を取り消す修正を、通常のリリースか hotfix で出すとしている。イメージとインフラ(state)をどう戻すかは、検証していない |
| Q24 | リリース後に、prod の LINE へ通知が届くことを確かめる手順 | [01-operations.md](../operations/01-operations.md) のリリースフローの手順 6「prod の動作を確認する」のうち、GCP 側の状態は [03-release.md](../operations/03-release.md) の「確認」にある。LINE に届いた通知を確かめる手順は未整備。通知は友だち全員に届く(D23)ため、どの通知で確かめるかの決めが要る |

## 決定

| ID | 決定内容 | 理由 |
| --- | --- | --- |
| D1 | GAS は使わず、Cloud Scheduler + Cloud Run + Secret Manager + Sheets API で構成する | GAS トリガーの揺らぎ回避。Sheets API 直接利用で権限管理を単純化 |
| D2 | スプレッドシートの権限管理は、サービスアカウントを共有先に追加して行う | Google の権限階層をそのまま利用する要件 |
| D3 | API Gateway 相当のサービスは使わない | エンドポイントを持たないため不要(旧: Cloud Run 標準 URL + IAM) |
| D4 | サービスアカウントは Scheduler 用と Job 実行用に分ける | 最小権限 |
| D5 | Scheduler のジョブは 1 つ(5 分ごと。旧: 毎分) | 無料枠(3 ジョブ)内。起動間隔は D35 |
| D6 | 判定は「開始まで n 分以内かつ未通知」とし、ちょうど n 分前には依存しない | 起動のずれ・取りこぼしの吸収 |
| D7 | インフラは Terraform で管理し、構成は `bootstrap/` `env/{dev,prod}` `modules/` とする | 環境差分とモジュールの再利用 |
| D8 | 環境は dev / prod の 2 つに分け、それぞれ別の GCP プロジェクトにする | リソース名の衝突回避と本番の保護 |
| D9 | tfstate は ops プロジェクトの GCS バケット(bootstrap で作成)に、環境ごとの prefix で保管する | dev / prod のどちらを消しても state が失われない |
| D10 | CD は GitHub Actions + Workload Identity Federation(鍵なし)。prod は `main` のみ許可する | 鍵の漏洩リスク排除 |
| D11 | Job のイメージは Artifact Registry に置き、Terraform は差分を無視して CD が更新する | インフラとアプリのデプロイ責務の分離 |
| D12 | バックエンドは Go。Cloud Run は **Job** として使い、HTTP エンドポイントは持たない。Scheduler が Jobs API で 5 分ごとに起動する(旧: 毎分。D35) | 純粋なコンピュート業務(03-backend.md) |
| D13 | ビルドする Go バイナリ・イメージは 1 つ。切り替えは環境変数 `ROLE` に一本化し、Job でも使う(`scan` / 将来 `api`) | 配布物の一本化。将来 API を同じコードベースで追加するため |
| D14 | MVP は Scheduler → Cloud Run Job → Sheets API / LINE の構成とする | 旧 GAS 案を廃止(D1) |
| D15 | GCP プロジェクトは ops / dev / prod の 3 つに分ける。ops には state バケットのみ置く | AWS のアカウント分割に相当。VPC は使わない |
| D16 | GitHub Actions は reusable workflow(qa / deploy-backend)を `ci` と `deploy-to-*` から呼ぶ。dev は `develop` への push + 手動、prod は手動のみ(main 限定。最新の `main` の SHA を入力し、最初のジョブ `guard` が一致を検査する。承認あり。Issue #18 で、`main` への push での起動をやめた) | main は prod の写し(operations/01-operations.md) |
| D17 | リリースは `main` のコミットにタグ(`vMAJOR.MINOR.PATCH`)を付け、GitHub Releases にリリースノートを書く | バージョンと変更点の追跡 |
| D18 | `main` の保護は、開発が安定した最後に掛ける予定だった(D33 で、マージ方法の固定のため、PR 必須まで前倒しした)。承認は、PR 必須のうえで、**他の人の Approve が 1 件必須**(自己承認・バイパスなし)にする。Approve できるメンバーが揃ってから設定する | 初期の開発速度を優先しつつ、安定後に品質を担保する |
| D19 | 全環境(ops / dev / prod)の課金アカウントは共通にする | 管理の簡素化 |
| D20 | バックエンドは DDD の構成(domain / application / infrastructure / interfaces / bootstrap)とし、依存は内側(domain)へ向かう一方向とする | ビジネスルールを外部サービスの事情から分離する |
| D21 | LINE のチャネルシークレットなどの秘匿情報は Secret Manager に保管し、Cloud Run が環境変数として注入する。アプリは Secret Manager API を呼ばない。使う Secret のみを注入する | 最小権限・SDK 依存の削減 |
| D22 | Terraform は GitHub Actions から実行する。デプロイ用サービスアカウントに Terraform 実行用のロールを付与し、state バケットへの権限は `bootstrap/` の `state_admins` で付与する | 手動の apply をなくす。実質管理者権限のため、prod は main 限定 + 承認で守る |
| D23 | 通知は Cloud Run(Job)から LINE Messaging API の **broadcast**(全友だちへ一斉配信)で送る。送信先の管理(ユーザー ID・グループ ID)は持たない | 友だち全員への通知が目的。ユーザー ID を集める Webhook が不要になる |
| D24 | GCP プロジェクトは `kototech-line-ops` / `kototech-line-dev` / `kototech-line-prod`、リージョンは `asia-northeast1`、state バケットは `kototech-line-notification-tfstate`(ops)。3 プロジェクトは同じ課金アカウントに紐付け済み | 作成済み(組織なし。個人アカウントのトップレベルのプロジェクト) |
| D25 | LINE のアクセストークンは**ステートレスなチャネルアクセストークン**(約 15 分)を使い、アプリ(`infrastructure/line` の認証トランスポート)が実行のたびに自動で発行・再発行する。アクセストークンは保管しない。注入するのは `LINE_CHANNEL_SECRET`(Secret Manager)と `LINE_CHANNEL_ID`(Terraform の変数) | 人手のローテーションによるミスを避ける。チャネルシークレットの再発行だけは手作業(02-infra.md) |
| D26 | 工程表は 1 行 1 プログラムの表とし、列はヘッダー名で探す(順序は自由、独自の列は無視する)。必須は `日付` `開始時刻` `プログラム名` `通知済み`、任意は `場所` `イベント名` `通知する` | 運用者が普通の表として編集でき、列の並べ替えや追加で壊れない(04-timetable-sheet.md) |
| D27 | 通知済みは、工程表の `通知済み` 列に日時(日本時間・RFC 3339)を書いて記録する。空でなければ通知済み(読めない値でも通知済みとして扱う)。再通知は、セルを空にする | 別の保管先を持たず、運用者が状態を見て直せる。LINE のリトライキーとの二重の重複防止 |
| D28 | 通知は、開始まで `NOTIFY_LEAD_MINUTES` 分以内(既定 10 分、Terraform の変数 `notify_lead_minutes`)で、開始前・未通知・`通知する` が有効なプログラムに送る。MVP の文面は固定(イベント名・プログラム名・開始時刻・場所を差し込む) | 起動のずれや取りこぼしを次回で吸収する(D6)。Flex Message 化は `docs/mocks/` で設計する |
| D29 | タイムゾーンは Asia/Tokyo 固定。日付をまたぐプログラムは、開始する日の日付で入力する | 国内のイベントが対象。日付またぎはテストで確認済み |
| D30 | LINE への通信は公式 SDK を使わず、標準 `net/http` で実装する。トークンの自動発行は `http.RoundTripper` で行う | トークンを自動で付ける仕組みを差し込みやすく、依存を増やさない |
| D31 | インテグレーションテスト(読み取りのみ)は、`deploy-to-dev` で**インフラの適用のあと**に実行する。デプロイ用サービスアカウントが、**実行用サービスアカウントを借りて**(`serviceAccountTokenCreator`。Terraform が dev のみ付与)、本番の Job と同じ主体として工程表を読む。デプロイ用アカウントには、シートを共有しない | 本番と同じ主体で確認でき、実行用アカウントへの共有漏れを検知できる。借りる権限はインフラが作るため、インフラの後に実行する。prod には付与しない(CI の侵害で本番の工程表を読まれないため) |
| D32 | 通知の文面は、Go のファイルの中の `text/template`(`infrastructure/line/template`、種類ごとに 1 ファイル)で作り、LINE に送る側(`infrastructure/line`)が、ドメインの `notice`(種類・元のプログラム・キー)を受け取って描画する。**通常はドメインに置く文面を、意図的に外へ出す** | 複数のテンプレートメッセージ(text・Flex など、送る形式が違うもの)を使うため。文面と LINE の形式はセットで変わり、ドメインに置くと形式の事情が入り込む。依存の向きは infrastructure → domain のまま |
| D33 | マージ方法を Ruleset でブランチごとに固定する: `develop` 宛て(作業ブランチからの PR)は**スカッシュのみ**、`main` 宛て(リリース PR)は**マージコミットのみ**。Ruleset でマージ方法を指定するには、PR 必須を同時に有効にする必要があるため、**フェーズ 2 に移行**し、`develop` と `main` への直 push を止めた。承認数は 0(必須ステータスチェックと、他の人の Approve は、メンバーが揃ってから) | スカッシュで作業中のコミットをまとめつつ、リリース PR はマージコミットにして、`main` と `develop` の履歴がずれないようにする |
| D34 | Issue・PR・ブランチの書き方を決める: ① Issue・PR のタイトルは、`【INFRA】【BE】【DOCS】【CORE】` のいずれかで始める ② 作業ブランチは `<種類>/<Issue番号>-<短い説明>` ③ PR は、先頭の `Closes #` で、必ず Issue と関連付ける ④ **1 PR で閉じるのは 1 Issue まで**(関わるだけなら `Ref #`) ⑤ PR の Milestone は、閉じる Issue と同じにする(Issue に無ければ、先に Issue に付ける) ⑥ PR の担当者は、作成者自身にする ⑦ レビュアーは、自動ではリクエストしない(`gh pr create` に `--reviewer` を付けない。人が必要なときに指定する) | 一覧から、領域と、どの Issue の作業かを分かるようにする。1 つの PR が複数の Issue を閉じると、変更の目的が混ざる。Milestone を Issue と揃えると、Issue と PR を同じ単位で追える。レビューをリクエストすると相手に通知が飛ぶため、自動では行わない |
| D35 | Scheduler の起動間隔は **5 分ごと**(`*/5 * * * *`、Asia/Tokyo)とする。通知は、開始の(`NOTIFY_LEAD_MINUTES` − 5)分前〜`NOTIFY_LEAD_MINUTES` 分前の間に届く(既定の 10 分なら 5〜10 分前)。許容する通知時刻のずれは、起動間隔(5 分)までとする(旧 Q6)。`NOTIFY_LEAD_MINUTES` は 5 以上にする | Cloud Run の Job は最低 1 分が課金されるため、毎分の起動は prod のみで月約 $44 かかる(目安)。5 分ごとなら約 $5(1 vCPU / 512 MiB、東京リージョン、30 日の月、無料枠は請求先アカウントで共有。公式の料金ページで 2026-11-01 に確認した。料金は変わりうるので、公式で再確認する)。D6 により、間隔を延ばしても通知を取りこぼさない |
