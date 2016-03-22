echo "TEST swap/01:"
echo " two nodes that do not sync but have enough funds"
echo " after syncing, can retrieve content from each other"

dir=`dirname $0`
source $dir/../../cmd/swarm/test.sh

file=/tmp/test.file
mininginterval=120
key=/tmp/key
logargs="--verbosity=0 --vmodule='miner/*=0,swarm/services/*=6,swarm/swarm=6,common/chequebook/*=6,swarm/network/depo=6,swarm/network/forwarding=6'"
# logargs='--verbosity=6'


# swarm init 2 --mine --bzznosync $logargs
# swarm stop all

swarm start 00 --mine --bzznosync $logargs
swarm start 01 --mine --bzznosync $logargs

# echo "Mining some ether..."
# sleep $mininginterval

swarm attach 00 -exec "'eth.getBalance(eth.accounts[0])'"
swarm attach 01 -exec "'eth.getBalance(eth.accounts[0])'"
swarm attach 00 -exec "'eth.getBalance(bzz.info().Swap.Contract)'"
swarm attach 01 -exec "'eth.getBalance(bzz.info().Swap.Contract)'"
swarm attach 00 -exec "'chequebook.balance'"
swarm attach 01 -exec "'chequebook.balance'"

randomfile 10 > $file
swarm up 00 $file|tail -n1 > $key
swarm needs 01 $key $file
swarm info 01

swarm attach 00 -exec "'eth.getBalance(eth.accounts[0])'"
swarm attach 01 -exec "'eth.getBalance(eth.accounts[0])'"
swarm attach 00 -exec "'eth.getBalance(bzz.info().Swap.Contract)'"
swarm attach 01 -exec "'eth.getBalance(bzz.info().Swap.Contract)'"
swarm attach 00 -exec "'chequebook.balance'"
swarm attach 01 -exec "'chequebook.balance'"

# randomfile 20 > $file
# swarm up 01 $file|tail -n1 > $key
# swarm needs 00 $key $file

# swarm attach 00 -exec "'eth.getBalance(eth.accounts[0])'"
# swarm attach 01 -exec "'eth.getBalance(eth.accounts[0])'"
# swarm attach 00 -exec "'eth.getBalance(bzz.info().Swap.Contract)'"
# swarm attach 01 -exec "'eth.getBalance(bzz.info().Swap.Contract)'"
# swarm attach 00 -exec "'chequebook.balance'"
# swarm attach 01 -exec "'chequebook.balance'"

# randomfile 10 > $file
# swarm up 00 $file|tail -n1 > $key
# swarm needs 01 $key $file | tail -1| grep -ql "PASS" && echo "FAIL" || echo "PASS <3"

# randomfile 10 > $file
# swarm up 01 $file|tail -n1 > $key
# swarm needs 00 $key $file | tail -1| grep -ql "PASS" && echo "FAIL" || echo "PASS <3"

swarm stop all
