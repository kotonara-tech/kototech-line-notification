variable "project_id" {
  type = string
}

variable "secret_ids" {
  description = "作成する Secret の ID"
  type        = list(string)
}
