#!/bin/sh
set -exu

GETH_BIN_PATH=${GETH_BIN_PATH:-geth}
GENESIS_L1_PATH=${GENESIS_L1_PATH:-/genesis.json}
GETH_VERBOSITY=${GETH_VERBOSITY:-3}
GETH_LOG_FORMAT=${GETH_LOG_FORMAT:-terminal}
GETH_LOG_TAGS=${GETH_LOG_TAGS:-}
GETH_SYNC_MODE=${GETH_SYNC_MODE:-snap}
GETH_STATE_SCHEME=${GETH_STATE_SCHEME:-path}
GETH_DATA_DIR=${GETH_DATA_DIR:-/data}
GETH_CHAINDATA_DIR="$GETH_DATA_DIR/geth/chaindata"
GETH_KEYSTORE_DIR="$GETH_DATA_DIR/keystore"
GETH_KEYSTORE_PASSWORD=${GETH_KEYSTORE_PASSWORD:-"primev"}
GETH_ZERO_FEE_ADDRESSES=${GETH_ZERO_FEE_ADDRESSES:-}
CHAIN_ID=$(cat "$GENESIS_L1_PATH" | jq -r .config.chainId)
RPC_PORT="${RPC_PORT:-8545}"
WS_PORT="${WS_PORT:-8546}"
BLOCK_SIGNER_PRIVATE_KEY=${BLOCK_SIGNER_PRIVATE_KEY:-""}

if [ -n "$GETH_LOG_TAGS" ]; then
    LOG_TAGS_OPTION="--log.tags=$GETH_LOG_TAGS"
else
    LOG_TAGS_OPTION=""
fi

if [ -n "$GETH_ZERO_FEE_ADDRESSES" ]; then
    ZERO_FEE_ADDRESSES="--zero-fee-addresses=$GETH_ZERO_FEE_ADDRESSES"
else
    ZERO_FEE_ADDRESSES=""
fi

# Generate signer key if needed
if [ "$GETH_NODE_TYPE" = "signer" ]; then
	if [ ! -f "$GETH_DATA_DIR/password" ]; then
		echo -n "$GETH_KEYSTORE_PASSWORD" > "$GETH_DATA_DIR"/password
	fi
	if [ ! -d "$GETH_KEYSTORE_DIR" ]; then
		if [ -n "$BLOCK_SIGNER_PRIVATE_KEY" ]; then
			echo "$GETH_KEYSTORE_DIR missing, running account import"
			echo -n "$BLOCK_SIGNER_PRIVATE_KEY" | sed 's/0x//' > "$GETH_DATA_DIR"/block-signer-key
			"$GETH_BIN_PATH" \
				--verbosity="$GETH_VERBOSITY" \
				--log.format="$GETH_LOG_FORMAT" \
				$LOG_TAGS_OPTION \
				--nousb \
				account import \
				--datadir="$GETH_DATA_DIR" \
				--password="$GETH_DATA_DIR"/password \
				"$GETH_DATA_DIR"/block-signer-key
		fi
	else
		echo "$GETH_KEYSTORE_DIR exists."
		if [ -z "$BLOCK_SIGNER_PRIVATE_KEY" ]; then
			GETH_ACCOUNT_LIST=$("$GETH_BIN_PATH" --verbosity="$GETH_VERBOSITY" account list --datadir "$GETH_DATA_DIR")
			BLOCK_SIGNER_ADDRESS_WITHOUT_PREFIX=$(echo "$GETH_ACCOUNT_LIST" | grep -oE '[0-9a-fA-F]{40}$')
			BLOCK_SIGNER_ADDRESS="0x$BLOCK_SIGNER_ADDRESS_WITHOUT_PREFIX"
			echo "Block signer address with 0x prefix: $BLOCK_SIGNER_ADDRESS"
		fi
	fi
fi

# Init geth if needed
if [ ! -d "$GETH_CHAINDATA_DIR" ]; then
	echo "$GETH_CHAINDATA_DIR missing, running init"
	echo "Initializing genesis."
	"$GETH_BIN_PATH" \
		--verbosity="$GETH_VERBOSITY" \
		--log.format="$GETH_LOG_FORMAT" \
		$LOG_TAGS_OPTION \
		--nousb \
		--state.scheme="$GETH_STATE_SCHEME" \
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
    NAT_FLAG="--nat=none"
fi

# (Optional) Echo the values for verification

if [ "$GETH_NODE_TYPE" = "bootnode" ]; then
	echo "Starting bootnode"
	echo "$NODE_KEY" > $GETH_DATA_DIR/nodekey

	exec "$GETH_BIN_PATH" \
		--verbosity="$GETH_VERBOSITY" \
		--log.format="$GETH_LOG_FORMAT" \
		$LOG_TAGS_OPTION \
		$ZERO_FEE_ADDRESSES \
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
		--syncmode="${GETH_SYNC_MODE}" \
		--gcmode=full \
		--state.scheme="$GETH_STATE_SCHEME" \
		--db.engine=pebble \
		--networkid=$CHAIN_ID \
		--nousb \
		--metrics \
		--metrics.addr="$NODE_IP" \
		--metrics.port=6060 \
		--pprof \
		--pprof.addr="$NODE_IP" \
		--pprof.port=60601 \
		--nodekey $GETH_DATA_DIR/nodekey \
		--netrestrict $NET_RESTRICT \
		"$NAT_FLAG" \
		--txpool.accountqueue=512 \
		--rpc.allow-unprotected-txs \
		--miner.gasprice=50000000 \
		--txpool.pricelimit=50000000 \
		--gpo.maxprice=500000000000 \

elif [ "$GETH_NODE_TYPE" = "signer" ]; then
	echo "Starting signer node"
	echo "$NODE_KEY" > $GETH_DATA_DIR/nodekey

	echo "BOOTNODE_ENDPOINT is set to: $BOOTNODE_ENDPOINT"
	GETH_PORT="${GETH_PORT:-30311}"

	exec "$GETH_BIN_PATH" \
		--verbosity="$GETH_VERBOSITY" \
		--log.format="$GETH_LOG_FORMAT" \
		$LOG_TAGS_OPTION \
		$ZERO_FEE_ADDRESSES \
		--datadir="$GETH_DATA_DIR" \
		--port="$GETH_PORT" \
		--syncmode="${GETH_SYNC_MODE}" \
		--gcmode=full \
		--state.scheme="$GETH_STATE_SCHEME" \
		--db.engine=pebble \
		--http \
		--http.corsdomain="*" \
		--http.vhosts="*" \
		--http.addr="$NODE_IP" \
		--http.port="$RPC_PORT" \
		--http.api=web3,debug,eth,txpool,net,engine,clique \
		--bootnodes $BOOTNODE_ENDPOINT \
		--networkid=$CHAIN_ID \
		--unlock=$BLOCK_SIGNER_ADDRESS \
		--password="$GETH_DATA_DIR"/password \
		--mine \
		--miner.etherbase=$BLOCK_SIGNER_ADDRESS \
		--allow-insecure-unlock \
		--nousb \
		--nodekey $GETH_DATA_DIR/nodekey \
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
		--miner.gasprice=50000000 \
		--txpool.pricelimit=50000000 \
		--gpo.maxprice=500000000000 \
		"$NAT_FLAG"

elif [ "$GETH_NODE_TYPE" = "member" ]; then
	echo "Starting member node"
	echo "$NODE_KEY" > $GETH_DATA_DIR/nodekey
	echo "BOOTNODE_ENDPOINT is set to: $BOOTNODE_ENDPOINT"
	GETH_PORT="${GETH_PORT:-30311}"

	exec "$GETH_BIN_PATH" \
		--verbosity="$GETH_VERBOSITY" \
		--log.format="$GETH_LOG_FORMAT" \
		$LOG_TAGS_OPTION \
		$ZERO_FEE_ADDRESSES \
		--datadir="$GETH_DATA_DIR" \
		--port="$GETH_PORT" \
		--syncmode="${GETH_SYNC_MODE}" \
		--gcmode=full \
		--state.scheme="$GETH_STATE_SCHEME" \
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
		--nodekey $GETH_DATA_DIR/nodekey \
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
		--miner.gasprice=50000000 \
		--txpool.pricelimit=50000000 \
		--gpo.maxprice=500000000000 \
		"$NAT_FLAG"
elif [ "$GETH_NODE_TYPE" = "archive" ]; then
	echo "Starting archive node"
	echo "BOOTNODE_ENDPOINT is set to: $BOOTNODE_ENDPOINT"
	GETH_PORT="${GETH_PORT:-30311}"

	exec "$GETH_BIN_PATH" \
		--verbosity="$GETH_VERBOSITY" \
		--log.format="$GETH_LOG_FORMAT" \
		$LOG_TAGS_OPTION \
		--datadir="$GETH_DATA_DIR" \
		--port="$GETH_PORT" \
		--syncmode="${GETH_SYNC_MODE}" \
		--gcmode=archive \
		--history.state=0 \
		--history.transactions=0 \
		--state.scheme="$GETH_STATE_SCHEME" \
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
		--miner.gasprice=50000000 \
		--txpool.pricelimit=50000000 \
		--gpo.maxprice=500000000000 \
		"$NAT_FLAG"
else
	echo "Invalid GETH_NODE_TYPE specified"
fi
