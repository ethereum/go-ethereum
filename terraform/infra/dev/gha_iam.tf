
### IAM for go-ethereum Github Actions
resource "aws_iam_role" "go-ethereum_github_actions" {
  name = "go-ethereum-github-actions-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17",
    Statement = [
      {
        Effect = "Allow",
        Principal = {
          Federated = "arn:aws:iam::${var.account_id}:oidc-provider/token.actions.githubusercontent.com"
        },
        Action = "sts:AssumeRoleWithWebIdentity",
        Condition = {
          StringLike = {
            "token.actions.githubusercontent.com:sub" : "repo:teddy931130/go-ethereum:*"
          },
          StringEquals = {
            "token.actions.githubusercontent.com:aud" : "sts.amazonaws.com",
          }
        }
      }
    ]
  })

  tags = {
    "Name"        = "go-ethereum"
    "Environment" = var.env
  }
}

resource "aws_iam_policy" "go-ethereum_github_actions_policy" {
  name        = "limechain-go-ethereum-github-actions-ecr-policy"
  description = "Permissions for Limechain-go-ethereum GitHub Actions to pull images from ECR"

  policy = jsonencode({
    Version = "2012-10-17",
    Statement = [
      {
        Effect = "Allow",
        Action = [
          "ecr:BatchCheckLayerAvailability",
          "ecr:BatchGetImage",
          "ecr:GetDownloadUrlForLayer",
          "ecr:DescribeImages",
          "ecr:InitiateLayerUpload",
          "ecr:UploadLayerPart",
          "ecr:CompleteLayerUpload",
          "ecr:PutImage",
        ],
        Resource = "arn:aws:ecr:${var.region}:${var.account_id}:repository/limechain-devops-task/go-ethereum"
      },
      {
        Effect = "Allow",
        Action = [
          "ecr:GetAuthorizationToken"
        ],
        Resource = "*"
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "go-ethereum_github_actions_attach" {
  role       = aws_iam_role.go-ethereum_github_actions.name
  policy_arn = aws_iam_policy.go-ethereum_github_actions_policy.arn
}
