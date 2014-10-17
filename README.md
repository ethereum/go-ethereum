Ethereum
========

[![Build
Status](http://build.ethdev.com/buildstatusimage?builder=Linux%20Go%20master%20branch)](http://build.ethdev.com:8010/builders/Linux%20Go%20master%20branch/builds/-1) master [![Build
Status](http://build.ethdev.com/buildstatusimage?builder=Linux%20Go%20develop%20branch)](http://build.ethdev.com:8010/builders/Linux%20Go%20develop%20branch/builds/-1) develop

Ethereum Go Client © 2014 Jeffrey Wilcke.

Current state: Proof of Concept 0.6.7.

For the development package please see the [eth-go package](https://github.com/ethereum/eth-go).

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

If you would like to contribute to Ethereum Go, please fork, fix, commit and
send a pull request to the main repository. Commits which do not comply with the coding standards explained below
will be ignored. If you send a pull request, make sure that you
commit to the `develop` branch and that you do not merge to `master`.
Commits that are directly based off of the `master` branch instead of the `develop` branch will be ignored.

To make this process simpler try following the [git flow](http://nvie.com/posts/a-successful-git-branching-model/) branching model, as it sets this process up and streamlines work flow.

Coding standards
================

Code should be formatted according to the [Go Formatting
Style](http://golang.org/doc/effective_go.html#formatting).

Unless struct fields are supposed to be directly accessible, provide
getters and hide the fields through Go's exporting facility.

Make comments in your code meaningful and only use them when necessary. Describe in detail what your code is trying to achieve. For example, this would be redundant and unnecessary commenting:

*wrong*

```go
// Check if the value at x is greater than y
if x > y {
    // It's greater!
}
```

Everyone reading the source code should know what this code snippet was meant to achieve, and so those are **not** meaningful comments.

While this project is constantly tested and run, code tests should be written regardless. There is not time to evaluate every person's code specifically, so it is expected of you to write tests for the code so that it does not have to be tested manually. In fact, contributing by simply writing tests is perfectly fine!

