variable "project_id" {
  type = string
}

variable "region" {
  type = string
}

variable "name" {
  description = "Cloud Run Job 名"
  type        = string
}

variable "image" {
  description = "初期イメージ。以降は CD が更新する"
  type        = string
  default     = "us-docker.pkg.dev/cloudrun/container/hello"
}

variable "env_vars" {
  description = "平文の環境変数(ROLE など、役割を切り替えるものを含む)"
  type        = map(string)
  default     = {}
}

variable "secret_env_vars" {
  description = "Secret Manager から注入する環境変数(環境変数名 => Secret ID)"
  type        = map(string)
  default     = {}
}

variable "timeout" {
  description = "1 回の実行のタイムアウト"
  type        = string
  default     = "300s"
}

variable "max_retries" {
  description = "失敗時の再試行回数。定期実行なので既定は 0(次回の実行で拾う)"
  type        = number
  default     = 0
}

variable "deletion_protection" {
  type    = bool
  default = true
}

variable "token_creator_members" {
  description = "実行用サービスアカウントのトークンを発行(借用)できる主体(例: serviceAccount:...)。既定は空"
  type        = list(string)
  default     = []
}
