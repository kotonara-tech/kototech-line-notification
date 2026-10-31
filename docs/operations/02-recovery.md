# 02. 復旧手順(削除・作り直しが起きたとき)

Terraform が、リソースを意図せず削除・作り直ししたときの復旧手順。v1.0.0 のリリースで、prod の GitHub 変数 `GCP_PROJECT_ID` の先頭に空白が入り、Terraform がこれを別のプロジェクトと見なして、prod のリソースを作り直す計画を作った。承認の後に、その計画がそのまま `apply` され、Artifact Registry・Scheduler・デプロイ用サービスアカウント・WIF の Provider などが削除された(Issue #6)。

## 前提

- 復旧するのは、**手元のオーナー権限**を持つ人。**CI からは復旧できない**(デプロイ用サービスアカウントと WIF が消えると、CI は認証できない)
- 以降のコマンドは、**読み取り**か、**オーナーが手元で実行する**もの。`terraform apply` は、`plan` を読んで、内容を確かめてから実行する
- 秘匿情報(Secret の値、トークン)は、この文書にも、コマンドの引数にも、シェルの履歴にも残さない
- 手元で `terraform` を実行するときは、`env/<環境>` に `terraform.tfvars`(コミットしない。`terraform.tfvars.example` を参照)を置くか、`TF_VAR_*` を渡す。認証は、オーナーのアカウントのトークンを、そのコマンドだけに渡す(例: `GOOGLE_OAUTH_ACCESS_TOKEN=$(gcloud auth print-access-token) terraform plan`)
- `<PROJECT_ID>` は、復旧する環境(prod / dev)の GCP プロジェクト ID に置き換える。dev も同じ手順

## 何が起きるか

削除と作り直しは、Terraform の外にも影響が及ぶ。この影響の大きさが、`apply` を止める理由である。

| 起きること | 理由 | 対処 |
| --- | --- | --- |
| WIF の Pool と Provider を、同じ ID で作り直せない | 削除から 30 日間は `DELETED` として残り、作ると `409 Requested entity already exists` になる | 手順 2 |
| `terraform init` が `403` で失敗する | サービスアカウントを作り直すと、同じメールアドレスでも別のアカウントになる。`bootstrap/` で state バケットに付けた権限が、`deleted:serviceAccount:...?uid=...` という、効かない記録になる | 手順 3 |
| 本物の Secret の値が、仮の値に置き換わる | 仮の version が、作り直しで新しい `latest` として作られる(防止は Issue #8) | 手順 4 |
| Cloud Run Job が残る(prod のみ) | `deletion_protection` が有効なため、削除が止まる | 手順 2 の `plan` で状態を読む |

## 防ぐ仕組み

- `reusable-infra-deploy.yml` の削除のガードが、`plan` に削除(作り直しを含む)があると、`apply` を止める([../engineering/02-infra.md](../engineering/02-infra.md) の「CD」)
- **Environment `prod` の承認は、ジョブの開始時に求められ、`plan` を見る前になる**。承認の画面で、`plan` の内容は分からない。承認する前に、次を確認する
  - 実行の種類。ジョブ名が `terraform (plan)` なら plan のみ、`terraform (plan / apply)` なら apply まで進む
  - 起動した人に、`allow_destroy` を `true` にしていないかを聞く(削除を許す指定)
  - `gh variable list --env prod` で、`GCP_PROJECT_ID` などの値の先頭・末尾に、空白がないか
- 迷ったときは、その実行を承認せず、キャンセルする。次に、`apply-to-prod` を `apply` が false で起動する。**この plan のみの実行も、Environment `prod` の承認を求められる**ため、承認する(`apply` しないので安全)。ステップサマリの「削除を含む変更」を読んでから、本番の実行に進む

## 手順

### 1. 状況を確かめて、原因を直す

1. 原因を直す。同じ値のまま再実行すると、同じ作り直しが起きる。`gh variable list --env prod` で値を確認し、直す
2. 何が消えたかを、失敗した実行のログと、読み取りのコマンドで確かめる

   ```
   gh run view <ID> --log-failed
   gcloud iam service-accounts list --project=<PROJECT_ID>
   gcloud artifacts repositories list --project=<PROJECT_ID>
   gcloud scheduler jobs list --project=<PROJECT_ID> --location=asia-northeast1
   ```

3. Scheduler が動いていないことを確かめる(prod の初回は、一時停止の状態)。復旧が済むまで、再開しない

### 2. WIF の Pool と Provider を、復元して state に取り込む

Pool の ID は `github-actions`、Provider の ID は `github`(`modules/github_oidc`)。削除状態のものを確かめて、**Pool、Provider の順**に復元する。

```
gcloud iam workload-identity-pools list --project=<PROJECT_ID> --location=global --show-deleted
gcloud iam workload-identity-pools undelete github-actions --project=<PROJECT_ID> --location=global
gcloud iam workload-identity-pools providers undelete github --project=<PROJECT_ID> --location=global --workload-identity-pool=github-actions
```

復元したものを、Terraform の state に取り込む(`infra/terraform/env/prod` で実行)。

```
terraform import module.github_oidc.google_iam_workload_identity_pool.github projects/<PROJECT_ID>/locations/global/workloadIdentityPools/github-actions
terraform import module.github_oidc.google_iam_workload_identity_pool_provider.github projects/<PROJECT_ID>/locations/global/workloadIdentityPools/github-actions/providers/github
```

取り込んだら、`terraform plan` を実行して、**差分を最後まで読む**。残りのリソース(サービスアカウント、Artifact Registry、Scheduler など)の作り直しだけが出ていることを確かめてから、`terraform apply` を実行する。想定外の `destroy` や `replace` があれば、実行しない。

### 3. state バケットの権限を付け直す

デプロイ用サービスアカウントを作り直すと、`bootstrap/` で付けた権限が効かなくなる。記録が `deleted:serviceAccount:` になっていることを、読み取りで確かめる。

```
gcloud storage buckets get-iam-policy gs://kototech-line-notification-tfstate | grep deleted:serviceAccount
```

サービスアカウントが存在する状態(手順 2 の `apply` の後)で、`infra/terraform/bootstrap` の `state_admins` を再適用する。`terraform.tfvars` の `state_admins` に、デプロイ用サービスアカウント(`github-deployer@<PROJECT_ID>.iam.gserviceaccount.com`)が入っていることを確かめ、`terraform plan` を読んでから、`terraform apply` を実行する。

### 4. Secret を登録し直す

仮の version が `latest` になっている。**値は読まず**、version の一覧で確かめる。

```
gcloud secrets versions list line-channel-secret --project=<PROJECT_ID>
```

本物の値を、新しい version として追加する。値は、LINE Developers コンソールから取り、画面に出さず、履歴に残さずに渡す。

```
read -rs LINE_CHANNEL_SECRET
printf '%s' "$LINE_CHANNEL_SECRET" | gcloud secrets versions add line-channel-secret --project=<PROJECT_ID> --data-file=-
unset LINE_CHANNEL_SECRET
```

次の Job の実行から、`latest` の新しい値が使われる。詳しくは [../engineering/02-infra.md](../engineering/02-infra.md) の「LINE チャネルシークレットの再発行」を参照する。

### 5. デプロイを再実行する

1. 手元で `terraform plan` を実行して、差分が出ないことを確かめる
2. `apply-to-prod` を、`apply` を false にして起動し、ステップサマリの「削除を含む変更」が「なし」であることを確かめる(承認は求められるが、plan のみで `apply` しない)
3. `deploy-to-prod` を再実行する(失敗した実行の再実行か、`main` からの手動実行)。イメージは、ビルドして Artifact Registry に push し直される
4. 削除を含む計画で止まったときは、**止まった理由を確かめる**。意図したものと確かめられるまで、`allow_destroy` を `true` にしない
5. デプロイが成功して、Job が動くことを確かめてから、Scheduler を再開する(Environment の variable `SCHEDULER_PAUSED` を `false` にして、再デプロイする)

### 6. 復旧の確認

```
gcloud iam workload-identity-pools describe github-actions --project=<PROJECT_ID> --location=global --format='value(state)'
gcloud iam service-accounts describe github-deployer@<PROJECT_ID>.iam.gserviceaccount.com --format='value(email)'
gcloud secrets versions list line-channel-secret --project=<PROJECT_ID>
```

- Pool の状態が `ACTIVE`、デプロイ用サービスアカウントが存在する
- `deploy-to-prod` が、`terraform init` の `403` なしに最後まで通る
- 手元の `terraform plan` に、差分がない
