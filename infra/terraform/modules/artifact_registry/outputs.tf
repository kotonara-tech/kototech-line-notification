output "repository_url" {
  description = "イメージの push 先 (例: <region>-docker.pkg.dev/<project>/<name>)"
  value       = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.this.repository_id}"
}
