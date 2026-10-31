terraform {
  backend "gcs" {
    # bootstrap/ で作成したバケット(変数は使えないため直書き)
    bucket = "kototech-line-notification-tfstate"
    prefix = "env/dev"
  }
}
