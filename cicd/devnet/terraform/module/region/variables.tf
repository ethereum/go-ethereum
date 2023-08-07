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