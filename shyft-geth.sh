#!/bin/bash

if [[ $# -lt 1 ]]; then
    echo Shyft-Geth: No flags detected, refer to readme for available commands.
    exit 1
fi

illegalCommands=()
while [[ $# -gt 0 ]]
do
key="$1"

case $key in
    --setup)
    sh ./shyft-cli/setup.sh
    shift # past argument
    ;;
    --start)
    sh ./shyft-cli/startShyftGeth.sh
    shift # past argument
    ;;
    --js)
    sh ./shyft-cli/runJs.sh ./shyft-cli/web3/$2.js
    shift # past argument
    shift # past argument
    ;;
    --reset)
    sh ./shyft-cli/shyftFullReset.sh
    shift # past argument
    ;;
    *)    # unknown option
    illegalCommands+=("$1") # save it in an array for later
    shift # past argument
    ;;
esac
done

if [[ "${#illegalCommands[@]}" -gt "0" ]]; then
    echo Shyft-Geth: The following commands are not supported: "${illegalCommands[*]}"
fi

