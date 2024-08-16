terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.13.1"
    }
  }
}

# Default
provider "aws" {
  region  = "us-east-1"
}

# WARNING: APSE-1 will only be used to host rpc node
# Workaround to avoid conflicts with existing ecs cluster in existing regions
provider "aws" {
  alias = "ap-southeast-1"
  region  = "ap-southeast-1"
}

module "devnet_rpc" {
  source = "./module/ec2_rpc"
  network = "devnet"
  vpc_id = local.vpc_id
  aws_subnet_id = local.aws_subnet_id
  ami_id = local.ami_id
  instance_type = "t3.xlarge"
  ssh_key_name = local.ssh_key_name
  rpc_image = local.rpc_image 
  volume_size = 1500

  providers = {
    aws = aws.ap-southeast-1
  }
}

module "testnet_rpc" {
  source = "./module/ec2_rpc"
  network = "testnet"
  vpc_id = local.vpc_id
  aws_subnet_id = local.aws_subnet_id
  ami_id = local.ami_id
  instance_type = "t3.large"
  ssh_key_name = local.ssh_key_name
  rpc_image = local.rpc_image 
  volume_size = 1500

  providers = {
    aws = aws.ap-southeast-1
  }
}

module "mainnet_rpc" {
  source = "./module/ec2_rpc"
  network = "mainnet"
  vpc_id = local.vpc_id
  aws_subnet_id = local.aws_subnet_id
  ami_id = local.ami_id
  instance_type = "t3.large"
  ssh_key_name = local.ssh_key_name
  rpc_image = local.rpc_image 
  volume_size = 3000

  providers = {
    aws = aws.ap-southeast-1
  }
}