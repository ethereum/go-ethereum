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

netstats="${NODE_NAME}-${wallet}-${instance_ip}:xinfin_xdpos_hybrid_network_stats@devnetstats.hashlabs.apothem.network:1999"


echo "Running a node with wallet: ${wallet} at IP: ${instance_ip}"
echo "Starting nodes with $bootnodes ..."

# Note: --gcmode=archive means node will store all historical data. This will lead to high memory usage. But sync mode require archive to sync
# https://github.com/XinFinOrg/XDPoSChain/issues/268

XDC --ethstats ${netstats} --gcmode archive \
--nat extip:${instance_ip} \
--bootnodes ${bootnodes} --syncmode full \
--datadir /work/xdcchain --networkid 551 \
--port $port --http --http-corsdomain "*" --http-addr 0.0.0.0 \
--http-port $rpc_port \
--http-api db,eth,debug,net,shh,txpool,personal,web3,XDPoS \
--http-vhosts "*" --unlock "${wallet}" --password /work/.pwd --mine \
--miner-gasprice "1" --miner-gaslimit "50000000" --verbosity ${log_level} \
--debugdatadir /work/xdcchain \
--store-reward \
--ws --ws-addr=0.0.0.0 --ws-port $ws_port \
--ws-origins "*" 2>&1 >>/work/xdcchain/xdc.log | tee -a /work/xdcchain/xdc.log
