variable "region" {
  description = "AWS region"
  type        = string
}

variable "nodeKeys" {
  description = "each miner's key"
  type        = map
}

variable "logLevel" {
  description = "containers log level"
  type        = string
}

variable "xdc_ecs_tasks_execution_role_arn" {
  description = "aws iam role resource arn"
  type        = string
}

variable "enableFixedIp" {
  description = "a flag to indicate whether fixed ip should be associated to the nodes. This is used for RPC node"
  type = bool
  default = false
}

variable "network" {
  description = "blockchain network"
  type = string
}

variable "cpu" {
  description = "container cpu"
  type = number
}

variable "memory" {
  description = "container memory"
  type = number
}

variable "vpc_cidr" {
  description = "vpc cidr"
  type = string
}

variable "subnet_cidr" {
  description = "subnet cidr"
  type = string
}