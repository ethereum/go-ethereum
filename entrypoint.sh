#!/bin/sh

init_node() {
	echo "MEV Geth Starting..."
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
          --gcmode archive \
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
          --gcmode archive \
          --cache 4096 \
          --snapshot=false \
          --maxpeers $connections
        if [ $? -ne 0 ]
        then
            echo "Node failed to start; exiting."
            exit 1
        fi
    fi
}


init_node
start_node