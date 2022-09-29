#!/usr/bin/env sh

# Instructions:
# Execute `./getconfig.sh`, and follow the instructions displayed on the terminal
# The `*-config.toml` file will be created in the same directory as start.sh
# It is recommended to check the flags generated in config.toml


read -p "* Path to start.sh: " startPath
# check if start.sh is present
if [[ ! -f $startPath ]]
then
    echo "Error: start.sh do not exist."
    exit 1
fi
read -p "* Your validator address (e.g. 0xca67a8D767e45056DC92384b488E9Af654d78DE2), or press Enter to skip if running a sentry node: " ADD

echo "\nThank you, your inputs are:"
echo "Path to start.sh: "$startPath
echo "Address: "$ADD

confPath=${startPath%.sh}"-config.toml"
echo "Path to the config file: "$confPath
# check if config.toml is present
if [[ -f $confPath ]]
then
    echo "WARN: config.toml exists, data will be overwritten."
fi

tmpDir="$(mktemp -d -t ./temp-dir-XXXXXXXXXXX || oops "Can't create temporary directory")"
cleanup() {
    rm -rf "$tmpDir"
}
trap cleanup EXIT INT QUIT TERM

# SHA1 hash of `tempStart` -> `3305fe263dd4a999d58f96deb064e21bb70123d9`
sed 's/bor --/go run getconfig.go notYet --/g' $startPath > $tmpDir/3305fe263dd4a999d58f96deb064e21bb70123d9.sh
chmod +x $tmpDir/3305fe263dd4a999d58f96deb064e21bb70123d9.sh
$tmpDir/3305fe263dd4a999d58f96deb064e21bb70123d9.sh $ADD
rm $tmpDir/3305fe263dd4a999d58f96deb064e21bb70123d9.sh


sed -i '' "s%*%'*'%g" ./temp

# read the flags from `./temp`
dumpconfigflags=$(head -1 ./temp)

# run the dumpconfig command with the flags from `./temp`
command="bor dumpconfig "$dumpconfigflags" > "$confPath
bash -c "$command"

rm ./temp

if [[ -f ./tempStaticNodes.json ]]
then
    echo "JSON StaticNodes found"
    staticnodesjson=$(head -1 ./tempStaticNodes.json)
    sed -i '' "s%static-nodes = \[\]%static-nodes = \[\"${staticnodesjson}\"\]%" $confPath
    rm ./tempStaticNodes.json
elif [[ -f ./tempStaticNodes.toml ]]
then
    echo "TOML StaticNodes found"
    staticnodestoml=$(head -1 ./tempStaticNodes.toml)
    sed -i '' "s%static-nodes = \[\]%static-nodes = \[\"${staticnodestoml}\"\]%" $confPath
    rm ./tempStaticNodes.toml
else
    echo "neither JSON nor TOML StaticNodes found"
fi

if [[ -f ./tempTrustedNodes.toml ]]
then
    echo "TOML TrustedNodes found"
    trustednodestoml=$(head -1 ./tempTrustedNodes.toml)
    sed -i '' "s%trusted-nodes = \[\]%trusted-nodes = \[\"${trustednodestoml}\"\]%" $confPath
    rm ./tempTrustedNodes.toml
else
    echo "neither JSON nor TOML TrustedNodes found"
fi

# comment flags in $configPath that were not passed through $startPath
# SHA1 hash of `tempStart` -> `3305fe263dd4a999d58f96deb064e21bb70123d9`
sed "s%bor --%go run getconfig.go ${confPath} --%" $startPath > $tmpDir/3305fe263dd4a999d58f96deb064e21bb70123d9.sh
chmod +x $tmpDir/3305fe263dd4a999d58f96deb064e21bb70123d9.sh
$tmpDir/3305fe263dd4a999d58f96deb064e21bb70123d9.sh $ADD
rm $tmpDir/3305fe263dd4a999d58f96deb064e21bb70123d9.sh

exit 0
