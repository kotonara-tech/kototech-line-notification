locals {
  env = "dev"
}

module "apis" {
  source = "../../modules/apis"

  project_id = var.project_id
}

module "github_oidc" {
  source = "../../modules/github_oidc"

  project_id        = var.project_id
  github_repository = var.github_repository
  allowed_ref       = ""

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
  secret_ids = ["line-channel-access-token", "line-channel-secret"]

  depends_on = [module.apis]
}

module "cloud_run_job_scan" {
  source = "../../modules/cloud_run_job"

  project_id          = var.project_id
  region              = var.region
  name                = "line-notifier-scan"
  deletion_protection = false

  # 同一バイナリを環境変数で切り替える。ROLE がこのプロセスの役割を決める
  env_vars = {
    APP_ENV        = local.env
    ROLE           = "scan"
    SPREADSHEET_ID = var.spreadsheet_id
  }

  # scan が使う Secret のみ注入する(最小権限)。line-channel-secret は Webhook を受ける ROLE=api で注入する
  secret_env_vars = {
    LINE_CHANNEL_ACCESS_TOKEN = module.secret_manager.secret_ids["line-channel-access-token"]
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
