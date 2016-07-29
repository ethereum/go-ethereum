#
swarm remote-run nodes.lst 'swarm netstatkill; swarm stop all;  rm -rf bin bzz .ethash /tmp/.eth* /tmp/swarm*; sudo reboot'
sleep 60
# update the control scripts on each host
swarm remote-update-scripts nodes.lst
# update the binary on each host node
swarm remote-update-bin nodes.lst

# spawn a 2-instance local cluster on each of our 10 host nodes
swarm remote-run nodes.lst 'swarm init 1; swarm netstatconf sworm; swarm netstatrun'

#collect enodes from all instances on all hosts
swarm remote-run nodes.lst 'swarm enode all' | tr -d '"' |grep -v running > enodes.lst

# swarm remote-run nodes.lst swarm addpeers <(enode)'

# copy enodes file to each host node
for node in `cat nodes.lst`; do scp enodes.lst $node:; done

# inject all enodes as peers to

swarm remote-run nodes.lst swarm addpeers all enodes.lst

# to bootstrap with  i

# restart an instance on the first two  host nodes to get the blockchain rolling
swarm remote-run <(head -2 nodes.lst) swarm restart 00 --mine

