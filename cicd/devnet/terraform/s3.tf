# Bucket need to be created first. If first time run terraform init, need to comment out the below section
terraform {
  backend "s3" {
    bucket = "tf-devnet-bucket" // This name need to be updated to be the same as local.s3BucketName. We can't use variable here.
    key    = "tf/terraform_new.tfstate"
    region = "us-east-1"
    encrypt = true
  }
}

data "aws_s3_object" "devnet_xdc_node_config" {
  bucket = local.s3BucketName
  key    = "node-config.json"
}
