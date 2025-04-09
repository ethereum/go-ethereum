output "random_node_pool_suffix_output" {
  description = "The random generated suffix for app cluster's node pool"
  value       = random_string.random_node_pool_suffix.result
}
