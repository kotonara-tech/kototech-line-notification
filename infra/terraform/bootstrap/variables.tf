# 変数の値は validation で検査する。値の先頭・末尾に空白が混入すると、別のリソースと見なされて
# 作り直しの計画になる(state バケットでは、置き場所や名前の変更が破壊的になる)ため、plan の前に止める。
# bootstrap は手元で実行するため、直し方は tfvars(または -var / TF_VAR_*)の値の確認になる。

variable "project_id" {
  description = "ops プロジェクトの ID(state バケットを置く)"
  type        = string

  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{4,28}[a-z0-9]$", var.project_id))
    error_message = "project_id の形式が不正です。小文字で始まる 6〜30 文字の小文字・数字・ハイフンで、ハイフンで終わらず、前後に空白や改行を含みません。terraform.tfvars の project_id(または TF_VAR_project_id)の値を確認してください(先頭・末尾の空白の混入が疑われます)。"
  }
}

variable "region" {
  description = "バケットのロケーション"
  type        = string
  default     = "asia-northeast1"

  validation {
    condition     = can(regex("^[a-z]+-[a-z]+[0-9]+$", var.region))
    error_message = "region の形式が不正です。asia-northeast1 のような値で、前後に空白や改行を含みません。terraform.tfvars の region(または TF_VAR_region)の値を確認してください(先頭・末尾の空白の混入が疑われます)。"
  }
}

variable "state_bucket_name" {
  description = "state 用バケット名(グローバルで一意。env/*/backend.tf の bucket と揃える)"
  type        = string
  default     = "kototech-line-notification-tfstate"

  validation {
    condition     = can(regex("^[a-z0-9][a-z0-9._-]{1,61}[a-z0-9]$", var.state_bucket_name))
    error_message = "state_bucket_name の形式が不正です。3〜63 文字の小文字・数字・ハイフン・アンダースコア・ドットで、英数字で始まり英数字で終わり、前後に空白や改行を含みません。terraform.tfvars の state_bucket_name(または TF_VAR_state_bucket_name)の値を確認してください(env/*/backend.tf の bucket とも揃えます)。"
  }
}

variable "state_admins" {
  description = "state バケットへの読み書きを許可する主体(例: user:you@example.com, serviceAccount:...)"
  type        = list(string)
  default     = []

  validation {
    condition     = alltrue([for m in var.state_admins : m != "" && m == trimspace(m)])
    error_message = "state_admins に、空の要素、または前後に空白や改行を含む要素があります。terraform.tfvars の state_admins の各要素(user:... や serviceAccount:...)を確認してください。"
  }
}
