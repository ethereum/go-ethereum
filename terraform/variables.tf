 variable "region" {
  default = "eu-west-1"
}

variable "tags" {
  default = {}
}

 variable "account_id" {
  description = "AWS Account ID"
  type        = string
  default     = null
}
