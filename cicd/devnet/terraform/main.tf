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

resource "aws_vpc" "devnet_vpc" {
  cidr_block = "10.0.0.0/16"
  instance_tenancy = "default"
  enable_dns_hostnames = true
  
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

resource "aws_default_security_group" "devnet_xdcnode_security_group" {
  vpc_id = aws_vpc.devnet_vpc.id

  ingress {
    from_port   = 0
    to_port     = 0
    protocol    = -1
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
  tags = {
    Name = "TfDevnetNode"
  }
}

# IAM policies
data "aws_iam_policy_document" "xdc_ecs_tasks_execution_role" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["ecs-tasks.amazonaws.com"]
    }
  }
}

# Create the role
resource "aws_iam_role" "devnet_xdc_ecs_tasks_execution_role" {
  name               = "devnet-xdc-ecs-task-execution-role"
  assume_role_policy = "${data.aws_iam_policy_document.xdc_ecs_tasks_execution_role.json}"
}

# Attached the AWS managed policies to the new role
resource "aws_iam_role_policy_attachment" "devnet_xdc_ecs_tasks_execution_role" {
  for_each = toset([
    "arn:aws:iam::aws:policy/AmazonElasticFileSystemClientFullAccess", 
    "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy",
    "arn:aws:iam::aws:policy/AmazonElasticFileSystemsUtils"
  ])
  role       = aws_iam_role.devnet_xdc_ecs_tasks_execution_role.name
  policy_arn = each.value
}

# Logs
resource "aws_cloudwatch_log_group" "devnet_cloud_watch_group" {
  for_each = local.devnetNodeKyes

  name = "tf-${each.key}"
  retention_in_days = 14 # Logs are only kept for 14 days
  tags = {
    Name = "TfDevnetCloudWatchGroup${each.key}"
  }
}