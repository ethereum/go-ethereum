#!/bin/sh
set -exu

GENESIS_L1_PATH="/genesis.json"
VERBOSITY=3
GETH_DATA_DIR=/data
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
	geth --verbosity="$VERBOSITY" \
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
	geth --verbosity="$VERBOSITY" \
		--nousb \
		--state.scheme=path \
		--db.engine=pebble \
		--datadir="$GETH_DATA_DIR" init \
		"$GENESIS_L1_PATH"
else
	echo "$GETH_CHAINDATA_DIR exists."
fi

# Obtain assigned container IP for p2p
NODE_IP=$(hostname -i)

if [ "$GETH_NODE_TYPE" = "bootnode" ]; then
	echo "Starting bootnode"

	# Generate boot.key
	echo "$BOOT_KEY" > $GETH_DATA_DIR/boot.key

	exec geth \
		--verbosity="$VERBOSITY" \
		--datadir="$GETH_DATA_DIR" \
		--port 30301 \
		--http \
		--http.corsdomain="*" \
		--http.vhosts="*" \
		--http.addr=0.0.0.0 \
		--http.port="$RPC_PORT" \
		--http.api=web3,debug,eth,txpool,net,engine \
		--ws \
		--ws.addr=0.0.0.0 \
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
		--metrics.addr=0.0.0.0 \
		--metrics.port=6060 \
  		--pprof \
    		--pprof.addr=0.0.0.0 \
        	--pprof.port=60601 \
		--nodekey $GETH_DATA_DIR/boot.key \
		--netrestrict $NET_RESTRICT \
		--nat extip:$NODE_IP \
		--txpool.accountqueue=512 \
		--rpc.allow-unprotected-txs

elif [ "$GETH_NODE_TYPE" = "signer" ]; then
	echo "Starting signer node"

	exec geth \
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
		--http.addr=0.0.0.0 \
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
		--metrics.addr=0.0.0.0 \
		--metrics.port=6060 \
    		--pprof \
    		--pprof.addr=0.0.0.0 \
        	--pprof.port=60601 \
		--ws \
		--ws.addr=0.0.0.0 \
		--ws.port="$WS_PORT" \
		--ws.origins="*" \
		--ws.api=debug,eth,txpool,net,engine \
		--rpc.allow-unprotected-txs \
		--authrpc.addr="0.0.0.0" \
		--authrpc.port="8551" \
		--authrpc.vhosts="*" \
		--txpool.accountqueue=512 \
		--nat extip:$NODE_IP
else
	echo "Invalid GETH_NODE_TYPE specified"
fi