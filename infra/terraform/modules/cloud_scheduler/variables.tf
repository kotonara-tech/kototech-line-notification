variable "project_id" {
  type = string
}

variable "region" {
  description = "Scheduler ジョブのリージョン"
  type        = string
}

variable "name" {
  type = string
}

variable "description" {
  type    = string
  default = ""
}

variable "schedule" {
  description = "cron 式。既定は毎分"
  type        = string
  default     = "* * * * *"
}

variable "time_zone" {
  type    = string
  default = "Asia/Tokyo"
}

variable "attempt_deadline" {
  type    = string
  default = "60s"
}

variable "paused" {
  description = "true なら一時停止する(初回デプロイ前など)"
  type        = bool
  default     = false
}

variable "job_name" {
  description = "起動する Cloud Run Job 名"
  type        = string
}

variable "job_location" {
  description = "Cloud Run Job のリージョン"
  type        = string
}
