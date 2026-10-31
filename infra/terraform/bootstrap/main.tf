# ops プロジェクトに、dev / prod 共通の Terraform state 用 GCS バケットを作る。
# 最初の 1 回だけ実行する。
# このディレクトリ自身の state はローカルに置く(.gitignore 済み)。

terraform {
  required_version = ">= 1.6"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

resource "google_project_service" "storage" {
  service            = "storage.googleapis.com"
  disable_on_destroy = false
}

resource "google_storage_bucket" "tfstate" {
  name     = var.state_bucket_name
  location = var.region

  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"
  force_destroy               = false

  versioning {
    enabled = true
  }

  # 世代が増え続けないよう、古い世代は新しい 10 世代を残して削除する
  lifecycle_rule {
    condition {
      num_newer_versions = 10
      with_state         = "ARCHIVED"
    }
    action {
      type = "Delete"
    }
  }

  depends_on = [google_project_service.storage]
}

# state を読み書きする主体(Terraform を実行する人・CI のサービスアカウント)に権限を付与する
resource "google_storage_bucket_iam_member" "state_admins" {
  for_each = toset(var.state_admins)

  bucket = google_storage_bucket.tfstate.name
  role   = "roles/storage.objectAdmin"
  member = each.value
}
