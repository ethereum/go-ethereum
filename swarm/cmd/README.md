
# install and setup swarm

swarm is developed on a branch of the ethereum/go-ethereum repo
at this stage of the project there is no packages or binary distro, you need to a dev environment and compile from source.
[This document spells out a complete server setup on ubuntu linux](https://gist.github.com/zelig/74eb365752ceaacf15e860fb80eacb3e) including git/ssh/screen config, golang and compilation, node/npm and network monitoring (might contain a few bits that are tangential to swarm).

Assuming you got your setup working, you will use the `swarm` command line tool to control your swarm of instances.
This command line tool is at the moment geared towards developers and testing.
It is likely that it will be replaced by two different tools, one for devel/testing and one for end users


The command can be used to update the code

```shell
swarm update upstream/swarm
```

Then compile with

```shell
godep go build -v ./cmd/geth
```

Make sure you have `GOPATH` variable set and also that the `swarm` executable is in your PATH.
These environment variables are relevant and set to the following defaults.
Make sure you are happy with them, otherwise change them, in which case best to put these lines in your `~/.profile`.

```
export GETH_DIR=$GOPATH/src/github.com/ethereum/go-ethereum
export GETH=$GETH_DIR/geth
export SWARM_DIR=~/bzz
export SWARM_NETWORK_ID=322
```

* `GETH_DIR` points to your git working copy (given `GOPATH` its standardly under `$GOPATH/src/github.com/ethereum/go-ethereum`)
* `GETH` points to the `geth` executable compiled from the swarm branch. If you have systemwide install or use multiple geths you may need to change this, otherwise it is assumed you compile to the working copy of the repo.
* `SWARM_DIR` is the root directory for all swarm related stuff: logs, configs, as well as geth datadirs, make sure this dir is on a device with sufficient disk space
* `SWARM_NETWORK_ID`: this is by default the network id of the swarm testnet. If you run your own swarm, you need to change it, choose a number that is not likely chosen by others to avoid others joining you.

# Deploying and remote control

the swarm command supports remote update and remote control of your instances.
In our setting we assume you want to run a cluster of potentially remote swarm nodes each running a local cluster of instances
The only assumption is that you have (passwordless) ssh access set up to your swarm servers.
Assume `nodes.lst` is a list of nodes in the format of `username@ip` one per line. blank lines and lines commented out with `#` are ignored.


This copies the  scripts found in  `swarm/cmd/swarm` on all remote  nodes listed in `nodes.lst`

```
swarm remote-update-scripts nodes.lst
```

If you just want to deploy a locally compiled binaries to all your remote nodes, this will fail if the remote instances are running, so make sure you stop them beforehand

```shell
swarm remote-run nodes.lst swarm stop all
swarm remote-update-bin nodes.lst
```



Once  you deployed the executables to the nodes, you can control them all with one command. For instance the following line initialises a cluster of two test swarm instances on each remote node.
Watch out, this will wipe your storage and  all swarm related data

```shell
swarm remote-run nodes.lst swarm init 2
```


To (re) start a particular instance on a specific remote node with alternative options (for instance mining and different logging verbosity), you can just:

```shell
swarm remote-run cicada@3.3.0.1 'swarm restart 01 --mine --verbosity=0 --vmodule=swarm/*=5'
```

# Logging

To check logs

```shell
swarm log 00 # taillog flow
swarm remote-run cicada@3.3.0.1 swarm log 00
```

You can view the log with a pager for an instance with

```
swarm viewlog 00
```

Logs are preserved and viewable with the above commands even when nodes are offline
Each new run logs to a different file

To purge logs

```
swarm cleanlog 01
```

To remove all logs on all nodes:

```
swarm remote-run nodes.lst swarm cleanlog all
```

# upload and dowload

upload and download via a running local instance

```shell
swarm up 00 /path/to/file/or/directory
swarm down 01 hash /path/to/destination
```

upload via remote swarm proxy or public gateway

```shell
swarm remote-up gateway-url /path/to/file/or/directory
wget -O- gateway-url/bzz:/swarm-url
```


# Further examples

```shell
# start with updaing
swarm update chambers

# display CLI options given to geth used to launch swarm instance 02
swarm options 02

# restart swarm instance 00 with alternatiev options
swarm restart 00 --mine --bzznosync --verbosity=0 --vmodule=swarm/*=6

# attach console to a running swarm instance
swarm attach 00

# execute a command; e.g., start mining on a running instance
swarm execute 00 'miner.stop(1)'

# display static info about a instance (even if its offline)
swarm info 00

# displays the enode url of a running instance
swarm enode 01

# add peers to a running swarm instance
swarm addpeers 00 "enode://1033c1cada...@3.3.0.1:30301"

# to compile a list of enodes from all instances on all remote nodes:
swarm remote-run nodes.lst 'swarm enode all' > enodes.lst

# to add all peers to all instances on  each node
for node in `cat nodes.lst|grep -v '^#'`; do scp enodes.lst $node:; done
swarm remote-run nodes.lst 'swarm addpeers enodes.lst'

# if you run a local network and  your nodes do not listen to external IPs
swarm  remote-run pivot.lst 'swarm restart all'

# to add just one or a few guardians and let the network bootstrap
# swarm remote-run 'swarm enode all'
swarm addpeers pivot.lst
# or directly
swarm addpeers all <(IP_ADDR='[::]' swarm enode 01|tr -d '"')

# stop all running instances on the node
swarm stop all

# stop all running instances on all remote nodes
swarm remote-run nodes.lst swarm stop all

# display peer connection table of running instance 00
swarm hive 00

# display peer connection table for a running instance and continually refresh every 4 seconds
swarm monitor 00 4

# display peer connection table for all running instance on a remote node and continually refresh every 10 seconds
swarm monitor cicada@3.3.0.1 all 10


# configure eth-net-intelligence-api network monitoring client API for a node (the name argument appears as a prefix for all instances in  your cluster)
swarm netstatconf cicada-sworm

# restart the net monitor client API
swarm netstatun

# configure eth-net-intelligence-api network monitoring client API and (re)start the monitor tool on all remote nodes
swarm remote-run nodes.lst 'swarm netstatconf cicada-sworm; swarm netstatrun'


swarm remote nodes.lst 'swarm netstatconf cicada-sworm; swarm netstatrun'
```


see also:

* https://github.com/ethereum/go-ethereum/tree/swarm/swarm/test
* https://github.com/ethereum/go-ethereum/tree/swarm/swarm/cmd

# ethereum netstats client setup

## install

nodejs and npm are prerequisites

```shell
# MAC
brew install node npm
# ubuntu
sudo apt-get install npm nodejs
```

clone the git repo and install:

```
git clone git@github.com:cubedro/eth-net-intelligence-api.git
cd eth-net-intelligence-api
npm install
npm install -g pm2
```

## configure and run netstats client for each node

```shell
swarm remote-run nodes.lst 'swarm netstatconf cicada-sworm; swarm netstatrun'
```
