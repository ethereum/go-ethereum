variable "devnet_node_kyes" {
  description = "Array of nodes keys."
  type        = map(any)
  
  /**
    Below is the list of private keys you need to specify. It follows the pattern of 
    {{Name of the node}}: {
      pk: {{Value of the node private key}},
      ... any other configuration we want to pass.
    }
    Note: No `n` is allowed in the node name
  **/
  default = {
    xdc1 = {
      pk  = "3efdb44088929167487da052125162b48d8d54fe8f7b7db11b5d5cc3b9a1c14b",
      isChaosNode = false # This is a placeholder, config not supported yet
    }
  }
}