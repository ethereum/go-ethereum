[![Stories in Ready](https://badge.waffle.io/ethereum/go-ethereum.png?label=ready&title=Ready)](https://waffle.io/ethereum/go-ethereum)
Ethereum
========

[![Build
Status](http://build.ethdev.com/buildstatusimage?builder=Linux%20Go%20master%20branch)](http://build.ethdev.com:8010/builders/Linux%20Go%20master%20branch/builds/-1) master [![Build
Status](http://build.ethdev.com/buildstatusimage?builder=Linux%20Go%20develop%20branch)](http://build.ethdev.com:8010/builders/Linux%20Go%20develop%20branch/builds/-1) develop

Ethereum Go Client © 2014 Jeffrey Wilcke.

Current state: Proof of Concept 0.6.7.

Ethereum is currently in its testing phase. 
For build instructions see the [Wiki](https://github.com/ethereum/go-ethereum/wiki/Building-Ethereum(Go)).

Ethereum Go is split up in several sub packages Please refer to each
individual package for more information.
  1. [eth](https://github.com/ethereum/go-ethereum)
  2. [ethchain](https://github.com/ethereum/go-ethereum/tree/master/ethchain)
  3. [ethwire](https://github.com/ethereum/go-ethereum/tree/master/ethwire)
  4. [ethdb](https://github.com/ethereum/go-ethereum/tree/master/ethdb)
  5. [ethutil](https://github.com/ethereum/go-ethereum/tree/master/ethutil)
  6. [ethpipe](https://github.com/ethereum/go-ethereum/tree/master/ethpipe)
  7. [ethvm](https://github.com/ethereum/go-ethereum/tree/master/ethvm)
  8. [ethtrie](https://github.com/ethereum/go-ethereum/tree/master/ethtrie)
  9. [ethreact](https://github.com/ethereum/go-ethereum/tree/master/ethreact)
  10. [ethlog](https://github.com/ethereum/go-ethereum/tree/master/ethlog)

The [eth](https://github.com/ethereum/go-ethereum) is the top-level package
of the Ethereum protocol. It functions as the Ethereum bootstrapping and
peer communication layer. The [ethchain](https://github.com/ethereum/go-ethereum/tree/master/ethchain)
contains the Ethereum blockchain, block manager, transaction and
transaction handlers. The [ethwire](https://github.com/ethereum/go-ethereum/tree/master/ethwire) contains
the Ethereum [wire protocol](http://wiki.ethereum.org/index.php/Wire_Protocol) which can be used
to hook in to the Ethereum network. [ethutil](https://github.com/ethereum/go-ethereum/tree/master/ethutil) contains
utility functions which are not Ethereum specific. The utility package
contains the [patricia trie](http://wiki.ethereum.org/index.php/Patricia_Tree),
[RLP Encoding](http://wiki.ethereum.org/index.php/RLP) and hex encoding
helpers. The [ethdb](https://github.com/ethereum/go-ethereum/tree/master/ethdb) package
contains the LevelDB interface and memory DB interface.

Build
=======

To build Mist (GUI):

`go get github.com/ethereum/go-ethereum/mist`

To build the node (CLI):

`go get github.com/ethereum/go-ethereum/ethereum`

For further, detailed, build instruction please see the [Wiki](https://github.com/ethereum/go-ethereum/wiki/Building-Ethereum(Go))

General command line options
====================

```
Shared between ethereum and Mist
-id      Set the custom identifier of the client (shows up on other clients)
-port    Port on which the server will accept incomming connections
-upnp    Enable UPnP
-maxpeer Desired amount of peers
-rpc     Start JSON RPC

-dir     Data directory used to store configs and databases
-import  Import a private key
-genaddr Generates a new address and private key (destructive action)
-h       This

Ethereum only
ethereum [options] [filename]
-js        Start the JavaScript REPL
filename   Load the given file and interpret as JavaScript
-m       Start mining blocks

Mist only
-asset_path    absolute path to GUI assets directory
```

Contribution
============

If you'd like to contribute to Ethereum please fork, fix, commit and
send a pull request. Commits who do not comply with the coding standards
are ignored (use gofmt!). If you send pull requests make absolute sure that you
commit on the `develop` branch and that you do not merge to master.
Commits that are directly based on master are simply ignored.

To make life easier try [git flow](http://nvie.com/posts/a-successful-git-branching-model/) it sets
this all up and streamlines your work flow.

Coding standards
================

Sources should be formatted according to the [Go Formatting
Style](http://golang.org/doc/effective_go.html#formatting).

Unless structs fields are supposed to be directly accesible, provide
Getters and hide the fields through Go's exporting facility.

When you comment put meaningfull comments. Describe in detail what you
want to achieve.

*wrong*

```go
// Check if the value at x is greater than y
if x > y {
    // It's greater!
}
```

Everyone reading the source probably know what you wanted to achieve
with above code. Those are **not** meaningful comments.

While the project isn't 100% tested I want you to write tests non the
less. I haven't got time to evaluate everyone's code in detail so I
expect you to write tests for me so I don't have to test your code
manually. (If you want to contribute by just writing tests that's fine
too!)

