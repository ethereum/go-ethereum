#!/bin/bash

if [ ! -d ./tmp/xdcchain ]
then
  echo "Creating a temporary directory for storing the xdcchain"
  mkdir tmp
  mkdir -p ./tmp/xdcchain
  touch ./tmp/.pwd
  
  # Randomly select a key from environment variable, seperated by ','
  if test -z "$PRIVATE_KEYS" 
  then
        echo "PRIVATE_KEYS environment variable has not been set. Please run again with `export PRIVATE_KEYS={{your key}} && make XDC-devnet-local`"
        exit 1
  fi
  IFS=', ' read -r -a private_keys <<< "$PRIVATE_KEYS"
  private_key=${private_keys[ $RANDOM % ${#private_keys[@]} ]}

  echo "${private_key}" >> ./tmp/key
  echo "Creating a new wallet"
  wallet=$(../../build/bin/XDC account import --password ./tmp/.pwd --datadir ./tmp/xdcchain ./tmp/key | awk -v FS="({|})" '{print $2}')
  ../../build/bin/XDC --datadir /tmp/xdcchain init ./genesis.json
else
  echo "Wallet already exist, re-use the same one. If you have changed the private key, please manually inspect the key if matches. Otherwise, delete the 'tmp' directory and start again!"
  wallet=$(../../build/bin/XDC account list --datadir /tmp/xdcchain | head -n 1 | awk -v FS="({|})" '{print $2}')
fi

input="./bootnodes.list"
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
  echo "Log level not set, default to verbosity of 3"
else
  echo "Log level found, set to $LOG_LEVEL"
  log_level=$LOG_LEVEL
fi

netstats="${NODE_NAME}-${wallet}-local:xinfin_xdpos_hybrid_network_stats@devnetstats.apothem.network:2000"

echo "Running a node with wallet: ${wallet} at local"

../../build/bin/XDC --ethstats ${netstats} --gcmode=archive \
--bootnodes ${bootnodes} --syncmode full \
--datadir ./tmp/xdcchain --networkid 551 \
-port 30303 --rpc --rpccorsdomain "*" --rpcaddr 0.0.0.0 \
--rpcport 8545 \
--rpcapi admin,db,eth,debug,miner,net,shh,txpool,personal,web3,XDPoS \
--rpcvhosts "*" --unlock "${wallet}" --password ./tmp/.pwd --mine \
--gasprice "1" --targetgaslimit "420000000" --verbosity ${log_level} \
--ws --wsaddr=0.0.0.0 --wsport 8555 \
--wsorigins "*" 2>&1 >>./tmp/xdc.log
