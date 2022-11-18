#!/bin/bash
set -e

balanceInit=$(docker exec bor0 bash -c "bor attach /root/.bor/data/bor.ipc -exec 'Math.round(web3.fromWei(eth.getBalance(eth.accounts[0])))'")

stateSyncFound="false"
checkpointFound="false"

while true
do
  
    balance=$(docker exec bor0 bash -c "bor attach /root/.bor/data/bor.ipc -exec 'Math.round(web3.fromWei(eth.getBalance(eth.accounts[0])))'")

    if ! [[ "$balance" =~ ^[0-9]+$ ]]; then
        echo "Something is wrong! Can't find the balance of first account in bor network."
        exit 1
    fi

    if (( $balance > $balanceInit )); then
        stateSyncFound="true"   
    fi

    checkpointID=$(curl -sL http://localhost:1317/checkpoints/latest | jq .result.id)

    if [ $checkpointID != "null" ]; then
        checkpointFound="true"
    fi

    if [ $stateSyncFound == "true" ]  && [ $checkpointFound == "true" ]; then
        break
    fi    

done
echo "Both state sync and checkpoint went through. All tests have passed!"
