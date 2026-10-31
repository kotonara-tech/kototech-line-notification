output "secret_ids" {
  description = "Secret ID のマップ(キーは入力した ID)"
  value       = { for k, s in google_secret_manager_secret.this : k => s.secret_id }
}
