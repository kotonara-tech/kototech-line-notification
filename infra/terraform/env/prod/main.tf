locals {
  env = "prod"
}

module "apis" {
  source = "../../modules/apis"

  project_id = var.project_id
}

module "github_oidc" {
  source = "../../modules/github_oidc"

  project_id        = var.project_id
  github_repository = var.github_repository
  allowed_ref       = "refs/heads/main"

  depends_on = [module.apis]
}

module "artifact_registry" {
  source = "../../modules/artifact_registry"

  project_id  = var.project_id
  region      = var.region
  name        = "line-notifier"
  description = "line-notifier のコンテナイメージ"

  depends_on = [module.apis]
}

module "secret_manager" {
  source = "../../modules/secret_manager"

  project_id = var.project_id
  secret_ids = ["line-channel-secret"]

  depends_on = [module.apis]
}

module "cloud_run_job_scan" {
  source = "../../modules/cloud_run_job"

  project_id          = var.project_id
  region              = var.region
  name                = "line-notifier-scan"
  deletion_protection = true

  # 同一バイナリを環境変数で切り替える。ROLE がこのプロセスの役割を決める
  env_vars = {
    APP_ENV             = local.env
    ROLE                = "scan"
    SPREADSHEET_ID      = var.spreadsheet_id
    LINE_CHANNEL_ID     = var.line_channel_id
    NOTIFY_LEAD_MINUTES = tostring(var.notify_lead_minutes)
  }

  # アクセストークンは保管しない。アプリが LINE_CHANNEL_ID と LINE_CHANNEL_SECRET から、
  # 実行のたびにステートレスなチャネルアクセストークンを発行する
  secret_env_vars = {
    LINE_CHANNEL_SECRET = module.secret_manager.secret_ids["line-channel-secret"]
  }

  depends_on = [module.apis]
}

module "cloud_scheduler_scan" {
  source = "../../modules/cloud_scheduler"

  project_id   = var.project_id
  region       = var.region
  name         = "line-notifier-scan"
  description  = "工程表をスキャンして開始間近のプログラムを通知する"
  job_name     = module.cloud_run_job_scan.name
  job_location = module.cloud_run_job_scan.location
  paused       = var.scheduler_paused

  depends_on = [module.apis]
}
