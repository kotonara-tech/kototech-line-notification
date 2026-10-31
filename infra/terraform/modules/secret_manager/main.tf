# Secret の入れ物と、仮の version だけを作る。本物の値は Terraform の state に平文で
# 残さないよう、gcloud などで別途 version として追加する(latest が最新の version を指す)。
#
# 仮の version を作る理由: Secret に version が 1 つもないと、その Secret を参照する
# Cloud Run Job を作成できないため(入れ物と Job を同じ apply で作れるようにする)。

resource "google_secret_manager_secret" "this" {
  for_each = toset(var.secret_ids)

  project   = var.project_id
  secret_id = each.value

  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "placeholder" {
  for_each = google_secret_manager_secret.this

  secret      = each.value.id
  secret_data = var.placeholder_value

  # 値は Terraform の管理外。仮の値の変更でバージョンが作り直されないようにする
  lifecycle {
    ignore_changes = [secret_data]
  }
}
