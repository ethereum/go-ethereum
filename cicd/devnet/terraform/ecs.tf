data template_file devnet_container_definition {
  for_each = var.devnet_node_kyes
  template = "${file("${path.module}/container-definition.tpl")}"

  vars = {
    xdc_environment = "devnet"
    private_keys = "${each.value.pk}",
    cloudwatch_group = "tf-${each.key}"
  }
}

resource "aws_ecs_task_definition" "devnet_task_definition_group" {
  for_each = var.devnet_node_kyes
  
  family = "devnet-${each.key}"
  requires_compatibilities = ["FARGATE"]
  network_mode = "awsvpc"
  container_definitions = data.template_file.devnet_container_definition[each.key].rendered
  execution_role_arn = aws_iam_role.devnet_xdc_ecs_tasks_execution_role.arn
  task_role_arn = aws_iam_role.devnet_xdc_ecs_tasks_execution_role.arn
  
  cpu = 1024
  memory = 2048
  volume {
    name = "efs"

    efs_volume_configuration {
      file_system_id          = aws_efs_file_system.devnet_efs.id
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