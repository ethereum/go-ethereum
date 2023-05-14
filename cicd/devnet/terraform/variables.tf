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
    predefinedNodesConfig = jsondecode(data.aws_s3_object.devnet_xdc_node_config.body)
    envs = { for tuple in regexall("(.*)=(.*)", file(".env")) : tuple[0] => tuple[1] }
    logLevel = local.envs["log_level"]

    regions = [
      {
        "name": "us-east-2", // Ohio
        "start": local.envs["us-east-2_start"],
        "end": local.envs["us-east-2_end"],
      },
      {
        "name": "eu-west-1", // Ireland
        "start": local.envs["eu-west-1_start"],
        "end": local.envs["eu-west-1_end"],
      },
      {
        "name": "ap-southeast-2", // Sydney
        "start": local.envs["ap-southeast-2_start"],
        "end": local.envs["ap-southeast-2_end"],
      }
   ]

    keyNames = {
      for r in local.regions : 
        r.name => [for i in range(r.start, r.end+1) : "xdc${i}"]
    }

    devnetNodeKeys = {
      for r in local.regions : 
        r.name => { for i in local.keyNames[r.name]: i => local.predefinedNodesConfig[i] }
    }

    s3BucketName = "tf-devnet-bucket"
}
