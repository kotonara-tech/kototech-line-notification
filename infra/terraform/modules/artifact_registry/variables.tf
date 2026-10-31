variable "project_id" {
  type = string
}

variable "region" {
  type = string
}

variable "name" {
  description = "リポジトリ名"
  type        = string
}

variable "description" {
  type    = string
  default = ""
}

variable "keep_count" {
  description = "残すイメージの世代数"
  type        = number
  default     = 10
}
