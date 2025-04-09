variable "project_id" {
  description = "The GCP project ID"
  type        = string
  default     = "lime-demo-aap"
}

variable "gke_cluster_location" {
  description = "the locaion of the cluster; region"
  type        = string
  default     = "europe-west1"
}

