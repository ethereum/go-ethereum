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
