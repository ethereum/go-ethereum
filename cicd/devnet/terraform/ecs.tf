data template_file devnet_container_definition {
  for_each = local.devnetNodeKyes
  template = "${file("${path.module}/container-definition.tpl")}"

  vars = {
    xdc_environment = "devnet"
    image_tag = "${lookup(each.value, "imageTag", "latest")}"
    node_name = "${each.key}"
    private_keys = "${each.value.pk}"
    cloudwatch_group = "tf-${each.key}"
    log_level = "${lookup(each.value, "logLevel", "${local.logLevel}")}"
  }
}

resource "aws_ecs_task_definition" "devnet_task_definition_group" {
  for_each = local.devnetNodeKyes
  
  family = "devnet-${each.key}"
  requires_compatibilities = ["FARGATE"]
  network_mode = "awsvpc"
  container_definitions = data.template_file.devnet_container_definition[each.key].rendered
  execution_role_arn = aws_iam_role.devnet_xdc_ecs_tasks_execution_role.arn
  task_role_arn = aws_iam_role.devnet_xdc_ecs_tasks_execution_role.arn
  
  # New nodes will consume a lot more CPU usage than existing nodes. 
  # This is due to sync is resource heavy. Recommending set to below if doing sync:
  # CPU = 2048, Memory = 4096
  # Please set it back to cpu 256 and memory of 2048 after sync is done to save the cost
  # cpu = 256
  # memory = 2048
  cpu = 256
  memory = 2048
  volume {
    name = "efs"

    efs_volume_configuration {
      file_system_id          = aws_efs_file_system.devnet_efs[each.key].id
      root_directory          = "/"
      transit_encryption      = "ENABLED"
      authorization_config {
        access_point_id = aws_efs_access_point.devnet_efs_access_point[each.key].id
        iam             = "DISABLED"
      }
    }
  }
  
  tags = {
       Name = "TfDevnetEcs-${each.key}"
   }
}

data "aws_ecs_task_definition" "devnet_ecs_task_definition" {
  for_each = local.devnetNodeKyes
  task_definition = aws_ecs_task_definition.devnet_task_definition_group[each.key].family
}

resource "aws_ecs_cluster" "devnet_ecs_cluster" {
  name = "devnet-xdcnode-cluster"
  tags = {
    Name        = "TfDevnetEcsCluster"
  }
}

resource "aws_ecs_service" "devnet_ecs_service" {
  for_each = local.devnetNodeKyes
  name                 = "ecs-service-${each.key}"
  cluster              = aws_ecs_cluster.devnet_ecs_cluster.id
  task_definition      = "${aws_ecs_task_definition.devnet_task_definition_group[each.key].family}:${max(aws_ecs_task_definition.devnet_task_definition_group[each.key].revision, data.aws_ecs_task_definition.devnet_ecs_task_definition[each.key].revision)}"
  launch_type          = "FARGATE"
  scheduling_strategy  = "REPLICA"
  desired_count        = 1
  force_new_deployment = true

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

  tags = {
    Name        = "TfDevnetEcsService-${each.key}"
  }
}