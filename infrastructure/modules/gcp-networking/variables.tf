variable "project_id" {
  description = "The GCP project ID"
  type        = string
}

variable "region" {
  description = "The VPC region"
  type        = string
  default     = "europe-west1"
}
