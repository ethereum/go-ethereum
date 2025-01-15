module "s3_bucket" {
  source  = "terraform-aws-modules/s3-bucket/aws"
  version = "4.4.0"

  bucket = "limechain-devops-task-tf-state"
  acl    = "private"

  control_object_ownership = true
  object_ownership         = "ObjectWriter"

  versioning = {
    enabled = true
  }
}

resource "aws_dynamodb_table" "terraform_lock_table" {
  name         = "terraform-lock-table"
  billing_mode = "PAY_PER_REQUEST"

  hash_key                    = "LockID"
  deletion_protection_enabled = true

  attribute {
    name = "LockID"
    type = "S"
  }
}
