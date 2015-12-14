echo "TEST swap/01:"
echo " two nodes that do not sync but have enough funds"
echo " can retrieve content from each other"

dir=`dirname $0`
source $dir/../../cmd/swarm/test.sh

file=/tmp/test.file
mininginterval=120
key=/tmp/key
logargs="--verbosity=0 --vmodule='swarm/*=6'"
# logargs='--verbosity=6'


swarm init 2 --mine --bzznosync $logargs

echo "Mining some ether..."
sleep $mininginterval

randomfile 10 > $file
swarm up 00 $file|tail -n1 > $key
swarm needs 01 $key $file
swarm info 01

randomfile 200000 > $file
swarm up 00 $file |tail -n1 > $key
swarm needs 01 $key $file

randomfile 10 > $file
swarm up 00 $file|tail -n1 > $key
swarm needs 01 $key $file | tail -1| grep -ql "PASS" && echo "FAIL" || echo "PASS <3"

swarm status 00
swarm status 01

randomfile 10 > $file
swarm up 01 $file|tail -n1 > $key
swarm needs 00 $key $file | tail -1| grep -ql "PASS" && echo "FAIL" || echo "PASS <3"

swarm status 00
swarm status 01

swarm stop all
