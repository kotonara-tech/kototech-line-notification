# 変数の値は validation で検査する。GitHub の変数の先頭・末尾に空白が混入すると、
# 別のプロジェクトと見なされてリソースが作り直される計画になるため、plan の前に止める。
# (Terraform がエラーに `var.xxx is "..."` と値を引用符つきで表示するので、空白の有無はそこで分かる。メッセージには値を含めない)

variable "project_id" {
  description = "この環境の GCP プロジェクト ID"
  type        = string

  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{4,28}[a-z0-9]$", var.project_id))
    error_message = "project_id の形式が不正です。小文字で始まる 6〜30 文字の小文字・数字・ハイフンで、ハイフンで終わらず、前後に空白や改行を含みません。GitHub の Environment 変数 GCP_PROJECT_ID の値を確認してください(先頭・末尾の空白の混入が疑われます)。"
  }
}

variable "region" {
  type    = string
  default = "asia-northeast1"

  validation {
    condition     = can(regex("^[a-z]+-[a-z]+[0-9]+$", var.region))
    error_message = "region の形式が不正です。asia-northeast1 のような値で、前後に空白や改行を含みません。GitHub の Environment 変数 REGION の値を確認してください(先頭・末尾の空白の混入が疑われます。未設定なら asia-northeast1 になります)。"
  }
}

variable "github_repository" {
  description = "GitHub Actions からの認証を許可するリポジトリ (owner/repo)"
  type        = string
  default     = "kotonara-tech/kototech-line-notification"

  validation {
    condition     = can(regex("^[A-Za-z0-9._-]+/[A-Za-z0-9._-]+$", var.github_repository))
    error_message = "github_repository の形式が不正です。owner/repo の形で、前後に空白や改行を含みません。CI は既定値を使うため、tfvars か環境変数 TF_VAR_github_repository で上書きしていないか確認してください。"
  }
}

variable "spreadsheet_id" {
  description = "この環境で読み書きする工程表のスプレッドシート ID"
  type        = string

  validation {
    condition     = can(regex("^[A-Za-z0-9_-]+$", var.spreadsheet_id))
    error_message = "spreadsheet_id の形式が不正です。英数字・ハイフン・アンダースコアだけで、空ではなく、前後に空白や改行を含みません。GitHub の Environment 変数 SPREADSHEET_ID の値を確認してください(先頭・末尾の空白の混入が疑われます)。"
  }
}

variable "line_channel_id" {
  description = "この環境で使う LINE チャネルの ID(ステートレスなアクセストークンの発行に使う。秘匿情報ではない)"
  type        = string

  validation {
    condition     = can(regex("^[0-9]+$", var.line_channel_id))
    error_message = "line_channel_id の形式が不正です。数字だけで、空ではなく、前後に空白や改行を含みません。GitHub の Environment 変数 LINE_CHANNEL_ID の値を確認してください(先頭・末尾の空白の混入が疑われます)。"
  }
}

variable "notify_lead_minutes" {
  description = "プログラムの開始の何分前から通知するか(アプリの環境変数 NOTIFY_LEAD_MINUTES)"
  type        = number
  default     = 10

  validation {
    # アプリ(backend/internal/config/config.go)の検証と合わせる: 1〜1440 の整数
    condition     = var.notify_lead_minutes >= 1 && var.notify_lead_minutes <= 1440 && floor(var.notify_lead_minutes) == var.notify_lead_minutes
    error_message = "notify_lead_minutes が不正です。1〜1440 の整数にしてください。GitHub の Environment 変数 NOTIFY_LEAD_MINUTES の値を確認してください(未設定なら 10 になります)。"
  }
}

variable "scheduler_paused" {
  description = "初回デプロイ前はイメージが placeholder のため、Scheduler を一時停止しておく"
  type        = bool
  default     = true
}
