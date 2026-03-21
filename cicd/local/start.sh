#!/bin/bash
if [ ! -d /work/xdcchain/XDC/chaindata ]
then
  if test -z "$PRIVATE_KEY"
  then
        echo "PRIVATE_KEY environment variable has not been set."
        exit 1
  fi
  echo $PRIVATE_KEY >> /tmp/key
  wallet=$(XDC account import --password .pwd --datadir /work/xdcchain /tmp/key | awk -F '[{}]' '{print $2}')
  XDC --datadir /work/xdcchain init /work/genesis.json
else
  wallet=$(XDC account list --datadir /work/xdcchain | head -n 1 | awk -F '[{}]' '{print $2}')
fi

input="/work/bootnodes.list"
bootnodes=""
while IFS= read -r line
do
    if [ -z "${bootnodes}" ]
    then
        bootnodes=$line
    else
        bootnodes="${bootnodes},$line"
    fi
done < "$input"
#check last line since it's not included in "read" command https://stackoverflow.com/questions/12916352/shell-script-read-missing-last-line
if [ -z "${bootnodes}" ]
then
    bootnodes=$line
else
    bootnodes="${bootnodes},$line"
fi

log_level=3
if test -z "$LOG_LEVEL"
then
  echo "Log level not set, default to verbosity of $log_level"
else
  echo "Log level found, set to $LOG_LEVEL"
  log_level=$LOG_LEVEL
fi

port=30303
if test -z "$PORT"
then
  echo "PORT not set, default to $port"
else
  echo "PORT found, set to $PORT"
  port=$PORT
fi

rpc_port=8545
if test -z "$RPC_PORT"
then
  echo "RPC_PORT not set, default to $rpc_port"
else
  echo "RPC_PORT found, set to $RPC_PORT"
  rpc_port=$RPC_PORT
fi

ws_port=8555
if test -z "$WS_PORT"
then
  echo "WS_PORT not set, default to $ws_port"
else
  echo "WS_PORT found, set to $WS_PORT"
  ws_port=$WS_PORT
fi

instance_ip=$(ifconfig eth0 | awk '/inet addr:/ {print $2}' | cut -d: -f2)
if test -z "$INSTANCE_IP"
then
  echo "INSTANCE_IP not set, default to $instance_ip"
else
  echo "INSTANCE_IP found, set to $INSTANCE_IP"
  instance_ip=$INSTANCE_IP
fi

sync_mode=full
if test -z "$SYNC_MODE"
then
  echo "SYNC_MODE not set, default to $sync_mode" #full or fast
else
  echo "SYNC_MODE found, set to $SYNC_MODE"
  sync_mode=$SYNC_MODE
fi

gc_mode=archive
if test -z "$GC_MODE"
then
  echo "GC_MODE not set, default to $gc_mode" #full or archive
else
  echo "GC_MODE found, set to $GC_MODE"
  gc_mode=$GC_MODE
fi

miner_gaslimit=50000000
if test -z "$MINER_GASLIMIT"
then
  echo "MINER_GASLIMIT not set, default to $miner_gaslimit"
else
  echo "MINER_GASLIMIT found, set to $MINER_GASLIMIT"
  miner_gaslimit=$MINER_GASLIMIT
fi

ethstats_address=localhost:2000
if test -z "$STATS_ADDRESS"
then
  echo "STATS_ADDRESS not set, default to $ethstats_address"
else
  echo "STATS_ADDRESS found, set to $STATS_ADDRESS"
  ethstats_address=$STATS_ADDRESS
fi

ethstats_secret=xinfin_xdpos_hybrid_local_stats
if test -z "$STATS_SECRET"
then
  echo "STATS_SECRET not set, default to $ethstats_secret"
else
  echo "STATS_SECRET found, set to $STATS_SECRET"
  ethstats_secret=$STATS_SECRET
fi

netstats="${NODE_NAME}-${wallet}:$ethstats_secret@$ethstats_address"

echo "Running a node with wallet: ${wallet} at IP: ${instance_ip}"
echo "Starting nodes with $bootnodes ..."

# Note: --gcmode=archive means node will store all historical data. This will lead to high memory usage. But sync mode require archive to sync
# https://github.com/XinFinOrg/XDPoSChain/issues/268

XDC \
--gcmode ${gc_mode} --syncmode ${sync_mode} \
--ethstats ${netstats} \
--nat extip:${instance_ip} \
--bootnodes ${bootnodes} \
--datadir /work/xdcchain \
--port $port --http --http-corsdomain "*" --http-addr 0.0.0.0 \
--http-port $rpc_port \
--http-api db,eth,net,txpool,web3,XDPoS \
--http-vhosts "*" --unlock "${wallet}" --password /work/.pwd --mine \
--miner-gasprice "1" --miner-gaslimit "${miner_gaslimit}" --verbosity ${log_level} \
--debugdatadir /work/xdcchain \
--store-reward \
--ws --ws-addr=0.0.0.0 --ws-port $ws_port \
--ws-origins "*" 2>&1 >>/work/xdcchain/xdc.log | tee -a /work/xdcchain/xdc.log
