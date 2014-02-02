Ethereum
========

[![Build Status](https://travis-ci.org/ethereum/go-ethereum.png?branch=master)](https://travis-ci.org/ethereum/go-ethereum)

Ethereum Go developer client (c) [0255c7881](https://github.com/ethereum/go-ethereum#copy)

A fair warning; Ethereum is not yet to be used in production. There's no
test-net and you aren't mining real blocks (just one which is the genesis block).


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

Deps
====

Ethereum Go makes use of a modified `secp256k1-go` and therefor GMP.

Ubuntu 12+
* `apt-get install gmp-dev`
 
OS X 10.9+: 
* `brew install gmp`

Build
=======

* `go get -u -t github.com/ethereum/go-ethereum`
* `cd $GOPATH/src/github.com/ethereum/go-etherum`
* `go build`


Command line options
====================

```
-c      launch the developer console
-m      start mining fake blocks and broadcast fake messages to the net
```

Contribution
============

If you'd like to contribute to Ethereum Go please fork, fix, commit and
send a pull request. Commits who do not comply with the coding standards
are ignored.

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

### Copy

69bce990a619e747b4f57483724b0e8a1732bb3b44ccf70b0dd6abd272af94550fc9d8b21232d33ebf30d38a148612f68e936094b4daeb9ea7174088a439070401 0255c78815d4f056f84c96de438ed9e38c69c0f8af24f5032248be5a79fe9071c3
