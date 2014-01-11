Ethereum
========

[![Build Status](https://travis-ci.org/ethereum/go-ethereum.png?branch=master)](https://travis-ci.org/ethereum/go-ethereum)

Ethereum Go

Deps
====

Ethereum Go makes use of a modified `secp256k1-go` and therefor GMP.

Install
=======

```go get https://github.com/ethereum/go-ethereum```


Command line options
====================

-c      launch the developer console
-m      start mining fake blocks and broadcast fake messages to the net

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

Don't "overcomment", meaning that your and my mom doesn't have to read
the source code.

*wrong*

```go
// Check if the value at x is greater than y
if x > y {
    // It's greater!
}
```
