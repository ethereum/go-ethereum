#!/bin/sh -x
# Starts the Mev-Geth node client
# Written by Luke Youngblood, luke@blockscale.net

# network=mainnet # normally set by environment
# syncmode=fast   # normally set by environment
# rpcport=8545    # normally set by environment
# wsport=8546     # normally set by environment
# netport=30303   # normally set by environment

init_node() {
	# You can put any commands you would like to run to initialize the node here.
	echo Initializing node...
}

start_node() {
	if [ $network = "goerli" ]
    then
        geth \
          --port $netport \
          --http \
          --http.addr 0.0.0.0 \
          --http.port $rpcport \
          --http.api eth,net,web3 \
          --http.vhosts '*' \
          --http.corsdomain '*' \
          --graphql \
          --graphql.corsdomain '*' \
          --graphql.vhosts '*' \
          --ws \
          --ws.addr 0.0.0.0 \
          --ws.port $wsport \
          --ws.api eth,net,web3 \
          --ws.origins '*' \
          --syncmode $syncmode \
          --cache 4096 \
          --maxpeers $connections \
          --goerli
        if [ $? -ne 0 ]
        then
            echo "Node failed to start; exiting."
            exit 1
        fi
    else 
        geth \
          --port $netport \
          --http \
          --http.addr 0.0.0.0 \
          --http.port $rpcport \
          --http.api eth,net,web3 \
          --http.vhosts '*' \
          --http.corsdomain '*' \
          --graphql \
          --graphql.corsdomain '*' \
          --graphql.vhosts '*' \
          --ws \
          --ws.addr 0.0.0.0 \
          --ws.port $wsport \
          --ws.api eth,net,web3 \
          --ws.origins '*' \
          --syncmode $syncmode \
          --cache 4096 \
          --maxpeers $connections
        if [ $? -ne 0 ]
        then
            echo "Node failed to start; exiting."
            exit 1
        fi
    fi
}

s3_sync() {
    # Determine data directory
    if [ $network = "goerli" ]
    then
        datadir=/root/.ethereum/goerli/geth/chaindata
    else
        datadir=/root/.ethereum/geth/chaindata
    fi
	# If the current1 key exists, node1 is the most current set of blockchain data
	echo "A 404 error below is expected and nothing to be concerned with."
	aws s3api head-object --request-payer requester --bucket $chainbucket --key current1
	if [ $? -eq 0 ]
	then
		s3key=node1
	else
		s3key=node2
	fi
	aws s3 sync --only-show-errors --request-payer requester --region $region s3://$chainbucket/$s3key $datadir
}

# main

init_node
s3_sync
start_node
