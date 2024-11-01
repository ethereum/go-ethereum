terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
    }
  }
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
    cidr_blocks = ["10.0.0.0/16"]
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

# Logs
resource "aws_cloudwatch_log_group" "devnet_cloud_watch_group" {
  for_each = var.devnetNodeKeys

  name = "tf-${each.key}"
  retention_in_days = 14 # Logs are only kept for 14 days
  tags = {
    Name = "TfDevnetCloudWatchGroup${each.key}"
  }
}