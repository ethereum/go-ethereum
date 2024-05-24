terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.13.1"
    }
  }
}
variable network {
  type = string
}
variable vpc_id {
  type = string
}
variable aws_subnet_id {
  type = string
}
variable ami_id {
  type = string
}
variable instance_type {
  type = string
}
variable ssh_key_name {
  type = string
}
variable rpc_image {
  type = string
}

resource "aws_security_group" "rpc_sg" {
  name_prefix = "${var.network}_rpc_sg"

  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
  
  ingress {
    from_port   = 30303
    to_port     = 30303
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 8545
    to_port     = 8545
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 8555
    to_port     = 8555
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_instance" "rpc_instance" {
  instance_type           = var.instance_type 
  ami                     = var.ami_id
  tags                    = {
                              Name = var.network
                            }
  key_name                = var.ssh_key_name
  vpc_security_group_ids  = [aws_security_group.rpc_sg.id]
  ebs_block_device {
    device_name = "/dev/xvda"
    volume_size = 500
  }


  #below still need to remove git checkout {{branch}} after files merged to master
  user_data = <<-EOF
              #!/bin/bash
              sudo yum update -y
              sudo yum upgrade -y
              sudo yum install git -y
              sudo yum install docker -y
              mkdir -p /root/.docker/cli-plugins
              curl -SL https://github.com/docker/compose/releases/download/v2.25.0/docker-compose-linux-x86_64 -o /root/.docker/cli-plugins/docker-compose
              sudo chmod +x /root/.docker/cli-plugins/docker-compose
              echo checking compose version
              docker compose version
              sudo systemctl enable docker
              sudo systemctl start docker
              mkdir -p /work
              cd /work
              git clone https://github.com/XinFinOrg/XinFin-Node
              cd /work/XinFin-Node/${var.network}
              export RPC_IMAGE="${var.rpc_image}"
              echo RPC_IMAGE=$RPC_IMAGE
              ./docker-up-hash.sh
              EOF
}