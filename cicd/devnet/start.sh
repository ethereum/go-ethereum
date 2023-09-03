#!/bin/bash

# Update AWS Max Health can resolve below issue, and it's already addressed
# echo "Preparing to start the XDC chain, it's likely to take up to 1 minute"
# Sleep for > 30 as we need to wait for the ECS tasks container being killed by fargate. Otherwise it will ended up with two same nodes running on a single /work/xdcchain directory
# sleep 60

if [ ! -d /work/xdcchain/XDC/chaindata ]
then
  # Randomly select a key from environment variable, seperated by ','
  if test -z "$PRIVATE_KEYS" 
  then
        echo "PRIVATE_KEYS environment variable has not been set. You need to pass at least one PK, or you can pass multiple PK seperated by ',', we will randomly choose one for you"
        exit 1
  fi
  IFS=', ' read -r -a private_keys <<< "$PRIVATE_KEYS"
  private_key=${private_keys[ $RANDOM % ${#private_keys[@]} ]}

  echo "${private_key}" >> /tmp/key
  echo "Creating a new wallet"
  wallet=$(XDC account import --password .pwd --datadir /work/xdcchain /tmp/key |  awk -F '[{}]' '{print $2}')
  XDC --datadir /work/xdcchain init /work/genesis.json
else
  echo "Wallet already exist, re-use the same one"
  wallet=$(XDC account list --datadir /work/xdcchain | head -n 1 |  awk -F '[{}]' '{print $2}')
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
  echo "WS_PORT not set, default to  $ws_port"
else
  echo "WS_PORT found, set to $WS_PORT"
  ws_port=$WS_PORT
fi

INSTANCE_IP=$(curl https://checkip.amazonaws.com)
netstats="${NODE_NAME}-${wallet}-${INSTANCE_IP}:xinfin_xdpos_hybrid_network_stats@devnetstats.apothem.network:2000"


echo "Running a node with wallet: ${wallet} at IP: ${INSTANCE_IP}"
echo "Starting nodes with $bootnodes ..."

# Note: --gcmode=archive means node will store all historical data. This will lead to high memory usage. But sync mode require archive to sync
# https://github.com/XinFinOrg/XDPoSChain/issues/268

XDC --ethstats ${netstats} --gcmode archive \
--nat extip:${INSTANCE_IP} \
--bootnodes ${bootnodes} --syncmode full \
--datadir /work/xdcchain --networkid 551 \
-port $port --rpc --rpccorsdomain "*" --rpcaddr 0.0.0.0 \
--rpcport $rpc_port \
--rpcapi admin,db,eth,debug,net,shh,txpool,personal,web3,XDPoS \
--rpcvhosts "*" --unlock "${wallet}" --password /work/.pwd --mine \
--gasprice "1" --targetgaslimit "420000000" --verbosity ${log_level} \
--periodicprofile --debugdatadir /work/xdcchain \
--ws --wsaddr=0.0.0.0 --wsport $ws_port \
--wsorigins "*" 2>&1 >>/work/xdcchain/xdc.log | tee -a /work/xdcchain/xdc.log
