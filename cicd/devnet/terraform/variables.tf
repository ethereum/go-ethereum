locals {
    /**
    Load the nodes data from s3
    Below is the the format the config needs to follow:
    {{Name of the node, in a pattern of 'xdc'+ number. i.e xdc50}}: {
      pk: {{Value of the node private key}},
      ... any other configuration we want to pass.
    }
    Note: No `n` is allowed in the node name
  **/
    predefinedNodesConfig = jsondecode(data.aws_s3_bucket_object.devnet_xdc_node_config.body)
    envs = { for tuple in regexall("(.*)=(.*)", file(".env")) : tuple[0] => tuple[1] }
    logLevel = local.envs["log_level"]
    keyNames =[for i in range(tonumber(local.envs["num_of_nodes"])) : "xdc${i}"]
    devnetNodeKyes = {
      for i in local.keyNames: i => local.predefinedNodesConfig[i]
    }
}