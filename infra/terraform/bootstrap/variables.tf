variable "project_id" {
  description = "ops プロジェクトの ID(state バケットを置く)"
  type        = string
}

variable "region" {
  description = "バケットのロケーション"
  type        = string
  default     = "asia-northeast1"
}

variable "state_bucket_name" {
  description = "state 用バケット名(グローバルで一意。env/*/backend.tf の bucket と揃える)"
  type        = string
  default     = "kototech-line-notification-tfstate"
}

variable "state_admins" {
  description = "state バケットへの読み書きを許可する主体(例: user:you@example.com, serviceAccount:...)"
  type        = list(string)
  default     = []
}
