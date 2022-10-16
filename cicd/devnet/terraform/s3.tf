

# This bucket had to be created before you can run the terraform init
resource "aws_s3_bucket" "terraform_s3_bucket" {
  bucket = "terraform-devnet-bucket"
  versioning {
    enabled = true
  }
}

# Bucket need to be created first. If first time run terraform init, need to comment out the below section
terraform {
  backend "s3" {
    bucket = "terraform-devnet-bucket"
    key    = "tf/terraform.tfstate"
    region = "us-east-1"
    encrypt = true
  }
}

data "aws_s3_bucket_object" "devnet_xdc_node_config" {
  bucket = "terraform-devnet-bucket"
  key    = "node-config.json"
}