resource "aws_s3_bucket" "bucket" {
  bucket = "pon-blog-media"
  acl    = "public-read"

  cors_rule {
    allowed_headers = ["*"]
    allowed_methods = ["GET"]
    allowed_origins = [var.allowed_origins]
    max_age_seconds = 3000
  }

  tags = {
    Name = var.app_name
  }
}
