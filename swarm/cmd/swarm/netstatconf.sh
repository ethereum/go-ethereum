# !/bin/bash
# bash netstatconf.sh <number_of_nodes> <name_prefix> <ws_server> <ws_secret>

# sets up a eth-net-intelligence app.json for a local ethereum network cluster of nodes
# - <number_of_nodes> is the number of nodes in the cluster
# - <name_prefix> is a prefix for the node names as will appear in the listing
# - <ws_server> is the eth-netstats server
# - <ws_secret> is the eth-netstats secret
#

# open http://localhost:3301

N=$1
shift
name_prefix=$1
shift
ws_server=$1
shift
ws_secret=$1
shift

echo -e "["

for ((i=0;i<N;++i)); do
    id=`printf "%02d" $i`
    single_template="  {\n    \"name\"        : \"$name_prefix-$i\",\n    \"cwd\"         : \".\",\n    \"script\"      : \"app.js\",\n    \"log_date_format\"   : \"YYYY-MM-DD HH:mm Z\",\n    \"merge_logs\"    : false,\n    \"watch\"       : false,\n    \"exec_interpreter\"  : \"node\",\n    \"exec_mode\"     : \"fork_mode\",\n    \"env\":\n    {\n      \"NODE_ENV\"    : \"production\",\n      \"RPC_HOST\"    : \"localhost\",\n      \"RPC_PORT\"    : \"622$id\",\n      \"INSTANCE_NAME\"   : \"$name_prefix-$i\",\n      \"WS_SERVER\"     : \"$ws_server\",\n      \"WS_SECRET\"     : \"$ws_secret\",\n    }\n  }"

    endline=""
    if [ "$i" -ne "$N" ]; then
        endline=","
    fi
    echo -e "$single_template$endline"
done

echo "]"
