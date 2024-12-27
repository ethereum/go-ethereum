locals {
    predefinedNodesConfig = jsondecode(data.aws_s3_object.xdc_node_config.body)
    envs = { for tuple in regexall("(.*)=(.*)", file(".env")) : tuple[0] => tuple[1] }
    logLevel = local.envs["log_level"]

    rpcDevnetNodeKeys = { "devnet-rpc1": local.predefinedNodesConfig["devnet-rpc1"]} // we hardcode the rpc to a single node for now
    rpcTestnetNodeKeys = { "testnet-rpc1": local.predefinedNodesConfig["testnet-rpc1"]} // we hardcode the rpc to a single node for now
    rpcMainnetNodeKeys = { "mainnet-rpc1": local.predefinedNodesConfig["mainnet-rpc1"]} // we hardcode the rpc to a single node for now
}

locals {
  ami_id = "ami-097c4e1feeea169e5"
  rpc_image = "xinfinorg/xdposchain:v2.2.0-beta1"
  vpc_id = "vpc-20a06846"
  aws_subnet_id = "subnet-4653ee20"
  ssh_key_name = "devnetkey"
}