output "state_bucket_name" {
  description = "env/*/backend.tf の bucket に設定する値"
  value       = google_storage_bucket.tfstate.name
}
