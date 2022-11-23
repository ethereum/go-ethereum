#!/bin/bash
set -e

delay=600

echo "Wait ${delay} seconds for state-sync..."
sleep $delay


balance=$(docker exec bor0 bash -c "bor attach /root/.bor/data/bor.ipc -exec 'Math.round(web3.fromWei(eth.getBalance(eth.accounts[0])))'")

if ! [[ "$balance" =~ ^[0-9]+$ ]]; then
    echo "Something is wrong! Can't find the balance of first account in bor network."
    exit 1
fi

echo "Found matic balance on account[0]: " $balance

if (( $balance <= 1001 )); then
    echo "Balance in bor network has not increased. This indicates that something is wrong with state sync."
    exit 1
fi

checkpointID=$(curl -sL http://localhost:1317/checkpoints/latest | jq .result.id)

if [ $checkpointID == "null" ]; then
    echo "Something is wrong! Could not find any checkpoint."
    exit 1
else
    echo "Found checkpoint ID:" $checkpointID
fi

echo "All tests have passed!"