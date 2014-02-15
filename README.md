Ethereum
========

[![Build Status](https://travis-ci.org/ethereum/go-ethereum.png?branch=master)](https://travis-ci.org/ethereum/go-ethereum)

Ethereum Go developer client (c) Jeffrey Wilcke

Ethereum is currently in its testing phase. The current state is "Proof
of Concept 2". For build instructions see the [Wiki](https://github.com/ethereum/go-ethereum/wiki/Building-Edge).

Ethereum Go is split up in several sub packages Please refer to each
individual package for more information.
  1. [eth](https://github.com/ethereum/eth-go)
  2. [ethchain](https://github.com/ethereum/ethchain-go)
  3. [ethwire](https://github.com/ethereum/ethwire-go)
  4. [ethdb](https://github.com/ethereum/ethdb-go)
  5. [ethutil](https://github.com/ethereum/ethutil-go)

The [eth](https://github.com/ethereum/eth-go) is the top-level package
of the Ethereum protocol. It functions as the Ethereum bootstrapping and
peer communication layer. The [ethchain](https://github.com/ethereum/ethchain-go)
contains the Ethereum blockchain, block manager, transaction and
transaction handlers. The [ethwire](https://github.com/ethereum/ethwire-go) contains
the Ethereum [wire protocol](http://wiki.ethereum.org/index.php/Wire_Protocol) which can be used
to hook in to the Ethereum network. [ethutil](https://github.com/ethereum/ethutil-go) contains
utility functions which are not Ethereum specific. The utility package
contains the [patricia trie](http://wiki.ethereum.org/index.php/Patricia_Tree),
[RLP Encoding](http://wiki.ethereum.org/index.php/RLP) and hex encoding
helpers. The [ethdb](https://github.com/ethereum/ethdb-go) package
contains the LevelDB interface and memory DB interface.

This executable is the front-end (currently nothing but a dev console) for
the Ethereum Go implementation.

If you'd like to start developing your own tools please check out the
[development](https://github.com/ethereum/eth-go) package.

Build
=======

For build instruction please see the [Wiki](https://github.com/ethereum/go-ethereum/wiki/Building-Edge)


Command line options
====================

```
-c       Launch the developer console
-m       Start mining blocks
-genaddr Generates a new address and private key (destructive action)
-p       Port on which the server will accept incomming connections (= 30303)
-upnp    Enable UPnP (= false)
-x       Desired amount of peers (= 5)
-h       This help
```

Developer console commands
==========================

```
addp <host>:<port>     Connect to the given host
tx <addr> <amount>     Send <amount> Wei to the specified <addr>
```

See the "help" command for *developer* options.

Contribution
============

If you'd like to contribute to Ethereum Go please fork, fix, commit and
send a pull request. Commits who do not comply with the coding standards
are ignored. If you send pull requests make absolute sure that you
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

