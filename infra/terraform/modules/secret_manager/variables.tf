variable "project_id" {
  type = string
}

variable "secret_ids" {
  description = "作成する Secret の ID"
  type        = list(string)
}

variable "placeholder_value" {
  description = "仮の version の値。本物の値は gcloud で別の version として追加する"
  type        = string
  default     = "CHANGE_ME"
}
