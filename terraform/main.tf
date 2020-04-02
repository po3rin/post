provider "aws" {
  region = "ap-northeast-1"
}

terraform {
  required_version = ">= 0.12"

  backend "s3" {
    bucket = "po3rin-tfstate"
    region = "ap-northeast-1"
    key    = "post/terraform.tfstate"
  }
}

module "s3" {
  source   = "./modules/s3"
  app_name = var.app_name
  allowed_origins = var.allowed_origins
}
