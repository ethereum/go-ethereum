data template_file container_definition {
  for_each = var.nodeKeys
  template = "${file("${path.module}/container-definition.tpl")}"

  vars = {
    image_environment = "${lookup(each.value, "imageEnvironment", "devnet")}"
    image_tag = "${lookup(each.value, "imageTag", "latest")}"
    node_name = "${each.key}"
    private_key = "${each.value.pk}"
    cloudwatch_group = "tf-${each.key}"
    cloudwatch_region = "${var.region}"
    log_level = "${lookup(each.value, "logLevel", "${var.logLevel}")}"
    chain_network = var.network
  }
}

resource "aws_ecs_task_definition" "task_definition_group" {
  for_each = var.nodeKeys
  
  family = "${var.network}-${each.key}"
  requires_compatibilities = ["FARGATE"]
  network_mode = "awsvpc"
  container_definitions = data.template_file.container_definition[each.key].rendered
  execution_role_arn = var.xdc_ecs_tasks_execution_role_arn
  task_role_arn = var.xdc_ecs_tasks_execution_role_arn
  
  # New nodes will consume a lot more CPU usage than existing nodes. 
  # This is due to sync is resource heavy. Recommending set to below if doing sync:
  # CPU = 2048, Memory = 4096
  # Please set it back to cpu 256 and memory of 2048 after sync is done to save the cost
  # cpu = 256
  # memory = 2048
  cpu = var.cpu
  memory = var.memory
  volume {
    name = "efs"

    efs_volume_configuration {
      file_system_id          = aws_efs_file_system.efs[each.key].id
      root_directory          = "/"
      transit_encryption      = "ENABLED"
      authorization_config {
        access_point_id = aws_efs_access_point.efs_access_point[each.key].id
        iam             = "DISABLED"
      }
    }
  }
  
  tags = {
       Name = "Tf${var.network}Ecs-${each.key}"
   }
}

data "aws_ecs_task_definition" "ecs_task_definition" {
  for_each = var.nodeKeys
  task_definition = aws_ecs_task_definition.task_definition_group[each.key].family
}

# ECS cluster
resource "aws_ecs_cluster" "ecs_cluster" {
  name    = "${var.network}-xdcnode-cluster"
  tags    = {
    Name        = "Tf${var.network}EcsCluster"
  }
}


resource "aws_ecs_service" "ecs_service" {
  for_each             = var.enableFixedIp ? {} : var.nodeKeys
  name                 = "ecs-service-${each.key}"
  cluster              = aws_ecs_cluster.ecs_cluster.id
  task_definition      = "${aws_ecs_task_definition.task_definition_group[each.key].family}:${max(aws_ecs_task_definition.task_definition_group[each.key].revision, data.aws_ecs_task_definition.ecs_task_definition[each.key].revision)}"
  launch_type          = "FARGATE"
  scheduling_strategy  = "REPLICA"
  desired_count        = 1
  force_new_deployment = true
  deployment_minimum_healthy_percent = 0
  deployment_maximum_percent = 100

  network_configuration {
    subnets          = [aws_subnet.subnet.id]
    assign_public_ip = true
    security_groups = [
      aws_default_security_group.xdcnode_security_group.id
    ]
  }

  deployment_circuit_breaker {
    enable = true
    rollback = false
  }

  tags = {
    Name        = "Tf${var.network}EcsService-${each.key}"
  }
}