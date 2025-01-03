#!/bin/bash
set -eo pipefail

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

log_level="${LOG_LEVEL:-3}"

port="${PORT:-30303}"

rpc_port="${RPC_PORT:-8545}"

ws_port="${WS_PORT:-8555}"

netstats="${NODE_NAME}-${wallet}:xinfin_xdpos_hybrid_network_stats@devnetstats.apothem.network:2000"


echo "Running a node with wallet: ${wallet}"
echo "Starting nodes with $bootnodes ..."

# Note: --gcmode=archive means node will store all historical data. This will lead to high memory usage. But sync mode require archive to sync
# https://github.com/XinFinOrg/XDPoSChain/issues/268

XDC --ethstats ${netstats} \
--gcmode archive \
--bootnodes ${bootnodes} \
--syncmode full \
--datadir /work/xdcchain \
--networkid 551 \
-port $port \
--rpc --rpccorsdomain "*" \
--rpcaddr 0.0.0.0 \
--rpcport $rpc_port \
--rpcapi db,eth,debug,net,shh,txpool,personal,web3,XDPoS \
--rpcvhosts "*" \
--unlock "${wallet}" \
--password /work/.pwd --mine \
--gasprice "1" --targetgaslimit "420000000" \
--verbosity ${log_level} \
--debugdatadir /work/xdcchain \
--ws \
--wsaddr=0.0.0.0 \
--wsport $ws_port \
--wsorigins "*" 2>&1 >>/work/xdcchain/xdc.log | tee -a /work/xdcchain/xdc.log
