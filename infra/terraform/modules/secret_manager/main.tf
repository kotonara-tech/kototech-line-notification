# Secret の入れ物と、(必要なときだけ)仮の version を作る。本物の値は Terraform の state に
# 平文で残さないよう、gcloud などで別途 version として追加する(latest が最新の version を指す)。
#
# 仮の version を作る理由: Secret に version が 1 つもないと、その Secret を参照する
# Cloud Run Job を作成できないため(入れ物と Job を同じ apply で作れるようにする)。
#
# 仮の version は、初回の apply のためだけに必要。本物の値を登録したあとも Terraform が
# 管理し続けると、リソースの作り直し(destroy → create)で「CHANGE_ME」が新しい version
# として作られて latest になり、本物の値を上書きしてしまう。そのため、本物の値を登録した
# Secret は placeholder_secret_ids から外して、仮の version をコードから外す。

locals {
  # 未指定(null)なら、すべての Secret に仮の version を作る(新しい環境の初回の apply)
  placeholder_secret_ids = var.placeholder_secret_ids == null ? var.secret_ids : var.placeholder_secret_ids
}

resource "google_secret_manager_secret" "this" {
  for_each = toset(var.secret_ids)

  project   = var.project_id
  secret_id = each.value

  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "placeholder" {
  # キー(= Secret ID)は従来と同じ。リストから外した Secret の仮の version は、破棄されるだけで、作り直されない
  for_each = toset(local.placeholder_secret_ids)

  secret      = google_secret_manager_secret.this[each.key].id
  secret_data = var.placeholder_value

  # 値は Terraform の管理外。仮の値の変更でバージョンが作り直されないようにする
  lifecycle {
    ignore_changes = [secret_data]
  }
}
