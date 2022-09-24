terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.16"
    }
  }

  required_version = ">= 1.2.0"
}

provider "aws" {
  region  = "us-east-1"
}

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

resource "aws_vpc" "devnet_vpc" {
  cidr_block = "10.0.0.0/16"
  instance_tenancy = "default"
  
  tags = {
    Name = "TfDevnetVpc"
  }
}

resource "aws_subnet" "devnet_subnet" {
  vpc_id = aws_vpc.devnet_vpc.id
  cidr_block = "10.0.0.0/20"
  map_public_ip_on_launch = true
  availability_zone = "us-east-1a"
  
  tags = {
    Name = "TfDevnetVpcSubnet"
  }
}

resource "aws_internet_gateway" "devnet_gatewat" {
  vpc_id = aws_vpc.devnet_vpc.id

  tags = {
    Name = "TfDevnetGateway"
  }
}

resource "aws_route_table" "devnet_route_table" {
  vpc_id = aws_vpc.devnet_vpc.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.devnet_gatewat.id
  }

  tags = {
    Name = "TfDevnetVpcRoutingTable"
  }
}

resource "aws_route_table_association" "devnet_route_table_association" {
  subnet_id      = aws_subnet.devnet_subnet.id
  route_table_id = aws_route_table.devnet_route_table.id
}