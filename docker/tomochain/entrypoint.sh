#!/bin/sh

# vars from docker env
# - IDENTITY (default to empty)
# - PASSWORD (default to empty)
# - PRIVATE_KEY (default to empty)
# - BOOTNODES (default to empty)
# - EXTIP (default to empty)
# - VERBOSITY (default to 3)
# - MAXPEERS (default to 25)
# - SYNC_MODE (default to 'full')
# - NETWORK_ID (default to '89')
# - WS_SECRET (default to empty)
# - NETSTATS_HOST (default to 'netstats-server:3000')
# - NETSTATS_PORT (default to 'netstats-server:3000')

# constants
DATA_DIR="data"
KEYSTORE_DIR="keystore"

# variables
genesisPath=""
params=""
accountsCount=$(
  tomo account list --datadir $DATA_DIR  --keystore $KEYSTORE_DIR \
  2> /dev/null \
  | wc -l
)

# file to env
for env in IDENTITY PASSWORD PRIVATE_KEY BOOTNODES WS_SECRET NETSTATS_HOST \
           NETSTATS_PORT EXTIP SYNC_MODE NETWORK_ID ANNOUNCE_TXS STORE_REWARD DEBUG_MODE MAXPEERS; do
  file=$(eval echo "\$${env}_FILE")
  if [[ -f $file ]] && [[ ! -z $file ]]; then
    echo "Replacing $env by $file"
    export $env=$(cat $file)
  elif [[ "$env" == "BOOTNODES" ]] && [[ ! -z $file ]]; then
    echo "Bootnodes file is not available. Waiting for it to be provisioned..."
    while true ; do
      if [[ -f $file ]] && [[ $(grep -e enode $file) ]]; then
        echo "Fount bootnode file."
        break
      fi
      echo "Still no bootnodes file, sleeping..."
      sleep 5
    done
    export $env=$(cat $file)
  fi
done

# networkid
if [[ ! -z $NETWORK_ID ]]; then
  case $NETWORK_ID in
    88 )
      genesisPath="mainnet.json"
      ;;
    89 )
      genesisPath="testnet.json"
      params="$params --tomo-testnet --gcmode archive --rpcapi db,eth,net,web3,personal,debug"
      ;;
    90 )
      genesisPath="devnet.json"
      ;;
    * )
      echo "network id not supported"
      ;;
  esac
  params="$params --networkid $NETWORK_ID"
fi

# custom genesis path
if [[ ! -z $GENESIS_PATH ]]; then
  genesisPath="$GENESIS_PATH"
fi

# data dir
if [[ ! -d $DATA_DIR/tomo ]]; then
  echo "No blockchain data, creating genesis block."
  tomo init $genesisPath --datadir $DATA_DIR 2> /dev/null
fi

# identity
if [[ -z $IDENTITY ]]; then
  IDENTITY="unnamed_$(< /dev/urandom tr -dc _A-Z-a-z-0-9 | head -c6)"
fi

# password file
if [[ ! -f ./password ]]; then
  if [[ ! -z $PASSWORD ]]; then
    echo "Password env is set. Writing into file."
    echo "$PASSWORD" > ./password
  else
    echo "No password set (or empty), generating a new one"
    $(< /dev/urandom tr -dc _A-Z-a-z-0-9 | head -c${1:-32} > password)
  fi
fi

# private key
if [[ $accountsCount -le 0 ]]; then
  echo "No accounts found"
  if [[ ! -z $PRIVATE_KEY ]]; then
    echo "Creating account from private key"
    echo "$PRIVATE_KEY" > ./private_key
    tomo  account import ./private_key \
      --datadir $DATA_DIR \
      --keystore $KEYSTORE_DIR \
      --password ./password
    rm ./private_key
  else
    echo "Creating new account"
    tomo account new \
      --datadir $DATA_DIR \
      --keystore $KEYSTORE_DIR \
      --password ./password
  fi
fi
account=$(
  tomo account list --datadir $DATA_DIR  --keystore $KEYSTORE_DIR \
  2> /dev/null \
  | head -n 1 \
  | cut -d"{" -f 2 | cut -d"}" -f 1
)
echo "Using account $account"
params="$params --unlock $account"

# bootnodes
if [[ ! -z $BOOTNODES ]]; then
  params="$params --bootnodes $BOOTNODES"
fi

# extip
if [[ ! -z $EXTIP ]]; then
  params="$params --nat extip:${EXTIP}"
fi

# syncmode
if [[ ! -z $SYNC_MODE ]]; then
  params="$params --syncmode ${SYNC_MODE}"
fi

# netstats
if [[ ! -z $WS_SECRET ]]; then
  echo "Will report to netstats server ${NETSTATS_HOST}:${NETSTATS_PORT}"
  params="$params --ethstats ${IDENTITY}:${WS_SECRET}@${NETSTATS_HOST}:${NETSTATS_PORT}"
else
  echo "WS_SECRET not set, will not report to netstats server."
fi

# annonce txs
if [[ ! -z $ANNOUNCE_TXS ]]; then
  params="$params --announce-txs"
fi

# store reward
if [[ ! -z $STORE_REWARD ]]; then
  params="$params --store-reward"
fi

# debug mode
if [[ ! -z $DEBUG_MODE ]]; then
  params="$params --gcmode archive --rpcapi db,eth,net,web3,personal,debug"
fi

# maxpeers
if [[ -z $MAXPEERS ]]; then
  MAXPEERS=25
fi

# dump
echo "dump: $IDENTITY $account $BOOTNODES"

set -x

exec tomo $params \
  --verbosity $VERBOSITY \
  --datadir $DATA_DIR \
  --keystore $KEYSTORE_DIR \
  --identity $IDENTITY \
  --maxpeers $MAXPEERS \
  --password ./password \
  --port 30303 \
  --txpool.globalqueue 5000 \
  --txpool.globalslots 5000 \
  --rpc \
  --rpccorsdomain "*" \
  --rpcaddr 0.0.0.0 \
  --rpcport 8545 \
  --rpcvhosts "*" \
  --ws \
  --wsaddr 0.0.0.0 \
  --wsport 8546 \
  --wsorigins "*" \
  --mine \
  --gasprice "250000000" \
  --targetgaslimit "84000000" \
  "$@"
