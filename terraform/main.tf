provider "kubernetes" {
  host                   = module.eks.cluster_endpoint
  cluster_ca_certificate = base64decode(module.eks.cluster_certificate_authority_data)
  alias                  = "lc"

  exec {
    api_version = "client.authentication.k8s.io/v1beta1"
    command     = "aws"
    # This requires the awscli to be installed locally where Terraform is executed
    args = ["eks", "get-token", "--cluster-name", module.eks.cluster_name]
  }
}

################################################################################
# EKS Module
################################################################################

module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "19.15.3"

  providers = {
    kubernetes = kubernetes.lc
  }

  cluster_addons = {
    aws-ebs-csi-driver = {
      resolve_conflicts = "OVERWRITE"
#      addon_version     = var.aws-ebs-csi-driver_addon_version
    }
    coredns = {
      preserve      = true
 #     addon_version = var.coredns_addon_version

      timeouts = {
        create = "25m"
        delete = "10m"
      }
    }
    kube-proxy = {
 #     addon_version = var.kube-proxy_addon_version
    }
  }

  cluster_name    = local.name
  cluster_version = local.cluster_version

  enable_irsa = true

  cluster_endpoint_private_access = true
  cluster_endpoint_public_access  = false

  create_kms_key                  = true
#  kms_key_administrators          = [data.aws_caller_identity.current.arn]
  kms_key_aliases                 = ["eks/application-infrastructure"]
  kms_key_enable_default_policy   = true
  kms_key_deletion_window_in_days = 7
  enable_kms_key_rotation         = true

  cluster_tags = {
    Name = local.name
  }

  vpc_id                   = module.vpc.vpc_id
  subnet_ids               = module.vpc.private_subnets
  control_plane_subnet_ids = module.vpc.private_subnets

  manage_aws_auth_configmap = true

  #When deploying the cluster in a new env, this option should be enabled at first deployment.
  #If you receive a timeout error during apply, disable it, and redeploy.
  create_aws_auth_configmap = true

  # Extend cluster security group rules
  cluster_security_group_additional_rules = {
    egress_nodes_ephemeral_ports_tcp = {
      description                = "To node 1025-65535"
      protocol                   = "tcp"
      from_port                  = 1025
      to_port                    = 65535
      type                       = "egress"
      source_node_security_group = true
    }
  }
  # Extend node-to-node security group rules
  node_security_group_additional_rules = {
    ingress_self_all = {
      description = "Node to node all ports/protocols"
      protocol    = "-1"
      from_port   = 0
      to_port     = 0
      type        = "ingress"
      self        = true
    }
    egress_all = {
      description = "Node all egress"
      protocol    = "-1"
      from_port   = 0
      to_port     = 0
      type        = "egress"
      cidr_blocks = ["0.0.0.0/0"]
    }
  }
  eks_managed_node_groups = {
    blue = {
      name            = local.name
      use_name_prefix = true

      iam_role_additional_policies = {
        EBS_CSI = "arn:aws:iam::aws:policy/service-role/AmazonEBSCSIDriverPolicy"
        EFS_CSI = "arn:aws:iam::aws:policy/service-role/AmazonEFSCSIDriverPolicy"
      }

      subnet_ids = module.vpc.private_subnets

      min_size     = 1
      max_size     = 10
      desired_size = 1

      force_update_version = true
      instance_types       = ["t3.small"]
      ami_type             = "AL2_x86_64"

      description = "EKS managed node group launch template"

      ebs_optimized           = true
      disable_api_termination = false
      enable_monitoring       = false

      create_iam_role          = true
      iam_role_name            = "${local.name}-node-group"
      iam_role_use_name_prefix = false
      iam_role_description     = "EKS managed node group complete role"
      iam_role_tags = {
        Purpose = "Protector of the kubelet"
      }

      iam_role_attach_cni_policy = true

      create_security_group          = true
      security_group_name            = "${local.name}-node-group-sg"
      security_group_use_name_prefix = false

      tags = {
        ExtraTag                                     = "EKS managed node group"
        "k8s.io/cluster-autoscaler/enabled"          = 1
        "k8s.io/cluster-autoscaler/APP-DEV-EKS-RCON" = 1
      }
    }
  }

  aws_auth_roles = [
    {
      rolearn  = "arn:aws:iam::${local.account_id}:role/HeleCloud-Admin"
      username = "HeleCloud"
      groups   = ["system:masters"]
    }
  ]


  tags = {
    ClusterName = local.name
  }
}