#!/bin/sh
set -exu

GETH_BIN_PATH=${GETH_BIN_PATH:-geth}
GENESIS_L1_PATH=${GENESIS_L1_PATH:-/genesis.json}
VERBOSITY=3
GETH_DATA_DIR=${GETH_DATA_DIR:-/data}
GETH_CHAINDATA_DIR="$GETH_DATA_DIR/geth/chaindata"
GETH_KEYSTORE_DIR="$GETH_DATA_DIR/keystore"
CHAIN_ID=$(cat "$GENESIS_L1_PATH" | jq -r .config.chainId)
RPC_PORT="${RPC_PORT:-8545}"
WS_PORT="${WS_PORT:-8546}"

# Generate signer key if needed
if [ ! -d "$GETH_KEYSTORE_DIR" ] && [ "$GETH_NODE_TYPE" = "signer" ]; then

	echo "$GETH_KEYSTORE_DIR missing, running account import"
	echo -n "pwd" > "$GETH_DATA_DIR"/password
	echo -n "$BLOCK_SIGNER_PRIVATE_KEY" | sed 's/0x//' > "$GETH_DATA_DIR"/block-signer-key
	"$GETH_BIN_PATH" --verbosity="$VERBOSITY" \
		--nousb \
		account import \
		--datadir="$GETH_DATA_DIR" \
		--password="$GETH_DATA_DIR"/password \
		"$GETH_DATA_DIR"/block-signer-key
else
	echo "$GETH_KEYSTORE_DIR exists."
fi

# Init geth if needed
if [ ! -d "$GETH_CHAINDATA_DIR" ]; then
	echo "$GETH_CHAINDATA_DIR missing, running init"
	echo "Initializing genesis."
	"$GETH_BIN_PATH" --verbosity="$VERBOSITY" \
		--nousb \
		--state.scheme=path \
		--db.engine=pebble \
		--datadir="$GETH_DATA_DIR" init \
		"$GENESIS_L1_PATH"
else
	echo "$GETH_CHAINDATA_DIR exists."
fi

# Obtain assigned container IP for p2p
NODE_IP=${NODE_IP:-$(hostname -i)}
echo "NODE_IP is set to: $NODE_IP"

PUBLIC_NODE_IP=${PUBLIC_NODE_IP:-""}
echo "EXTERNAL_NODE_IP is set to: $PUBLIC_NODE_IP"

# Set NAT_FLAG based on whether PUBLIC_NODE_IP is empty or not
if [ -n "$PUBLIC_NODE_IP" ]; then
    NAT_FLAG="--nat=extip:$PUBLIC_NODE_IP"
else
    NAT_FLAG="--nat=any"
fi

# (Optional) Echo the values for verification

if [ "$GETH_NODE_TYPE" = "bootnode" ]; then
	echo "Starting bootnode"

	# Generate boot.key
	echo "$BOOT_KEY" > $GETH_DATA_DIR/boot.key

	exec "$GETH_BIN_PATH" \
		--verbosity="$VERBOSITY" \
		--datadir="$GETH_DATA_DIR" \
		--port 30301 \
		--http \
		--http.corsdomain="*" \
		--http.vhosts="*" \
		--http.addr="$NODE_IP" \
		--http.port="$RPC_PORT" \
		--http.api=web3,debug,eth,txpool,net,engine \
		--ws \
		--ws.addr="$NODE_IP" \
		--ws.port="$WS_PORT" \
		--ws.origins="*" \
		--ws.api=debug,eth,txpool,net,engine \
		--syncmode=full \
		--gcmode=full \
		--state.scheme=path \
		--db.engine=pebble \
		--networkid=$CHAIN_ID \
		--nousb \
		--metrics \
		--metrics.addr="$NODE_IP" \
		--metrics.port=6060 \
  		--pprof \
    		--pprof.addr="$NODE_IP" \
        	--pprof.port=60601 \
		--nodekey $GETH_DATA_DIR/boot.key \
		--netrestrict $NET_RESTRICT \
		"$NAT_FLAG" \
		--txpool.accountqueue=512 \
		--rpc.allow-unprotected-txs

elif [ "$GETH_NODE_TYPE" = "signer" ]; then
	echo "Starting signer node"
 	echo "BOOTNODE_ENDPOINT is set to: $BOOTNODE_ENDPOINT"

	exec "$GETH_BIN_PATH" \
		--verbosity="$VERBOSITY" \
		--datadir="$GETH_DATA_DIR" \
		--port 30311 \
		--syncmode=full \
		--gcmode=full \
		--state.scheme=path \
		--db.engine=pebble \
		--http \
		--http.corsdomain="*" \
		--http.vhosts="*" \
		--http.addr="$NODE_IP" \
		--http.port="$RPC_PORT" \
		--http.api=web3,debug,eth,txpool,net,engine \
		--bootnodes $BOOTNODE_ENDPOINT \
		--networkid=$CHAIN_ID \
		--unlock=$BLOCK_SIGNER_ADDRESS \
		--password="$GETH_DATA_DIR"/password \
		--mine \
		--miner.etherbase=$BLOCK_SIGNER_ADDRESS \
		--allow-insecure-unlock \
		--nousb \
		--netrestrict $NET_RESTRICT \
		--metrics \
		--metrics.addr="$NODE_IP" \
		--metrics.port=6060 \
    		--pprof \
    		--pprof.addr="$NODE_IP" \
        	--pprof.port=60601 \
		--ws \
		--ws.addr="$NODE_IP" \
		--ws.port="$WS_PORT" \
		--ws.origins="*" \
		--ws.api=debug,eth,txpool,net,engine \
		--rpc.allow-unprotected-txs \
		--authrpc.addr="$NODE_IP" \
		--authrpc.port="8551" \
		--authrpc.vhosts="*" \
		--txpool.accountqueue=512 \
		"$NAT_FLAG"

elif [ "$GETH_NODE_TYPE" = "member" ]; then
	echo "Starting member node"
	echo "BOOTNODE_ENDPOINT is set to: $BOOTNODE_ENDPOINT"

	exec "$GETH_BIN_PATH" \
		--verbosity="$VERBOSITY" \
		--datadir="$GETH_DATA_DIR" \
		--port 30311 \
		--syncmode=full \
		--gcmode=full \
		--state.scheme=path \
		--db.engine=pebble \
		--http \
		--http.corsdomain="*" \
		--http.vhosts="*" \
		--http.addr="$NODE_IP" \
		--http.port="$RPC_PORT" \
		--http.api=web3,debug,eth,txpool,net,engine \
		--bootnodes $BOOTNODE_ENDPOINT \
		--networkid=$CHAIN_ID \
		--password="$GETH_DATA_DIR"/password \
		--metrics \
		--metrics.addr="$NODE_IP" \
		--metrics.port=6060 \
    		--pprof \
    		--pprof.addr="$NODE_IP" \
        	--pprof.port=60601 \
		--ws \
		--ws.addr="$NODE_IP" \
		--ws.port="$WS_PORT" \
		--ws.origins="*" \
		--ws.api=debug,eth,txpool,net,engine \
		--rpc.allow-unprotected-txs \
		--authrpc.addr="$NODE_IP" \
		--authrpc.port="8551" \
		--authrpc.vhosts="*" \
		--txpool.accountqueue=512 \
		"$NAT_FLAG"
else
	echo "Invalid GETH_NODE_TYPE specified"
fi
