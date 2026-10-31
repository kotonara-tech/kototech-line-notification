---
name: infra-professional
description: Terraform の書き方と GCP の概念(プロジェクト・IAM・サービスアカウント・Workload Identity Federation・Cloud Run Job・Cloud Scheduler・Secret Manager・Artifact Registry・GCS state)のベストプラクティスと、このプロジェクト(infra/terraform/)の構成・権限設計・運用ルール。infra/ 配下の .tf を書く・直す・レビューするとき、GCP のリソースや権限(IAM)、GitHub Actions からのデプロイ・認証、Terraform の plan / apply、環境(dev / prod / ops)の追加に触れるときは、ユーザーが「Terraform」と明示しなくても必ず使うこと。
---

# infra-professional

Terraform と GCP の作法をまとめたもの。最優先は **最小権限**、**再現性(コードがすべて)**、**state と秘匿情報を守ること**。

設計の正本は `docs/engineering/02-infra.md`。食い違う場合は先にそちらを確認し、変えるなら設計ドキュメントと `docs/engineering/00-openquestion.md`(決定の記録)も同じ変更で更新する。

> **`terraform apply` は、ユーザーが明示的に指示するまで実行しない**(`CLAUDE.md`)。検証は `fmt` / `validate` / `plan` まで。

## このプロジェクトの構成

```
infra/terraform/
├── bootstrap/        # ops プロジェクトの state 用 GCS バケット(最初に 1 回。state はローカル)
├── env/{dev,prod}/   # 環境ごとのルートモジュール(state の prefix は env/dev, env/prod)
└── modules/          # apis / github_oidc / artifact_registry / secret_manager /
                      # cloud_run_job / cloud_scheduler
```

- GCP プロジェクトは **ops / dev / prod の 3 つ**。ops には state バケットだけを置く。課金アカウントは共通
- 環境は**ディレクトリで分ける**(workspace は使わない)。prod だけ `refs/heads/main` に制限する、削除保護を有効にする、といった差を、ファイルの差分として読めるようにするため
- 環境のルートモジュールは「モジュールを呼んで値を渡す」だけにする。リソースを直接書くのは、モジュールにするほどでない少数のものに限る
- Terraform は GitHub Actions から実行する(`reusable-infra-deploy.yml`)。最初の `apply` だけは、OIDC ができる前なので手元で行う

## GCP の概念(AWS との対応)

| GCP | 概念 | AWS で近いもの |
| --- | --- | --- |
| プロジェクト | リソース・IAM・クォータ・課金の単位 | アカウント |
| フォルダ / 組織 | プロジェクトの階層・ポリシーの継承 | OU / Organizations |
| 課金アカウント | 請求の支払い元。複数プロジェクトを紐付ける | 一括請求の支払いアカウント |
| サービスアカウント(SA) | アプリや CI が使う、人でない ID | IAM ロール |
| IAM のロール | 権限の束。主体(誰が)・ロール(何を)・リソース(どこに)で付与する | ポリシー |
| Workload Identity Federation(WIF) | 外部 ID(GitHub Actions)を、鍵なしで SA に結び付ける | OIDC プロバイダ + AssumeRole |
| ADC | Application Default Credentials。環境から自動で認証情報を見つける仕組み | 認証情報プロバイダチェーン |
| VPC | プロジェクトの中のネットワーク | VPC |

押さえておくこと:

- プロジェクトを分けても**ネットワークは分離されない**。分離されるのはリソース・IAM・クォータ・課金。このプロジェクトは VPC を使わない(すべてマネージドサービスの API 呼び出し)
- **API の有効化はプロジェクト単位**。使うサービスの API は `google_project_service` で有効化し、リソースは `depends_on` で待たせる
- SA に何かをさせる(Job を SA で実行する、Scheduler が SA のトークンで呼ぶ)には、操作する主体にその SA への **`iam.serviceAccounts.actAs`**(`roles/iam.serviceAccountUser`)が要る。権限エラーの定番の原因
- SA の**鍵(JSON)は作らない**。Cloud Run には SA を紐付け、CI は WIF を使う。鍵が必要になったら設計を見直す合図
- ロールは**付与する場所が狭いほど安全**。プロジェクト全体より、リソース単位(Secret、Cloud Run Job、バケット)で付ける

## Terraform の作法

### 基本

- `terraform fmt -recursive` と `terraform validate` を通す(CI でも検査される)
- `required_version` と provider の `version`(`~> 6.0` など)を固定する。`.terraform.lock.hcl` は**コミットする**(再現性のため)
- 変数には必ず `description` と `type` を付ける。リソース名・プロジェクト ID・リージョンを**ハードコードしない**(環境ごとの変数、またはモジュールの入力にする)
- 出力は、CI や他の層が使う値だけ。機密の値には `sensitive = true`
- `tfvars` はコミットしない(`*.tfvars.example` だけコミット)。CI へは Environment の variables を `TF_VAR_*` で渡す

### リソースの書き方

- 複数のリソースは `for_each`(安定したキー)で作る。`count` は「有る / 無し」の切り替えだけ。インデックスで管理すると、途中の削除で後ろが作り直しになる
- モジュールは**単一の目的**に絞り、入力を最小にする。モジュールの中で `provider` を設定しない。出力は呼び出し側が使うものだけ
- 名前は `kebab-case`、リソースのラベル(Terraform 上の識別子)は `snake_case`。単一のリソースは `this` を使う
- 依存は参照で表現する(暗黙の依存)。`depends_on` は、参照で表せない API 有効化などに限る
- `lifecycle` は理由を**コメントで書く**(例: CD がイメージを更新するため `ignore_changes`)。`ignore_changes = all` のような広い指定はしない
- 壊すと困るものは保護する(`deletion_protection`、`prevent_destroy`)。prod は有効、dev は無効
- `provisioner`(`local-exec` など)は使わない。Terraform の外でやるべきことは手順書にする

### IAM

- 追加的な `*_iam_member` を使う。`*_iam_binding` と `*_iam_policy` は**そのロール全体を上書き**するため、既存の権限を消す危険がある
- 権限を足すときは、**どの主体に・どのロールを・どのリソースに・なぜ**を、コメントかドキュメントに残す
- 強い権限(`projectIamAdmin`、`serviceAccountAdmin` など)は、実質プロジェクトの管理者権限になる。使う環境(prod は main 限定 + 承認)と理由を明記する
- `roles/owner` / `roles/editor` を付けない。必要な権限を持つ事前定義ロールを探す。広すぎるなら、カスタムロールを検討する

### 秘匿情報

- Secret Manager は **Secret の入れ物と仮の version(`CHANGE_ME`)だけ**を Terraform で作る(version が 0 件だと、参照する Cloud Run Job を作れないため)。本物の値は state に平文で残るため、Terraform 外(`gcloud secrets versions add`)で別の version として登録する
- アプリへは Cloud Run が**環境変数として注入**する(`secret_key_ref`)。参照する SA に、その Secret への `secretAccessor` だけを付ける
- 注入するのは、その Job / サービスが実際に使うものだけ
- state にも機密が入りうる前提で、バケットを保護する(公開アクセス禁止、バージョニング、アクセスできる主体を限定)

### state

- リモート(GCS)に置く。バケットは ops プロジェクトで、dev / prod のどちらを消しても失われないようにする
- バケット名は backend ブロックでは変数が使えないため直書きになる。`bootstrap/` と `env/*/backend.tf` の値を揃える
- state を直接編集しない。`terraform state mv` / `import` は、理由と手順を PR に書く
- 同時実行はロックで防がれるが、同じ環境の `apply` は並べて実行しない(CI の `concurrency` で直列化する)

## 変更の進め方

1. 設計に影響する変更は、先に `docs/engineering/02-infra.md` と `00-openquestion.md` を確認・更新する
2. `terraform fmt -recursive` → `terraform validate`(`init -backend=false` で検証できる)
3. `plan` の差分を読む。**作り直し(replace)と削除(destroy)を特に探す**。意図しない差分があれば原因を調べてから進める
4. CI では plan を保存し、その**保存したプランをそのまま `apply`** する(plan と apply の間で内容がずれない)
5. prod は `main` からのみ。Environment の承認を挟む

作成時にエラーが出ても、リソース自体は作られて健全なことがある(IAM の反映待ちなど)。Terraform はそのリソースを `tainted` にして、次の `apply` で**削除して作り直そうとする**。`deletion_protection` が有効だと削除で止まるため、先に実体の状態(`gcloud` で確認)を見て、健全なら `terraform untaint` で印を外してから `plan` を確認する。

注意: WIF のプール・SA・IAM 自体を変更する `apply` は、実行中の認証を壊す恐れがある。差分を必ず確認する。

## このプロジェクト固有の設計

### Cloud Run Job と Scheduler

- 処理は Cloud Run **Job**(HTTP エンドポイントなし)。Scheduler が毎分 **Cloud Run Jobs API の `:run`** を呼ぶ
- 呼び出し先が Google の API なので、Scheduler は **OAuth トークン**(`oauth_token`、スコープ `cloud-platform`)を使う。OIDC は、自前の HTTP エンドポイントを呼ぶときの方式(使い分けを間違えない)
- Scheduler 用 SA には、**対象の Job への `roles/run.invoker` だけ**を付ける
- 再試行は 0(毎分実行なので次回で拾う)。実行が重なりうるため、冪等性はアプリ側で担保する
- 初回デプロイ前は Job のイメージが placeholder のため、Scheduler を停止しておく(`scheduler_paused`)
- イメージは Terraform で管理しない(`ignore_changes`)。CD が `gcloud run jobs update` で更新する。環境変数と Secret は Terraform が管理する

### GitHub Actions 用 WIF

- Provider の `attribute_condition` で、対象リポジトリ(prod はさらに `refs/heads/main`)以外のトークンを拒否する
- SA への `workloadIdentityUser` も、対象リポジトリに限定する
- 認証済みの CI の SA が使う権限は、アプリのデプロイ用と Terraform 実行用に分けて変数で管理する(`deployer_roles` / `terraform_roles`)

### リソースを足すときのチェック

- [ ] 必要な API が `modules/apis` で有効化されているか
- [ ] 実行主体の SA に、**必要最小限**のロールだけを付けたか(リソース単位で)
- [ ] 機密の値を state に入れていないか
- [ ] dev と prod の差(削除保護、ref 制限など)を意識したか
- [ ] CI の SA(`github-deployer`)で `plan` / `apply` できる権限があるか(足りなければ `terraform_roles`)
- [ ] 出力が必要な値(GitHub の Environment variables に入れるもの)を出力したか
- [ ] `docs/engineering/02-infra.md` と構成図(`docs/infra/architecture.png`。`docs/infra/generate_architecture.py` を直して再生成し、目で確認する)を更新したか
