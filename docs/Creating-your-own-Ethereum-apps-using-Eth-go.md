The modular nature of Go and the Ethereum Go implementation, [eth-go](https://github.com/ethereum/eth-go), make it very easy to build your own Ethereum native applications. 

This post will cover the minimal steps required to build an native Ethereum application.

Ethereum comes with a global config found in the [ethutil](https://github.com/ethereum/eth-go/tree/master/ethutil) package. The global config requires you to set a base path to store it's files (database, settings, etc).

```go
func main() {
	// Read config
	ethutil.ReadConfig(".test", ethutil.LogStd, nil, "MyEthApp")
}
```  

ReadConfig takes four arguments. The data folder to use, a log flag, a globalConf instance and an id string to identify your app to other nodes in the network.

Once you've configured the global config you can set up and create your Ethereum node. The Ethereum Object, or Node, will handle all trafic from and to the Ethereum network as well as handle all incoming block and transactions. A new node can be created through the `new` method found in [eth-go](https://github.com/ethereum/eth-go).

```go
func main() {
		// Read config
        ethutil.ReadConfig(".test", ethutil.LogStd, nil, "MyEthApp")
        
        // Create a new ethereum node
        ethereum, err := eth.New(eth.CapDefault, false)
        if err != nil {
            	panic(fmt.Sprintf("Could not start node: %s\n", err))
        }
        // Set the port (default 30303)
        ethereum.Port = "10101"
        // Once we reach max, bounce them off.
        ethereum.MaxPeers = 10
}
```

New requires two arguments; the capabilities of the node and whether or not to use UPNP for port-forwarding. If you don't want to fallback to client-only features set an Ethereum port and the max amount of peers this node can connect to. 

In order to identify the node to the network you'll be required to create a private key. The easiest way to create a new keypair is by using the `KeyRing` found in the `ethutil` package.

```go
func main() {
    	// Read config
        ethutil.ReadConfig(".test", ethutil.LogStd, nil, "MyEthApp")
        
        // Create a new ethereum node
        ethereum, err := eth.New(eth.CapDefault, false)
        if err != nil {
            panic(fmt.Sprintf("Could not start node: %s\n", err))
        }
        // Set the port (default 30303)
        ethereum.Port = "10101"
        // Once we reach max, bounce them off.
        ethereum.MaxPeers = 10
        
        keyRing := ethutil.GetKeyRing()
        // Create a new key if non exist
        if keyRing.Len() == 0 {
    	    	// Create a new keypair
	        	keyPair, err := ethutil.GenerateNewKeyPair()
    	        if err != nil {
        		    	panic(err)
	            }
            
    	        // Add the keypair to the key ring
        	    keyRing.Add(keyPair)
        }
}
```

Once the base Ethereum stack has been set up it's time to fire up its engines and connect to the main network.

```go
package main

import (
		"github.com/ethereum/eth-go"
		"github.com/ethereum/eth-go/ethutil"
)

func main() {
	    // Read config
    	ethutil.ReadConfig(".test", ethutil.LogStd, nil, "MyEthApp")
    
	    // Create a new ethereum node
    	ethereum, err := eth.New(eth.CapDefault, false)
	    if err != nil {
    		    panic(fmt.Sprintf("Could not start node: %s\n", err))
	    }
    	// Set the port (default 30303)
	    ethereum.Port = "10101"
    	// Once we reach max, bounce them off.
	    ethereum.MaxPeers = 10
    
    	keyRing := ethutil.GetKeyRing()
	    // Create a new key if non exist
    	if keyRing.Len() == 0 {
        		// Create a new keypair
		        keyPair, err := ethutil.GenerateNewKeyPair()
        		if err != nil {
			            panic(err)
        		}
        
		        // Add the keypair to the key ring
        		keyRing.Add(keyPair)
	    }

    	ethereum.Start(true)
	    ethereum.WaitForShutdown()
}
```

`ethereum.Start()` takes one argument, whether or not we want to connect to one of the known seed nodes. If you want your own little testnet-in-a-box you can disable it else set it to true.

Your node should now be catching up with the blockchain. From here on out you are on your own. You could create a reactor to listen to specific events or just dive into the chain state directly. If you want to look at some example code you can check [DNSEth here.](https://github.com/maran/dnseth)

Have fun!