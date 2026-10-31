variable "project_id" {
  type = string
}

variable "apis" {
  description = "有効化する API"
  type        = list(string)
  default = [
    "iam.googleapis.com",
    "iamcredentials.googleapis.com",
    "sts.googleapis.com",
    "cloudresourcemanager.googleapis.com",
    "run.googleapis.com",
    "artifactregistry.googleapis.com",
    "cloudscheduler.googleapis.com",
    "secretmanager.googleapis.com",
    "sheets.googleapis.com",
  ]
}
