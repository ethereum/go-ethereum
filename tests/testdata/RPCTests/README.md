See https://github.com/ethereum/cpp-ethereum/blob/7cc43bed7de890a496d7238092837c30c7e90729/scripts/runalltests.sh#L38 for how cpp-ethereum uses this.

FAQ
===

Cannot find module
------------------

I get an error:
```
$ node main.js $workdir/cpp-ethereum/build/eth/eth
module.js:471
    throw err;
    ^

Error: Cannot find module '/home/yh/src/tests/RLPTests/main.js'
    at Function.Module._resolveFilename (module.js:469:15)
    at Function.Module._load (module.js:417:25)
    at Module.runMain (module.js:604:10)
    at run (bootstrap_node.js:393:7)
    at startup (bootstrap_node.js:150:9)
    at bootstrap_node.js:508:3
```

Answer: if your `main.js` is in your current directory, use `./main.js` instead of just `main.js`.


Cannot find module web3
-----------------------

I get an error:
```
$ node ./main.js ~/src/cpp-ethereum/build/eth/eth
(node:27647) UnhandledPromiseRejectionWarning: Unhandled promise rejection (rejection id: 1): Error: Cannot find module 'web3'
```

Answer: `npm install web3`


Some tests fail
---------------

```
$ node ./main.js ~/src/cpp-ethereum/build/eth/eth
TEST_newAccount OK
TEST_addPeerOnNode2 OK
TEST_getPeerCountOnNode1 OK
TEST_mineBlockOnNode1 FAILED
TEST_mineBlockOnNode1 FAILED
TEST_getBlockHashOnNode2 OK
TEST_mineBlockOnNode2 FAILED
TEST_mineBlockOnNode2 FAILED
TEST_getBlockHashOnNode1 OK
(node:30406) UnhandledPromiseRejectionWarning: Unhandled promise rejection (rejection id: 1): Error: Callback was already called.
```

Answer: everybody experiences these failures now. They are being tracked in [issue 377](https://github.com/ethereum/tests/issues/377).


Do these failures indicate bugs in cpp-ethereum or in the test?
---------------------------------------------------------------

Different opinions exist
* https://github.com/ethereum/tests/pull/376#issuecomment-349799774
* https://github.com/ethereum/tests/pull/376#issuecomment-349933405

Has any other clients been tested with this?
--------------------------------------------

No.
