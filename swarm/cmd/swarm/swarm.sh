# !/bin/bash
# bash cluster <root> <network_id> <number_of_nodes>  <runid> <local_IP> [[params]...]
# https://github.com/ethereum/go-ethereum/wiki/Setting-up-monitoring-on-local-cluster

# sets up a local ethereum network cluster of nodes
# - <number_of_nodes> is the number of nodes in cluster
# - <root> is the root directory for the cluster, the nodes are set up
#   with datadir `<root>/<network_id>/00`, `<root>/ <network_id>/01`, ...
# - new accounts are created for each node
# - they launch on port 30300, 30301, ...
# - they star rpc on port 8100, 8101, ...
# - by collecting the nodes nodeUrl, they get connected to each other
# - if enode has no IP, `<local_IP>` is substituted
# - if `<network_id>` is not 0, they will not connect to a default client,
#   resulting in a private isolated network
# - the nodes log into `<root>/<network_id>/00.<runid>.log`, `<root>/<network_id>/01.<runid>.log`, ...
# - The nodes launch in mining mode
# - the cluster can be killed with `killall geth` (FIXME: should record PIDs)
#   and restarted from the same state
# - if you want to interact with the nodes, use rpc
# - you can supply additional params on the command line which will be passed
#   to each node, for instance `-mine`

if [ "$GETH" = "" ]; then
  echo "env var GETH not set "
  exit 1
fi

srcdir=`dirname $0`

root=$1
shift
network_id=$1
shift
cmd=$1
shift
# ip_addr=`curl ipecho.net/plain 2>/dev/null;echo `

# echo "external IP: $ip_addr"
swarmoptions='--dev --maxpeers=40 --shh=false --nodiscover'
tmpdir=/tmp

function attach {
  id=$1
  shift
  echo "attaching console to instance $id"
  cmd="$GETH $* attach ipc:$root/$network_id/data/$id/geth.ipc"
  echo $cmd
  eval $cmd
}

function log {
  id=$1
  shift
  echo "streaming logs for instance $id"
  cmd="tail -f $root/$network_id/log/$id.log"
  echo $cmd
  eval $cmd
}

function cleanlog {
  id=$1
  shift
  if [ $id = "all" ]; then
    echo "remove logs for all instances"
    rm -rf "$root/$network_id/log/"
  else
    echo "remove logs for instance $id"
    rm -rf $root/$network_id/log/$id*
  fi
}

function cleanbzz {
  id=$1
  shift
  if [ $id = "all" ]; then
    echo "remove bzz data for all instances"
    rm -rf $root/$network_id/data/*/bzz
  else
    echo "remove bzz data for instance $id"
    rm -rf "$root/$network_id/data/$id"
  fi
}

function less {
  id=$1
  shift
  echo "viewing logs for instance $id"
  cmd="/usr/bin/less $root/$network_id/log/$id.log"
  echo $cmd
  eval $cmd
}

function start {
  id=$1
  shift
  # echo -n "starting instance $id - "
  cmd="bash $srcdir/gethup.sh $root/$network_id/ $id '$ip_addr' --networkid=$network_id $swarmoptions $*"
  # echo "pid="`cat $root/$network_id/pids/$id.pid`
  # echo $cmd
  eval $cmd
}

function stop {
  id=$1
  shift
  if [ $id = "all" ]; then
    procs=`cat $root/$network_id/pids/*.pid 2>/dev/null |perl -pe 's/^\s+//;s/\s+\\$//;s/\s+/\n/g'`
    # echo "stopping processes $procs"
    for p in $procs; do
      shutdown $p
    done
    rm -rf $root/$network_id/pids/*
  else
    pid=$root/$network_id/pids/$id.pid
    if [ -f $pid ]; then
      echo "stopping instance $id, pid="`cat $pid`
      shutdown `cat $pid`
      rm $pid
    fi
  fi
  # ps auxwww|grep geth|grep bzz|grep -v grep
}

function shutdown {
  echo -n "stopping $1..."
  kill -2 $1
  while true   ;do
    ps auxwww|grep geth|grep -v grep|awk '{print $2}'|grep -ql $1 || break
    sleep 1
  done
  echo "stopped"
}

function restart {
  id=$1
  shift
  stop $id
  start $id $*
}

function init {
  stop all
  killall geth
  reset all
  cluster $*
  enode all
  connect all
}

function reset {
  id=$1
  shift
  if [ $id = "all" ]; then
    rm -rf $root/$network_id
  else
    rm -rf$root/$network_id/*/$id*
  fi

}

function enode {
  dir=$root/$network_id
  id=$1
  shift
  if [ $id = "all" ]; then
    N=`ls -1 $dir/enodes/|wc -l`
    enodes=$dir/enodes.all
    rm -f $enodes
    # build a static nodes(-like) list of all enodes of the local cluster
    echo "[" >> $enodes
    for ((i=0;i<N;++i)); do
      id=`printf "%02d" $i`
      enode=$dir/enodes/$id.enode
      enode $id
      if [ -f "$enode" ] && [ ! -z "$enode" ]; then
        cat "$enode" >> $enodes
        echo "," >> $enodes
      fi
    done
    echo "\"\"]" >> $enodes
    cmd=$dir/connect.js
    for ((i=0;i<N;++i)); do
      id=`printf "%02d" $i`
      enode=$dir/enodes/$id.enode
      if [ -f "$enode" ] && [ ! -z "$enode" ]; then
        echo -n "admin.addPeer(" >> $cmd
        cat "$enode" >> $cmd
        echo ");" >> $cmd
      fi
    done
  else
    enode=$dir/enodes/$id.enode
    attach $id --exec "'console.log(admin.nodeInfo.enode)'" |head -2 |tail -1| perl -pe 's/^/"/;s/$/"/'|perl -pe 's/\s*$//' > $enode
    # cat $enode
  fi

}

function connect {
  dir=$root/$network_id
  id=$1
  shift
  if [ $id = "all" ]; then
    for ((i=0;i<N;++i)); do
      id=`printf "%02d" $i`
      connect $id
    done
  else
    cmd="$GETH --preload $dir/connect.js --exec '\"admin.peers\"' attach ipc:$root/$network_id/data/$id/geth.ipc $dir/connect.js"
    # echo $cmd
    eval $cmd
    cat $dir/connect.js
  fi
}

function cluster {
  N=$1
  shift
  echo "launching cluster of $N instances"
  # cmd="bash $srcdir/gethcluster.sh $root $network_id $N '' $swarmoptions $*"
  # echo $cmd
  # eval $cmd
  dir=$root/$network_id
  mkdir -p $dir/data
  mkdir -p $dir/enodes
  mkdir -p $dir/pids
  mkdir -p $dir/log

  for ((i=0;i<N;++i)); do
    id=`printf "%02d" $i`
    mkdir -p $dir/data/$id
    echo "launching node $i/$N ---> tail -f $dir/log/$id.log"
    start $id $vmodule $*
  done
}


function needs {
  id=$1
  keyfile=$2
  target=$3
  dir=`dirname $3`
  dest=$tmpdir/down
  mkdir -p $dest
  file=$dest/`basename $target`
  rm -f $file
  echo -n "waiting for root hash in '$keyfile'..."
  while true; do
   if [ -f $keyfile ] && [ ! -z $keyfile ]; then
    break
   fi
   sleep 1
   echo -n "."
  done
  key=`cat $keyfile|tr -d \"`
  echo " => $key"
  download $id $key $dest && cmp --silent $file $target && echo "PASS" || echo "FAIL"
  # && ls -l $keyfile $file $target
}


function up { #port, file
  echo "Upload file '$2' to node $1... " 1>&2
  file=`basename $2`
  attach $1 "--exec 'bzz.upload(\"$2\", \"$file\")'"|tail -n1> /tmp/key
  # key=`bash swarm/cmd/bzzup.sh $2 86$1`
  cat /tmp/key
}

function download {
  echo "download '$2' from node $1 to '$3'"
  # echo attach $1 "--exec 'bzz.download(\"$2\", \"$3\")'"
  attach $1 "--exec 'bzz.download(\"$2\", \"$3\")'" > /dev/null
}


function down {
  echo -n "Download hash '$2' from node $1... "
  # echo "wget -O- http://localhost:86$1/$2 > /dev/null 2>&1 && echo 'got it' || echo 'not found'"
  # wget -O- http://localhost:86$1/$2 > /dev/null 2>&1 && echo "got it" || echo "not found"
  while true; do
    attach $1 "--exec 'bzz.get(\"$2\")'" 2> /dev/null |grep -qil "status" && break
    sleep 1
    echo -n "."
    if ((i++>10)); then
      echo "not found"
      return
    fi
  done
  echo "found OK"
}

function clean { #index
  echo "Clean up for $1"
  rm -rf $root/$network_id/data/$1/{bzz/*/chunks,bzz/*/requests/,bzz/*/bzz-peers.json,chaindata,nodes}
}

function info {
  echo "swarm node information"
  echo "ROOTDIR: $root"
  echo "DATADIR: $root/$network_id/data/$1"
  echo "LOGFILE: $root/$network_id/log/$1.log"
  echo "HTTPAPI: http://localhost:322$1"
  echo "ETHPORT: 303$1"
  echo "RPCPORT: 302$1"
  echo "ACCOUNT:" 0x`ls -1 $root/$network_id/data/$1/bzz`
  echo "CHEQUEB:" `cat $root/$network_id/data/$1/bzz/*/config.json|grep Contract|awk -F\" '{print $4}'`
  echo "ROOTDIR: $root"
  echo "DATADIR: $root/$network_id/data/$1"
  echo "LOGFILE: $root/$network_id/log/$1.log"
}


function status {
  attach 00 -exec "'console.log(eth.getBalance(eth.accounts[0])); console.log(eth.getBalance(bzz.info().Swap.Contract)); console.log(chequebook.balance)'"
}

function netstatconf {
  begin=$1
  N=$2
  name_prefix=$3
  ws_server=$4
  ws_secret=$5
  conf="$root/$network_id/$name_prefix.netstat.json"

  echo "writing netstat conf for cluster $name_prefix to $conf"

  echo -e "[" > $conf

  for ((i=$begin;i<$start+$N;++i)); do
      id=`printf "%02d" $i`
      single_template="  {\n    \"name\"        : \"$name_prefix-$i\",\n    \"cwd\"         : \".\",\n    \"script\"      : \"app.js\",\n    \"log_date_format\"   : \"YYYY-MM-DD HH:mm Z\",\n    \"merge_logs\"    : false,\n    \"watch\"       : false,\n    \"exec_interpreter\"  : \"node\",\n    \"exec_mode\"     : \"fork_mode\",\n    \"env\":\n    {\n      \"NODE_ENV\"    : \"production\",\n      \"RPC_HOST\"    : \"localhost\",\n      \"RPC_PORT\"    : \"302$id\",\n      \"INSTANCE_NAME\"   : \"$name_prefix-$i\",\n      \"WS_SERVER\"     : \"$ws_server\",\n      \"WS_SECRET\"     : \"$ws_secret\",\n    }\n  }"

      endline=""
      if (($i<$N-1)); then
      # if [ "$i" -ne "$N" ]; then
          endline=","
      fi
      echo -e "$single_template$endline" >> $conf
  done

  echo "]" >> $conf
}

function remote-update-scripts {
  scriptdir=$1
  remotes=$2
  cd $GETH_DIR
  for remote in `cat $remotes|grep -v '^#'`; do echo "updating scripts on $remote..."; ssh $remote mkdir -p bin && scp -r $scriptdir/* $remote:bin/; done
}

function remote-update-bin {
  remotes=$1
  remote-update-scripts $GETH_DIR/swarm/cmd/swarm/ $remotes
  for remote in `cat $remotes|grep -v '^#'`; do  echo "updating binary on $remote..."; scp -r $GETH_DIR/geth $remote:bin/; done
}

function remote-run {
  remotes=$1
  shift
  for remote in `cat $remotes|grep -v '^#'`; do echo "running on $remote..."; ssh $remote ". ~/bin/env.sh; $*"; done
}

function update-src {
  branch=$1
  echo "cd $GETH_DIR &&  git remote update && git reset --hard $branch"
  (cd $GETH_DIR &&  git remote update && git reset --hard $branch)
}

function netstatrun {
  cd ~/eth-net-intelligence-api
  pm2 kill
  pm2 start $root/$network_id/*.netstat.json
}


case $cmd in
  "info" )
    info $*;;
  "enode" )
    enode $*;;
  "connect" )
    connect $*;;
  "status" )
    status $*;;
  "clean" )
    clean $*;;
  "needs" )
    needs $*;;
  "up" )
    up $*;;
  "down" )
    down $*;;
  "download" )
    download $*;;
  "init" )
    init $*;;
  "start" )
    start $*;;
  "stop" )
    stop $* ;;
  "restart" )
    restart $*;;
  "reset" )
    reset $*;;
  "cluster" )
    cluster $*;;
  "attach" )
    attach $*;;
  "cleanbzz" )
    cleanbzz $*;;
  "cleanlog" )
    cleanlog $*;;
  "log" )
    log $*;;
  "less" )
    less $*;;
  "remote-update-scripts" )
    remote-update-scripts $*;;
  "remote-update-bin" )
    remote-update-bin $*;;
  "update-src" )
    update-src $*;;
  "remote-run" )
    remote-run $*;;
  "netstatconf" )
    netstatconf  $*;;
  "netstatrun" )
    netstatrun  $*;;

esac
