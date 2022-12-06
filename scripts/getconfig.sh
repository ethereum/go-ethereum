#!/bin/bash
set -e

# Instructions:
# Execute `./getconfig.sh`, and follow the instructions displayed on the terminal
# The `*-config.toml` file will be created in the same directory as start.sh
# It is recommended to check the flags generated in config.toml

# Some checks to make commands OS independent
OS="$(uname -s)"
MKTEMPOPTION=
SEDOPTION= ## Not used as of now (TODO)
shopt -s nocasematch; if [[ "$OS" == "darwin"* ]]; then
  SEDOPTION="''"
  MKTEMPOPTION="-t"
fi

read -p "* Path to start.sh: " startPath
# check if start.sh is present
if [[ ! -f $startPath ]]
then
    echo "Error: start.sh do not exist."
    exit 1
fi
read -p "* Your validator address (e.g. 0xca67a8D767e45056DC92384b488E9Af654d78DE2), or press Enter to skip if running a sentry node: " ADD

if [[ -f $HOME/.bor/data/bor/static-nodes.json ]]
then
cp $HOME/.bor/data/bor/static-nodes.json ./static-nodes.json
else
read -p "* You dont have '~/.bor/data/bor/static-nodes.json' file. If you want to use static nodes, enter the path to 'static-nodes.json' here (press Enter to skip): " STAT
if [[ -f $STAT ]]; then cp $STAT ./static-nodes.json; fi
fi

printf "\nThank you, your inputs are:\n"
echo "Path to start.sh: "$startPath
echo "Address: "$ADD

confPath=${startPath%.sh}"-config.toml"
echo "Path to the config file: "$confPath
# check if config.toml is present
if [[ -f $confPath ]]
then
    echo "WARN: config.toml exists, data will be overwritten."
fi
printf "\n"

tmpDir="$(mktemp -d $MKTEMPOPTION ./temp-dir-XXXXXXXXXXX || oops "Can't create temporary directory")"
cleanup() {
    rm -rf "$tmpDir"
}
trap cleanup EXIT INT QUIT TERM

# SHA1 hash of `tempStart` -> `3305fe263dd4a999d58f96deb064e21bb70123d9`
sed 's/bor --/go run getconfig.go notYet --/g' $startPath > $tmpDir/3305fe263dd4a999d58f96deb064e21bb70123d9.sh
chmod +x $tmpDir/3305fe263dd4a999d58f96deb064e21bb70123d9.sh
$tmpDir/3305fe263dd4a999d58f96deb064e21bb70123d9.sh $ADD
rm $tmpDir/3305fe263dd4a999d58f96deb064e21bb70123d9.sh

shopt -s nocasematch; if [[ "$OS" == "darwin"* ]]; then
    sed -i '' "s%*%'*'%g" ./temp
else
    sed -i "s%*%'*'%g" ./temp
fi

# read the flags from `./temp`
dumpconfigflags=$(head -1 ./temp)

# run the dumpconfig command with the flags from `./temp`
command="bor dumpconfig "$dumpconfigflags" > "$confPath
bash -c "$command"

rm ./temp

printf "\n"

if [[ -f ./tempStaticNodes.json ]]
then
    echo "JSON StaticNodes found"
    staticnodesjson=$(head -1 ./tempStaticNodes.json)
    shopt -s nocasematch; if [[ "$OS" == "darwin"* ]]; then
        sed -i '' "s%static-nodes = \[\]%static-nodes = \[\"${staticnodesjson}\"\]%" $confPath
    else
        sed -i "s%static-nodes = \[\]%static-nodes = \[\"${staticnodesjson}\"\]%" $confPath
    fi
    rm ./tempStaticNodes.json
elif [[ -f ./tempStaticNodes.toml ]]
then
    echo "TOML StaticNodes found"
    staticnodestoml=$(head -1 ./tempStaticNodes.toml)
    shopt -s nocasematch; if [[ "$OS" == "darwin"* ]]; then
        sed -i '' "s%static-nodes = \[\]%static-nodes = \[\"${staticnodestoml}\"\]%" $confPath
    else
        sed -i "s%static-nodes = \[\]%static-nodes = \[\"${staticnodestoml}\"\]%" $confPath
    fi
    rm ./tempStaticNodes.toml
else
    echo "neither JSON nor TOML StaticNodes found"
fi

if [[ -f ./tempTrustedNodes.toml ]]
then
    echo "TOML TrustedNodes found"
    trustednodestoml=$(head -1 ./tempTrustedNodes.toml)
    shopt -s nocasematch; if [[ "$OS" == "darwin"* ]]; then
        sed -i '' "s%trusted-nodes = \[\]%trusted-nodes = \[\"${trustednodestoml}\"\]%" $confPath
    else
        sed -i "s%trusted-nodes = \[\]%trusted-nodes = \[\"${trustednodestoml}\"\]%" $confPath
    fi
    rm ./tempTrustedNodes.toml
else
    echo "neither JSON nor TOML TrustedNodes found"
fi

if [[ -f ./tempHTTPTimeoutsReadTimeout.toml ]]
then
    echo "HTTPTimeouts.ReadTimeout found"
    read=$(head -1 ./tempHTTPTimeoutsReadTimeout.toml)
    shopt -s nocasematch; if [[ "$OS" == "darwin"* ]]; then
        sed -i '' "s%read = \"30s\"%read = \"${read}\"%" $confPath
    else
        sed -i "s%read = \"30s\"%read = \"${read}\"%" $confPath
    fi
    rm ./tempHTTPTimeoutsReadTimeout.toml
fi

if [[ -f ./tempHTTPTimeoutsWriteTimeout.toml ]]
then
    echo "HTTPTimeouts.WriteTimeout found"
    write=$(head -1 ./tempHTTPTimeoutsWriteTimeout.toml)
    shopt -s nocasematch; if [[ "$OS" == "darwin"* ]]; then
        sed -i '' "s%write = \"30s\"%write = \"${write}\"%" $confPath
    else
        sed -i "s%write = \"30s\"%write = \"${write}\"%" $confPath
    fi
    rm ./tempHTTPTimeoutsWriteTimeout.toml
fi

if [[ -f ./tempHTTPTimeoutsIdleTimeout.toml ]]
then
    echo "HTTPTimeouts.IdleTimeout found"
    idle=$(head -1 ./tempHTTPTimeoutsIdleTimeout.toml)
    shopt -s nocasematch; if [[ "$OS" == "darwin"* ]]; then
        sed -i '' "s%idle = \"2m0s\"%idle = \"${idle}\"%" $confPath
    else
        sed -i "s%idle = \"2m0s\"%idle = \"${idle}\"%" $confPath
    fi
    rm ./tempHTTPTimeoutsIdleTimeout.toml
fi

if [[ -f ./tempHTTPTimeoutsTrieTimeout.toml ]]
then
    echo "Eth.TrieTimeout found"
    timeout=$(head -1 ./tempHTTPTimeoutsTrieTimeout.toml)
    shopt -s nocasematch; if [[ "$OS" == "darwin"* ]]; then
        sed -i '' "s%timeout = \"1h0m0s\"%timeout = \"${timeout}\"%" $confPath
    else
        sed -i "s%timeout = \"1h0m0s\"%timeout = \"${timeout}\"%" $confPath
    fi
    rm ./tempHTTPTimeoutsTrieTimeout.toml
fi

printf "\n"

# comment flags in $configPath that were not passed through $startPath
# SHA1 hash of `tempStart` -> `3305fe263dd4a999d58f96deb064e21bb70123d9`
sed "s%bor --%go run getconfig.go ${confPath} --%" $startPath > $tmpDir/3305fe263dd4a999d58f96deb064e21bb70123d9.sh
chmod +x $tmpDir/3305fe263dd4a999d58f96deb064e21bb70123d9.sh
$tmpDir/3305fe263dd4a999d58f96deb064e21bb70123d9.sh $ADD
rm $tmpDir/3305fe263dd4a999d58f96deb064e21bb70123d9.sh

if [[ -f $HOME/.bor/data/bor/static-nodes.json ]]
then
rm ./static-nodes.json
fi

exit 0
