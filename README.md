Ethereum
========

[![Build Status](https://travis-ci.org/ethereum/go-ethereum.png?branch=master)](https://travis-ci.org/ethereum/go-ethereum)

Ethereum Go Development package (C) Jeffrey Wilcke

Ethereum is currently in its testing phase. The current state is "Proof
of Concept 5.0 RC6". For build instructions see the [Wiki](https://github.com/ethereum/go-ethereum/wiki/Building-Ethereum(Go)).

Ethereum Go is split up in several sub packages Please refer to each
individual package for more information.
  1. [eth](https://github.com/ethereum/eth-go)
  2. [ethchain](https://github.com/ethereum/eth-go/tree/master/ethchain)
  3. [ethwire](https://github.com/ethereum/eth-go/tree/master/ethwire)
  4. [ethdb](https://github.com/ethereum/eth-go/tree/master/ethdb)
  5. [ethutil](https://github.com/ethereum/eth-go/tree/master/ethutil)

The [eth](https://github.com/ethereum/eth-go) is the top-level package
of the Ethereum protocol. It functions as the Ethereum bootstrapping and
peer communication layer. The [ethchain](https://github.com/ethereum/eth-go/tree/master/ethchain)
contains the Ethereum blockchain, block manager, transaction and
transaction handlers. The [ethwire](https://github.com/ethereum/eth-go/tree/master/ethwire) contains
the Ethereum [wire protocol](http://wiki.ethereum.org/index.php/Wire_Protocol) which can be used
to hook in to the Ethereum network. [ethutil](https://github.com/ethereum/eth-go/tree/master/ethutil) contains
utility functions which are not Ethereum specific. The utility package
contains the [patricia trie](http://wiki.ethereum.org/index.php/Patricia_Tree),
[RLP Encoding](http://wiki.ethereum.org/index.php/RLP) and hex encoding
helpers. The [ethdb](https://github.com/ethereum/eth-go/tree/master/ethdb) package
contains the LevelDB interface and memory DB interface.

This is the bootstrap package. Eth-go contains all the necessary code to
get a node and connectivity going.

Build
=======

This is the Developer package. For the Ethereal client please see
[Ethereum(G)](https://github.com/ethereum/go-ethereum).

`go get -u github.com/ethereum/eth-go`

Contribution
============

If you'd like to contribute to Eth please fork, fix, commit and
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

