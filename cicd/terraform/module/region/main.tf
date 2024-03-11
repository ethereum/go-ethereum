terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.13.1"
    }
  }
}

resource "aws_vpc" "vpc" {
  cidr_block = var.vpc_cidr
  instance_tenancy = "default"
  enable_dns_hostnames = true

  tags = {
    Name = "Tf${var.network}Vpc"
  }
}

resource "aws_subnet" "subnet" {
  vpc_id = aws_vpc.vpc.id
  cidr_block = var.subnet_cidr
  map_public_ip_on_launch = true

  tags = {
    Name = "Tf${var.network}VpcSubnet"
  }
}

resource "aws_internet_gateway" "gatewat" {
  vpc_id = aws_vpc.vpc.id

  tags = {
    Name = "Tf${var.network}Gateway"
  }
}

resource "aws_route_table" "route_table" {
  vpc_id = aws_vpc.vpc.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.gatewat.id
  }

  tags = {
    Name = "Tf${var.network}VpcRoutingTable"
  }
}

resource "aws_route_table_association" "route_table_association" {
  subnet_id      = aws_subnet.subnet.id
  route_table_id = aws_route_table.route_table.id
}

resource "aws_default_security_group" "xdcnode_security_group" {
  vpc_id = aws_vpc.vpc.id

  ingress {
    description = "listener port"
    from_port   = 30303
    to_port     = 30303
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    description = "discovery port"
    from_port   = 30303
    to_port     = 30303
    protocol    = "udp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    description = "rpc port"
    from_port   = 8545
    to_port     = 8545
    protocol    = "tcp"
    cidr_blocks = [var.vpc_cidr]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
  tags = {
    Name = "Tf${var.network}Node"
  }
}

# Logs
resource "aws_cloudwatch_log_group" "cloud_watch_group" {
  for_each = var.nodeKeys

  name = "tf-${each.key}"
  retention_in_days = 14 # Logs are only kept for 14 days
  tags = {
    Name = "Tf${var.network}CloudWatchGroup${each.key}"
  }
}