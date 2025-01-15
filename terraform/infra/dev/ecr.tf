module "ecr_go-ethereum" {
  source  = "terraform-aws-modules/ecr/aws"
  version = "2.3.1"

  repository_name = "limechain-devops-task/go-ethereum"
  repository_type = "private"

  repository_read_write_access_arns = ["arn:aws:iam::${var.account_id}:role/${aws_iam_role.go-ethereum_github_actions.name}"]
  create_repository_policy          = true
  repository_lifecycle_policy = jsonencode({
    rules = [
      {
        rulePriority = 1,
        description  = "Keep last 20 images",
        selection = {
          tagStatus   = "any",
          countType   = "imageCountMoreThan",
          countNumber = 20
        },
        action = {
          type = "expire"
        }
      }
    ]
  })

  # Registry Scanning Configuration
  manage_registry_scanning_configuration = true
  registry_scan_type                     = "BASIC"
  registry_scan_rules = [
    {
      scan_frequency = "SCAN_ON_PUSH"
      filter = [
        {
          filter      = "*"
          filter_type = "WILDCARD"
        }
      ]
    }
  ]

  tags = {
    "Name"        = "limechain-devops-task"
    "Environment" = var.env
  }
}
