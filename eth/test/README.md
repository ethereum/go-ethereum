= Integration tests for eth protocol and blockpool 

This is a simple suite of tests to fire up a local test node with peers to test blockchain synchronisation and download. 
The scripts call ethereum (assumed to be compiled in go-ethereum root). 

To run a test:

    . run.sh 00 02

Without arguments, all tests are run. 

Peers are launched with preloaded imported chains. In order to prevent them from synchronizing with each other they are set with `-dial=false` and `-maxpeer 1` options. They log into `/tmp/eth.test/nodes/XX` where XX is the last two digits of their port. 

Chains to import can be bootstrapped by letting nodes mine for some time. This is done with 

    . bootstrap.sh 

Only the relative timing and forks matter so they should work if the bootstrap script is rerun. 
The reference blockchain of tests are soft links to these import chains and check at the end of a test run. 

Connecting to peers and exporting blockchain is scripted with JS files executed by the JSRE, see `tests/XX.sh`. 

Each test is set with a timeout. This may vary on different computers so adjust sensibly. 
If you kill a test before it completes, do not forget to kill all the background processes, since they will impact the result. Use:

    killall ethereum

