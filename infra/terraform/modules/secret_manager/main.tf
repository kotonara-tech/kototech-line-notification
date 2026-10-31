# Secret の「入れ物」だけを作る。値(version)は Terraform の state に残さないよう、
# gcloud などで別途登録する。

resource "google_secret_manager_secret" "this" {
  for_each = toset(var.secret_ids)

  project   = var.project_id
  secret_id = each.value

  replication {
    auto {}
  }
}
