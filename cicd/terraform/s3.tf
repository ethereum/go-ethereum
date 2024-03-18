# Bucket need to be created first. If first time run terraform init, need to comment out the below section
terraform {
  backend "s3" {
    bucket = "tf-xinfin-bucket"
    key    = "tf/terraform_rpc.tfstate"
    region = "us-east-1"
    encrypt = true
  }
}

data "aws_s3_object" "xdc_node_config" {
  bucket = "tf-xinfin-bucket"
  key    = "node-config.json"
}
