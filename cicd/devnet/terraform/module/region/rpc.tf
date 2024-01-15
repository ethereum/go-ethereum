# Allocate an Elastic IP for the NLB
resource "aws_eip" "nlb_eip" {
  domain = "vpc"
}


# Create a Network Load Balancer
resource "aws_lb" "rpc_node_nlb" {
  count = var.enableFixedIp ? 1 : 0
  name               = "rpc-node-nlb"
  load_balancer_type = "network"

  enable_deletion_protection = false

  subnet_mapping {
    subnet_id     = aws_subnet.devnet_subnet.id
    allocation_id = aws_eip.nlb_eip.id
  }
}

# Listener and Target Group for the rpc node container
resource "aws_lb_target_group" "rpc_node_tg_8545" {
  count = var.enableFixedIp ? 1 : 0
  name     = "rpc-node-tg"
  port     = 8545
  protocol = "TCP"
  vpc_id   = aws_vpc.devnet_vpc.id
  target_type = "ip"
}

resource "aws_lb_listener" "rpc_node_listener_8545" {
  count = var.enableFixedIp ? 1 : 0
  load_balancer_arn = aws_lb.rpc_node_nlb[0].arn
  port              = 8545
  protocol          = "TCP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.rpc_node_tg_8545[0].arn
  }
}

resource "aws_ecs_service" "devnet_rpc_node_ecs_service" {
  for_each             = var.enableFixedIp ? var.devnetNodeKeys : {}
  name                 = "ecs-service-${each.key}"
  cluster              = aws_ecs_cluster.devnet_ecs_cluster.id
  task_definition      = "${aws_ecs_task_definition.devnet_task_definition_group[each.key].family}:${max(aws_ecs_task_definition.devnet_task_definition_group[each.key].revision, data.aws_ecs_task_definition.devnet_ecs_task_definition[each.key].revision)}"
  launch_type          = "FARGATE"
  scheduling_strategy  = "REPLICA"
  desired_count        = 1
  force_new_deployment = true
  deployment_minimum_healthy_percent = 0
  deployment_maximum_percent = 100

  network_configuration {
    subnets          = [aws_subnet.devnet_subnet.id]
    assign_public_ip = true
    security_groups = [
      aws_default_security_group.devnet_xdcnode_security_group.id
    ]
  }

  deployment_circuit_breaker {
    enable = true
    rollback = false
  }
  
  load_balancer {
    target_group_arn = aws_lb_target_group.rpc_node_tg_8545[0].arn
    container_name   = "tfXdcNode"
    container_port   = 8545
  }

  depends_on = [
    aws_lb_listener.rpc_node_listener_8545
  ]

  tags = {
    Name        = "TfDevnetRpcNodeEcsService-${each.key}"
  }
}

# Target Group for port 30303
resource "aws_lb_target_group" "rpc_node_tg_30303" {
  count = var.enableFixedIp ? 1 : 0
  name     = "rpc-node-tg-30303"
  port     = 30303
  protocol = "TCP"
  vpc_id   = aws_vpc.devnet_vpc.id
  target_type = "ip"
}

# Listener for port 30303
resource "aws_lb_listener" "rpc_node_listener_30303" {
  count = var.enableFixedIp ? 1 : 0
  load_balancer_arn = aws_lb.rpc_node_nlb[0].arn
  port              = 30303
  protocol          = "TCP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.rpc_node_tg_30303[0].arn
  }
}