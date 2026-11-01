variable "project_id" {
  type = string
}

variable "github_repository" {
  description = "認証を許可するリポジトリ (owner/repo)"
  type        = string
}

variable "allowed_ref" {
  description = "認証を許可する ref (例: refs/heads/main)。空なら制限しない"
  type        = string
  default     = ""
}

variable "pool_id" {
  description = "Workload Identity Pool の ID"
  type        = string
  default     = "github-actions"
}

variable "deployer_roles" {
  description = "アプリのデプロイ(イメージの push と Cloud Run Job の更新)に必要なプロジェクトレベルのロール"
  type        = list(string)
  default = [
    "roles/run.admin",
    "roles/artifactregistry.writer",
    "roles/iam.serviceAccountUser",
  ]
}

# GitHub Actions から Terraform(plan / apply)を実行するために必要なロール。
# このモジュール自身(WIF・サービスアカウント・プロジェクトの IAM)も管理対象に含むため、強い権限になる。
# 権限の乱用を避けるため、prod は allowed_ref で main のみに限定すること。Environment の承認は、開発の立ち上げ期は挟まない(D36)。
# 承認なしで prod を守る仕組みは、docs/engineering/02-infra.md の「prod の保護(承認なし)」。
variable "terraform_roles" {
  description = "Terraform の実行に必要なプロジェクトレベルのロール"
  type        = list(string)
  default = [
    "roles/serviceusage.serviceUsageAdmin",  # API の有効化(google_project_service)
    "roles/iam.workloadIdentityPoolAdmin",   # Workload Identity Pool / Provider
    "roles/iam.serviceAccountAdmin",         # サービスアカウントとその IAM
    "roles/resourcemanager.projectIamAdmin", # プロジェクトの IAM バインディング
    "roles/artifactregistry.admin",          # Artifact Registry リポジトリ
    "roles/secretmanager.admin",             # Secret とその IAM
    "roles/cloudscheduler.admin",            # Cloud Scheduler ジョブ
  ]
}
