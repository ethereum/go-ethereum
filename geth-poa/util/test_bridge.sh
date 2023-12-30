#!/bin/sh
set -x
set -e

# TODO: Stress test bridge back to sepolia, including not having enough balance on sidechain.

read -p "Has the whitelist contract been deployed, updated with hypERC20 addr, and have router addresss been pasted into this file? (y/n): " answer
if [ "$answer" = "y" ]; then
    echo "Continuing with bridging process..."
else
    echo "Exiting..."
    exit 1
fi

# Test account, this must be funded on Sepolia
# Address:     0xa43b806D2f09AE94dfa38bc00d6F75426D274540
# Private key: 0x8b21e3bc5c26d3327109d341d121fbfb7cb79c95fba5eb2f8c064f87332df7dd
ADDRESS=0xa43b806D2f09AE94dfa38bc00d6F75426D274540
PRIVATE_KEY=0x8b21e3bc5c26d3327109d341d121fbfb7cb79c95fba5eb2f8c064f87332df7dd

# "make print-warp-deploy" prints these contract addrs
SEPOLIA_ROUTER=0xBc96ff3E2458D4c6dDCE0DE5E87bC4B254626a64
SIDECHAIN_ROUTER=0x4b2DC8A5C4da51f821390AbD2B6fe8122BC6fA97

SEPOLIA_URL=https://ethereum-sepolia.publicnode.com
SIDECHAIN_URL=http://69.67.151.95:8545

# Store initial sidechain native balance
SIDECHAIN_BALANCE=$(cast balance --rpc-url $SIDECHAIN_URL $ADDRESS)

# sepolia -> dest chain (account must be funded on sepolia)
cast call --rpc-url $SEPOLIA_URL $SEPOLIA_ROUTER "quoteGasPayment(uint32)" "17864" 
# Above returns 1 wei, therefore ether value is 1 wei larger than value function argument
cast send --rpc-url $SEPOLIA_URL --private-key $PRIVATE_KEY $SEPOLIA_ROUTER "transferRemote(uint32,bytes32,uint256)" "17864" "0x000000000000000000000000a43b806d2f09ae94dfa38bc00d6f75426d274540" "5000000000000000" --value 5000000000000001wei

# Block until native balance is incremented
MAX_RETRIES=20
RETRY_COUNT=0
while [ $(printf '%d' $(cast balance --rpc-url $SIDECHAIN_URL $ADDRESS)) -eq $(printf '%d' $SIDECHAIN_BALANCE) ]
do
    echo "Waiting for native balance to increment..."
    sleep 5

    RETRY_COUNT=$((RETRY_COUNT + 1))

    if [ $RETRY_COUNT -ge $MAX_RETRIES ]; then
        echo "Maximum retries reached"
        break
    fi
done

# Store sepolia balance
SEPOLIA_BALANCE=$(cast balance --rpc-url $SEPOLIA_URL $ADDRESS)

# Send some sidechain ether from genesis funded account to pay for tx fees on sidechain
cast send --rpc-url $SIDECHAIN_URL --private-key 0xc065f4c9a6dda0785e2224f5af8e473614de1c029acf094f03d5830e2dd5b0ea $ADDRESS --value 0.1ether 

sleep 5

# dest chain -> sepolia (account must be funded on dest chain)
cast call --rpc-url $SIDECHAIN_URL $SIDECHAIN_ROUTER "quoteGasPayment(uint32)" "11155111" 
# Above returns 0 wei, therefore ether value is same as function argument
cast send \
    --rpc-url $SIDECHAIN_URL \
    --private-key $PRIVATE_KEY \
    $SIDECHAIN_ROUTER "transferRemote(uint32,bytes32,uint256)" \
     "11155111" \
     "0x000000000000000000000000a43b806d2f09ae94dfa38bc00d6f75426d274540" \
     "5000000000000000" \
     --value 500000000000000wei

# Block until sepolia balance is incremented
MAX_RETRIES=30
RETRY_COUNT=0
while [ $(cast balance --rpc-url $SEPOLIA_URL $ADDRESS) -eq $SEPOLIA_BALANCE ]
do
    echo "Waiting for sepolia balance to increment..."
    sleep 5

    RETRY_COUNT=$((RETRY_COUNT + 1))

    if [ $RETRY_COUNT -ge $MAX_RETRIES ]; then
        echo "Maximum retries reached"
        break
    fi
done

echo "Success"
