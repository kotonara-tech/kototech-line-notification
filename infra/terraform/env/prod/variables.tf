variable "project_id" {
  description = "この環境の GCP プロジェクト ID"
  type        = string
}

variable "region" {
  type    = string
  default = "asia-northeast1"
}

variable "github_repository" {
  description = "GitHub Actions からの認証を許可するリポジトリ (owner/repo)"
  type        = string
  default     = "kotonara-tech/kototech-line-notification"
}

variable "spreadsheet_id" {
  description = "この環境で読み書きする工程表のスプレッドシート ID"
  type        = string
}

variable "line_channel_id" {
  description = "この環境で使う LINE チャネルの ID(ステートレスなアクセストークンの発行に使う。秘匿情報ではない)"
  type        = string
}

variable "notify_lead_minutes" {
  description = "プログラムの開始の何分前から通知するか(アプリの環境変数 NOTIFY_LEAD_MINUTES)"
  type        = number
  default     = 10
}

variable "scheduler_paused" {
  description = "初回デプロイ前はイメージが placeholder のため、Scheduler を一時停止しておく"
  type        = bool
  default     = true
}
