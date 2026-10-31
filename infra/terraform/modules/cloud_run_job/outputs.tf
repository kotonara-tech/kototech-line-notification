output "name" {
  value = google_cloud_run_v2_job.this.name
}

output "location" {
  value = google_cloud_run_v2_job.this.location
}

output "runtime_service_account" {
  description = "スプレッドシートの共有先に追加するメールアドレス"
  value       = google_service_account.runtime.email
}
