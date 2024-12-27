terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
    }
  }
}

# Default
provider "aws" {
  region  = "us-east-1"
}

provider "aws" {
  alias = "us-east-2"
  region  = "us-east-2"
}

module "us-east-2" {
  source = "./module/region"
  region = "us-east-2"
  devnetNodeKeys = local.devnetNodeKeys["us-east-2"]
  logLevel = local.logLevel
  devnet_xdc_ecs_tasks_execution_role_arn = aws_iam_role.devnet_xdc_ecs_tasks_execution_role.arn
  docker_tag = var.docker_tag
  providers = {
    aws = aws.us-east-2
  }
}

provider "aws" {
  alias = "eu-west-1"
  region  = "eu-west-1"
}

module "eu-west-1" {
  source = "./module/region"
  region = "eu-west-1"
  devnetNodeKeys = local.devnetNodeKeys["eu-west-1"]
  logLevel = local.logLevel
  devnet_xdc_ecs_tasks_execution_role_arn = aws_iam_role.devnet_xdc_ecs_tasks_execution_role.arn
  docker_tag = var.docker_tag
  providers = {
    aws = aws.eu-west-1
  }
}

provider "aws" {
  alias = "ap-southeast-2"
  region  = "ap-southeast-2"
}

module "ap-southeast-2" {
  source = "./module/region"
  region = "ap-southeast-2"
  devnetNodeKeys = local.devnetNodeKeys["ap-southeast-2"]
  logLevel = local.logLevel
  devnet_xdc_ecs_tasks_execution_role_arn = aws_iam_role.devnet_xdc_ecs_tasks_execution_role.arn
  docker_tag = var.docker_tag
  providers = {
    aws = aws.ap-southeast-2
  }
}
