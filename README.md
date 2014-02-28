Ethereum
========

[![Build Status](https://travis-ci.org/ethereum/go-ethereum.png?branch=master)](https://travis-ci.org/ethereum/go-ethereum)

Ethereum Go Client (c) Jeffrey Wilcke

The current state is "Proof of Concept 3".

For the development Go Package please see [eth-go package](https://github.com/ethereum/eth-go).

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
-gui     Launch with GUI (= true)
-dir     Data directory used to store configs and databases (=".ethereum")
-import  Import a private key (hex)
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

