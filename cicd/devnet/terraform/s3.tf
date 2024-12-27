terraform {
  backend "s3" {
    bucket = "tf-xinfin-bucket" // This name need to be updated to be the same as local.s3BucketName. We can't use variable here.
    key    = "tf/terraform_devnet.tfstate"
    region = "us-east-1"
    encrypt = true
  }
}

data "aws_s3_object" "devnet_xdc_node_config" {
  bucket = "tf-xinfin-bucket"
  key    = "node-config.json"
}
