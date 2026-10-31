output "workload_identity_provider" {
  description = "GitHub Actions: workload_identity_provider"
  value       = module.github_oidc.workload_identity_provider
}

output "deployer_service_account" {
  description = "GitHub Actions: service_account"
  value       = module.github_oidc.deployer_service_account
}

output "cloud_run_job_scan" {
  description = "デプロイ対象の Cloud Run Job 名"
  value       = module.cloud_run_job_scan.name
}

output "runtime_service_account" {
  description = "工程表スプレッドシートの共有先(編集者)に追加するメールアドレス"
  value       = module.cloud_run_job_scan.runtime_service_account
}

output "artifact_registry_repository_url" {
  description = "イメージの push 先"
  value       = module.artifact_registry.repository_url
}
