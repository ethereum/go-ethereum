variable "region" {
  description = "AWS region"
  type        = string
}

variable "devnetNodeKeys" {
  description = "each miner's key"
  type        = map
}

variable "logLevel" {
  description = "containers log level"
  type        = string
}

variable "devnet_xdc_ecs_tasks_execution_role_arn" {
  description = "aws iam role resource arn"
  type        = string
}

variable "enableFixedIp" {
  description = "a flag to indicate whether fixed ip should be associated to the nodes. This is used for RPC node"
  type = bool
  default = false
}

variable docker_tag {
  type        = string
  default     = "latest"
  description = "description"
}
