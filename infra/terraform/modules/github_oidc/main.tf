# GitHub Actions から鍵なしで GCP に認証するための Workload Identity Federation

locals {
  repository_condition = "assertion.repository == \"${var.github_repository}\""
  ref_condition        = var.allowed_ref == "" ? "" : " && assertion.ref == \"${var.allowed_ref}\""
}

resource "google_iam_workload_identity_pool" "github" {
  project                   = var.project_id
  workload_identity_pool_id = var.pool_id
  display_name              = "GitHub Actions"
}

resource "google_iam_workload_identity_pool_provider" "github" {
  project                            = var.project_id
  workload_identity_pool_id          = google_iam_workload_identity_pool.github.workload_identity_pool_id
  workload_identity_pool_provider_id = "github"
  display_name                       = "GitHub"

  attribute_mapping = {
    "google.subject"             = "assertion.sub"
    "attribute.repository"       = "assertion.repository"
    "attribute.repository_owner" = "assertion.repository_owner"
    "attribute.ref"              = "assertion.ref"
  }

  # 対象リポジトリ(必要ならブランチも)以外のトークンは Provider 段階で拒否する
  attribute_condition = "${local.repository_condition}${local.ref_condition}"

  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }
}

resource "google_service_account" "deployer" {
  project      = var.project_id
  account_id   = "github-deployer"
  display_name = "GitHub Actions deployer"
}

# 対象リポジトリの Actions にだけ、このサービスアカウントへのなりすましを許可する
resource "google_service_account_iam_member" "deployer_wif" {
  service_account_id = google_service_account.deployer.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.github.name}/attribute.repository/${var.github_repository}"
}

locals {
  project_roles = toset(concat(var.deployer_roles, var.terraform_roles))
}

resource "google_project_iam_member" "deployer" {
  for_each = local.project_roles

  project = var.project_id
  role    = each.value
  member  = "serviceAccount:${google_service_account.deployer.email}"
}
