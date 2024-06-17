locals {
  name            = "Limechain-project"
  cluster_version = "1.28"
  region          = var.region
  account_id      = "924841524423"

  tags = merge({
    Example    = local.name
    GithubRepo = "terraform-aws-eks"
    GithubOrg  = "terraform-aws-modules"
  }, var.tags)
}

