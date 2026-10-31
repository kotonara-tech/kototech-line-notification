# Cloud Run Jobs API の :run を呼んでジョブを起動する。
# Job にはエンドポイントがないため、Cloud Run Admin API を OAuth で呼ぶ。

resource "google_service_account" "scheduler" {
  project      = var.project_id
  account_id   = "${var.name}-scheduler"
  display_name = "${var.name} Cloud Scheduler"
}

# Scheduler 用サービスアカウントにだけ、対象の Job を実行する権限を付与する
resource "google_cloud_run_v2_job_iam_member" "invoker" {
  project  = var.project_id
  location = var.job_location
  name     = var.job_name
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_service_account.scheduler.email}"
}

resource "google_cloud_scheduler_job" "this" {
  project     = var.project_id
  region      = var.region
  name        = var.name
  description = var.description
  schedule    = var.schedule
  time_zone   = var.time_zone
  paused      = var.paused

  # 毎分の定期実行なので、失敗時の再試行はしない(次回の実行で拾う)。
  # retry_config を書かない場合の既定が「再試行なし」。retry_count = 0 と書くと、
  # API が値を返さないため plan のたびに差分が出る。

  attempt_deadline = var.attempt_deadline

  http_target {
    http_method = "POST"
    uri         = "https://run.googleapis.com/v2/projects/${var.project_id}/locations/${var.job_location}/jobs/${var.job_name}:run"

    oauth_token {
      service_account_email = google_service_account.scheduler.email
      scope                 = "https://www.googleapis.com/auth/cloud-platform"
    }
  }
}
