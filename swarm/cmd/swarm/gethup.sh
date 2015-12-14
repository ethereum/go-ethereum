#!/bin/bash
# Usage:
# bash /path/to/eth-utils/gethup.sh <datadir> <instance_name> <ip_addr>

root=$1  # base directory to use for datadir and logs
shift
id=$1  # double digit instance id like 00 01 02
shift
ip_addr=$1 # ip address to substitute
shift

# logs are output to a date-tagged file for each run , while a link is
# created to the latest, so that monitoring be easier with the same filename
# TODO: use this if GETH not set
# GETH=geth
# echo "ls -l $GETH"
# ls -l $GETH

# geth CLI params       e.g., (dd=04, run=09)
datetag=`date "+%c%y%m%d-%H%M%S"|cut -d ' ' -f 5`
datadir=$root/data/$id        # /tmp/eth/04
log=$root/log/$id.$datetag.log     # /tmp/eth/04.09.log
linklog=$root/log/$id.current.log     # /tmp/eth/04.09.log
stablelog=$root/log/$id.log     # /tmp/eth/04.09.log
password=$id            # 04
port=303$id              # 34504
bzzport=322$id              # 32204
rpcport=302$id            # 3204

mkdir -p $root/data
mkdir -p $root/enodes
mkdir -p $root/pids
mkdir -p $root/log
ln -sf "$log" "$linklog"
# if we do not have an account, create one
# will not prompt for password, we use the double digit instance id as passwd
# NEVER EVER USE THESE ACCOUNTS FOR INTERACTING WITH A LIVE CHAIN
keystoredir="$datadir/keystore/"
# echo "KeyStore dir: $keystoredir"
if [ ! -d "$keystoredir" ]; then
  # echo "create an account with password $id [DO NOT EVER USE THIS ON LIVE]"
  # mkdir -p $datadir/keystore
  $GETH --datadir $datadir --password <(echo -n $id) account new >/dev/null 2>&1
      # create account with password 00, 01, ...
  # note that the account key will be stored also separately outside
  # datadir
  # this way you can safely clear the data directory and still keep your key
  # under `<rootdir>/keystore/dd
  # LS=`ls $datadir/keystore`
  # echo $LS
  while [ ! -d "$keystoredir" ]; do
    echo "."
    ((i++))
    if ((i>10)); then break; fi
    sleep 1
  done
  # echo "copying keys $datadir/keystore $root/keystore/$id"
  mkdir -p $root/keystore/$id
  cp -R "$datadir/keystore/" $root/keystore/$id
fi

# # mkdir -p $datadir/keystore
# if [ ! -d "$datadir/keystore" ]; then
#   echo "copying keys $root/keystore/$id $datadir/keystore"
#   cp -R $root/keystore/$id/keystore/ $datadir/keystore/
# fi


# query node's enode url
if [ $ip_addr="" ]; then
  pattern='\d+\.\d+\.\d+\.\d+'
  ip_addr="[::]"
else
  pattern='\[\:\:\]'
fi

geth="$GETH --datadir $datadir --port $port"

# echo -n "enode for instance $id...  "
if [ ! "$GETH" = "" ] && [ ! -f $root/enodes/$id.enode ]; then
  cmd="$geth js <(echo 'console.log(admin.nodeInfo.enode); exit();') "
  # echo $cmd '2>/dev/null |grep enode | perl -pe "s/'$pattern'/'$ip_addr'/g" | perl -pe "s/^/\"/; s/\s*$/\"/;" > '$root/enodes/$id.enode
  eval $cmd 2>/dev/null |grep enode | perl -pe "s/$pattern/$ip_addr/g" | perl -pe "s/^/\"/; s/\s*\$/\"/;" > $root/enodes/$id.enode
fi
# cat  $root/enodes/$id.enode
echo

# copy cluster enodes list to node's static node list
# echo "copy cluster enodes list to node's static node list"
if [  -f $root/enodes.all ]; then
  cp $root/enodes.all $datadir/static-nodes.json
fi

if [ ! -f $root/pids/$id.pid ]; then
  # bring up node `dd` (double digit)
  # - using <rootdir>/<dd>
  # - listening on port 303dd, (like 30300, 30301, ...)
  # - with the account unlocked
  # - launching json-rpc server on port 81dd (like 8100, 8101, 8102, ...)
  # echo "BZZKEY=$geth account list|head -n1|perl -ne '/([a-f0-9]{40})/ && print \$1'"
  BZZKEY=`$geth account list|head -n1|perl -ne '/([a-f0-9]{40})/ && print \$1'`
  echo -n "starting instance $id ($BZZKEY @ $datadir )..."
  # echo "$geth \
  #   --identity=$id \
  #   --bzzaccount=$BZZKEY --bzzport=$bzzport \
  #   --unlock=$BZZKEY \
  #   --password=<(echo -n $id) \
  #   --rpc --rpcport=$rpcport --rpccorsdomain='*' $* \
  #   2>&1 | tee "$stablelog" > "$log" &  # comment out if you pipe it to a tty etc.
  # " >&2

  if [ -f $log ] && [ -f $root/stablelog ]; then
    cp $stablelog `cat $root/prevlog`
  fi
  echo $log > $root/prevlog

  $GETH --datadir=$datadir \
    --identity=$id \
    --bzzaccount=$BZZKEY --bzzport=$bzzport \
    --port=$port \
    --unlock=$BZZKEY \
    --password=<(echo -n $id) \
    --rpc --rpcport=$rpcport --rpccorsdomain='*' $* \
     > "$stablelog" 2>&1 &  # comment out if you pipe it to a tty etc.

     # wait until ready
  # pid=`ps auxwww|grep geth|grep "ty=$id"|grep -v grep|awk '{print $2}'`
  # echo "pid: $pid"
  # ps auxwww|grep geth|grep "ty=$id"|grep -v grep
  # echo $pid > $root/pids/$id.pid
  #echo $! > $root/pids/$id.pid
  while true; do
    $GETH --exec="net" attach ipc:$datadir/geth.ipc > /dev/null 2>&1 && break
    sleep 1
    echo -n "."
    if ((i++>10)); then
      echo "instance $id failed to start"
      exit 1
    fi
  done
  echo -n "started - "
  pid=`ps auxwww|grep geth|grep "ty=$id"|grep -v grep|awk '{print $2}'`
  echo "pid: $pid"
  # ps auxwww|grep geth|grep "ty=$id"|grep -v grep
  echo $pid > $root/pids/$id.pid
fi

# to bring up logs, uncomment
# tail -f $log
