variable "project_id" {
  type = string
}

variable "secret_ids" {
  description = "作成する Secret の ID"
  type        = list(string)
}

variable "placeholder_secret_ids" {
  description = "仮の version を作る Secret の ID(secret_ids の一部)。null(既定)なら、すべての Secret。初回の apply(version がないと Cloud Run Job を作れない)のためだけに必要で、本物の値を登録した Secret は、ここから外す(空リストなら、1 つも作らない)。残したままだと、リソースの作り直しで仮の値が新しい latest になり、本物の値を上書きする"
  type        = list(string)
  default     = null
}

variable "placeholder_value" {
  description = "仮の version の値。本物の値は gcloud で別の version として追加する"
  type        = string
  default     = "CHANGE_ME"
}
