# HTTP エンドポイントを持たない、実行して終了するだけの Cloud Run Job。
# どの役割で動くかは環境変数(ROLE)で切り替え、イメージ(Go バイナリ)は共通とする。

resource "google_service_account" "runtime" {
  project      = var.project_id
  account_id   = "${var.name}-runtime"
  display_name = "${var.name} Cloud Run Job runtime"
}

resource "google_cloud_run_v2_job" "this" {
  project  = var.project_id
  name     = var.name
  location = var.region

  deletion_protection = var.deletion_protection

  template {
    # 1 回の実行で 1 タスクのみ
    parallelism = 1
    task_count  = 1

    template {
      service_account = google_service_account.runtime.email
      timeout         = var.timeout
      max_retries     = var.max_retries

      containers {
        image = var.image

        dynamic "env" {
          for_each = var.env_vars
          content {
            name  = env.key
            value = env.value
          }
        }

        dynamic "env" {
          for_each = var.secret_env_vars
          content {
            name = env.key
            value_source {
              secret_key_ref {
                secret  = env.value
                version = "latest"
              }
            }
          }
        }
      }
    }
  }

  # Job は作成時に、実行用サービスアカウントが参照する Secret を読めるかを検証する。
  # 参照だけでは依存が表せないため、権限の付与を先に済ませる(同時に作ると失敗することがある)
  depends_on = [google_secret_manager_secret_iam_member.runtime_accessor]

  # イメージは GitHub Actions のデプロイで更新するため、Terraform では差分を無視する
  lifecycle {
    ignore_changes = [
      template[0].template[0].containers[0].image,
      client,
      client_version,
    ]
  }
}

# 指定した主体が、実行用サービスアカウントのトークンを発行(借用)できるようにする。
# CI のインテグレーションテストが、本番の Job と同じ主体として工程表を読めるかを確認するために使う。
# 工程表や Secret へのアクセスを広げることになるため、必要な環境(dev)でだけ指定する。
resource "google_service_account_iam_member" "token_creator" {
  for_each = toset(var.token_creator_members)

  service_account_id = google_service_account.runtime.name
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = each.value
}

# 実行用サービスアカウントに、参照する Secret の読み取り権限を付与する
resource "google_secret_manager_secret_iam_member" "runtime_accessor" {
  for_each = toset(values(var.secret_env_vars))

  project   = var.project_id
  secret_id = each.value
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.runtime.email}"
}
