output "workload_identity_provider" {
  description = "google-github-actions/auth の workload_identity_provider に設定する値"
  value       = google_iam_workload_identity_pool_provider.github.name
}

output "deployer_service_account" {
  description = "google-github-actions/auth の service_account に設定する値"
  value       = google_service_account.deployer.email
}
