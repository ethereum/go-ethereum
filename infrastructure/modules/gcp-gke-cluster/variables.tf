variable "project_id" {
  description = "The GCP project ID"
  type        = string
}

variable "node_pool_node_count" {
  description = "If it is regional cluster it is nodes per zone. Otherwise number of total nodes"
  type        = number
  default     = 2
}

variable "node_pool_machine_type" {
  description = "The machine type used for pool-* nodes."
  type        = string
  default     = "e2-small"
}

variable "node_pool_disk_size_gb" {
  description = "The size of the nodes' disk in GB. Minimum 10G"
  type        = number
  default     = 10
}

variable "node_pool_disk_type" {
  description = "The disk type used for nodes."
  type        = string
  default     = "pd-standard"
}

variable "cluster_kubelet_config" {
  description = "Enable or disable cluster node pool kubelet_config"
  type        = bool
  default     = false
}

variable "cluster_name" {
  description = "The name of the cluster"
  type        = string
  default     = "test-cluster"
}

variable "cluster_location" {
  description = "the locaion of the cluster; region"
  type        = string
  default     = "europe-west1-b" # "europe-west1"
}