THIS WIKI IS BEING EDITED AND REVIEWED NOW. PLEASE DO NOT RELY ON IT.

# Account types and transactions

There are two types of accounts in Ethereum state:
* Normal or externally controlled accounts and
* contracts, i.e., sinppets of code, think a class.

Both types of accounts have an ether balance.

Transactions can be fired from from both types of accounts, though contracts only fire transactions in response to other transactions that they have received. Therefore, all action on ethereum block chain is set in motion by transactions fired from externally controlled accounts.

The simplest transactions are ether transfer transactions. But before we go into that you should read up on [accounts](https://github.com/ethereum/go-ethereum/wiki/Managing-your-accounts) and perhaps on [mining](https://github.com/ethereum/go-ethereum/wiki/Mining).

## Ether transfer

Assuming the account you are using as sender has sufficient funds, sending ether couldn't be easier. Which is also why you should probably be careful with this! You have been warned.

```js
eth.sendTransaction({from: '0x036a03fc47084741f83938296a1c8ef67f6e34fa', to: '0xa8ade7feab1ece71446bed25fa0cf6745c19c3d5', value: web3.toWei(1, "ether")})
```

Note the unit conversion in the `value` field. Transaction values are expressed in weis, the most granular units of value. If you want to use some other unit (like `ether` in the example above), use the function `web3.toWei` for conversion.

Also, be advised that the amount debited from the source account will be slightly larger than that credited to the target account, which is what has been specified. The difference is a small transaction fee, discussed in more detail later.

Contracts can receive transfers just like externally controlled accounts, but they can also receive more complicated transactions that actually run (parts of) their code and update their state. In order to understand those transactions, a rudimentary understanding of contracts is required.

# contract のコンパイル

blockchain 上で有効となる contract は Ethereum 特別仕様の バイナリの形式で、EVM byte コード と呼ばれます。
しかしながら、典型的には、contract は [solidity](https://github.com/ethereum/wiki/wiki/Solidity-Tutorial) のような高級言語で記述され、blockchain 上に upload するために、この byte コードへコンパイルされます。

flontier リリースでは、geth は Christian R. と Lefteris K が手がけた、コマンドライン [solidity コンパイラ](https://github.com/ethereum/cpp-ethereum/tree/develop/solc) である `solc` をシステムコールで呼び出すことを通して、solidity コンパイルをサポートしています。
以下もお試しください。
* [Solidity realtime compiler](https://chriseth.github.io/cpp-ethereum/) (by Christian R) 
* [Cosmo](http://meteor-dapp-cosmo.meteor.com) 
* [Mix]() 
* [AlethZero]()



Note that other languages also exist, notably [serpent]() and [lll]().

If you start up your `geth` node, you can check if this option is immediately available. This is what happens, if it is not:

```js
eth.getCompilers()
['' ]
> eth.compile.solidity("")
error: eth_compileSolidity method not implemented
Invalid JSON RPC response
```

After you found a way to install `solc`, you make sure it's in the path, if [`eth.getCompilers()`](https://github.com/ethereum/wiki/wiki/JavaScript-API#web3ethgetcompilers) still does not find it (returns an empty array), you can set a custom path to the `sol` executable on the command line using th `solc` flag.

```
geth --datadir ~/frontier/00 --solc /usr/local/bin/solc --natspec
```

You can also set this option at runtime via the console:

```js
> admin.setSolc("/usr/local/bin/solc")
solc v0.9.13
Solidity Compiler: /usr/local/bin/solc
Christian <c@ethdev.com> and Lefteris <lefteris@ethdev.com> (c) 2014-2015
true
```

Let us take this simple contract source:

```js
> source = "contract test { function multiply(uint a) returns(uint d) { return a * 7; } }"
```

This contract offers a unary method: called with a positive integer `a`, it returns `a * 7`. 
Note that this document is not about writing interesting contracts or about the features of solidity.

For more information on contract language, go through [solidity tutorial](https://github.com/ethereum/wiki/wiki/Solidity-Tutorial), browse the contracts in our [dapp-bin](https://github.com/ethereum/dapp-bin/wiki), see other solidity and dapp resources. 

You are ready to compile solidity code in the `geth` JS console using [`eth.compile.solidity`](https://github.com/ethereum/wiki/wiki/JavaScript-API#web3ethcompilesolidity):

```js
> contract = eth.compile.solidity(source)
{
  code: '605280600c6000396000f3006000357c010000000000000000000000000000000000000000000000000000000090048063c6888fa114602e57005b60376004356041565b8060005260206000f35b6000600782029050604d565b91905056',
  info: {
    language: 'Solidity',
    languageVersion: '0',
    compilerVersion: '0.9.13',
    abiDefinition: [{
      constant: false,
      inputs: [{
        name: 'a',
        type: 'uint256'
      } ],
      name: 'multiply',
      outputs: [{
        name: 'd',
        type: 'uint256'
      } ],
      type: 'function'
    } ],
    userDoc: {
      methods: {
      }
    },
    developerDoc: {
      methods: {
      }
    },
    source: 'contract test { function multiply(uint a) returns(uint d) { return a * 7; } }'
  }
}
```

The compiler is also available via [RPC](https://github.com/ethereum/wiki/wiki/JSON-RPC) and therefore via [web3.js](https://github.com/ethereum/wiki/wiki/JavaScript-API#web3ethcompilesolidity) to any in-browser Ðapp connecting to `geth` via RPC.

The following example shows how you interface `geth` via JSON-RPC to use the compiler.

```
./geth --datadir ~/eth/ --loglevel 6 --logtostderr=true --rpc --rpcport 8100 --rpccorsdomain '*' --mine console  2>> ~/eth/eth.log
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_compileSolidity","params":["contract test { function multiply(uint a) returns(uint d) { return a * 7; } }"],"id":1}' http://127.0.0.1:8100
```

The compiler output is combined into an object representing a single contract and is serialised as json. It contains the following fields:

* `code`: the compiled EVM code
* `source`: the source code 
* `language`: contract language (Solidity, Serpent, LLL)
* `languageVersion`: contract language version
* `compilerVersion`: compiler version 
* `abiDefinition`: [Application Binary Interface Definition](https://github.com/ethereum/wiki/wiki/Ethereum-Contract-ABI)
* `userDoc`: [NatSpec user Doc](https://github.com/ethereum/wiki/wiki/Ethereum-Natural-Specification-Format)
* `developerDoc`: [NatSpec developer Doc](https://github.com/ethereum/wiki/wiki/Ethereum-Natural-Specification-Format)

The immediate structuring of the compiler output (into `code` and `info`) reflects the two very different **paths of deployment**. 
The compiled EVM code is sent off to the blockchain with a contract creation transaction while the rest (info) will ideally live on the decentralised cloud as publicly verifiable metadata complementing the code on the blockchain.

# Creating and deploying a contract

Now that you got both an unlocked account as well as some funds, you can create a contract on the blockchain by [sending a transaction](https://github.com/ethereum/wiki/wiki/JavaScript-API#web3ethsendtransaction) to the empty address with the evm code as data. Simple, eh?

```js
primaryAddress = eth.accounts[0]
contractAddress = eth.sendTransaction({from: primaryAddress, data: evmCode})
```

All binary data is serialised in hexadecimal form. Hex strings always have a hex prefix `0x`.

Note that this step requires you to pay for execution. Your balance on the account (that you put as sender in the `from` field) will be reduced according to the gas rules of the VM once your transaction makes it into a block. More on that later. After some time, your transaction should appear included in a block confirming that the state it brought about is a consensus. Your contract now lives on the blockchain. 

The asynchronous way of doing the same looks like this:

```js
eth.sendTransaction({from: primaryAccount, data: evmCode}, function(err, address) {
  if (!err)
    console.log(address); 
});
```

# Gas and transaction costs

So how did you pay for all this? Under the hood, the transaction specified a gas limit and a gasprice, both of which could have been specified directly in the transaction object.

Gas limit is there to protect you from buggy code running until your funds are depleted. The product of `gasPrice` and `gas` represents the maximum amount of Wei that you are willing to pay for executing the transaction. What you specify as `gasPrice` is used by miners to rank transactions for inclusion in the blockchain. It is the price in Wei of one unit of gas, in which VM operations are priced.

The gas expenditure incurred by running your contract will be bought by the ether you have in your account at a price you specified in the transaction with `gasPrice`. If you do not have the ether to cover all the gas requirements to complete running your code, the processing aborts and all intermediate state changes roll back to the pre-transaction snapshot. The gas used up to the point where execution stopped were used after all, so the ether balance of your account will be reduced. These parameters can be adjusted on the transaction object fields `gas` and `gasPrice`. The `value` field is used the same as in ether transfer transactions between normal accounts. In other words transferring funds is available between any two accounts, either normal (i.e. externally controlled) or contract. If your contract runs out of funds, you should see an insufficient funds error. Note that all funds on contract accounts will be irrecoverably lost, once we release [Homestead](https://github.com/ethereum/go-ethereum/wiki/Homestead) (see [the rules of the game](https://github.com/ethereum/go-ethereum/wiki/Frontier)).

For testing and playing with contracts you can use the test network or [set up a private node (or cluster)](https://github.com/ethereum/go-ethereum/wiki/Setting-up-private-networklock-or-local-cluster) potentially isolated from all the other nodes. If you then mine, you can make sure that your transaction will be included in the next block. You can see the pending transactions with:

```js
eth.getBlock("pending", true).transactions
```

You can retrieve blocks by number (height) or by their hash:

```js
genesis = eth.getBlock(0)
eth.getBlock(genesis.hash).hash == genesis.hash
true
```

Use `eth.blockNumber` to get the current blockchain height and the "latest" magic parameter to access the current head (newest block).

```js
currentHeight = eth.blockNumber()
eth.getBlock("latest").hash == eth.getBlock(eth.blockNumber).hash
true
```

# Contract info (metadata)

In the previous sections we explained how you create a contract on the blockchain. Now we deal with the rest of the compiler output, the **contract metadata** or contract info. 
The idea is that 

* contract info is uploaded somewhere identifiable by a `url` which is publicly accessible
* anyone can find out what the `url` is only knowing the contracts address

These requirements are achieved very simply by using a 2 step blockchain registry. The first step registers the contract code (hash) with a content hash in a contract called `HashReg`. The second step registers a url with the content hash in the `UrlHint` contract. 
These [simple registry contracts]() will be part of the frontier proposition.

By using this scheme, it is sufficient to know a contract's address to look up the url and fetch the actual contract metadata info bundle. Read on to learn why this is good.

So if you are a conscientious contract creator, the steps are the following:

1. Get the contract info json file. 
2. Deploy contract info json file to any url of your choice
3. Register codehash ->content hash -> url
4. Deploy the contract itself to the blockchain

The JS API makes this process very easy by providing helpers. Call [`admin.contractInfo.register`]() to extract info from the contract, write out its json serialisation in the given file, calculates the content hash of the file and finally registers this content hash to the contract's code hash.
Once you deployed that file to any url, you can use [`admin.contractInfo.registerUrl`]() to register the url with your content hash on the blockchain as well. (Note that in case a fixed content addressed model is used as document store, the url-hint is no longer necessary.)

```js
source = "contract test { function multiply(uint a) returns(uint d) { return a * 7; } }"
// compile with solc
contract = eth.compile.solidity(source)
// send off the contract to the blockchain
address = eth.sendTransaction({from: primaryAccount, data: contract.code})
// extracts info from contract, save the json serialisation in the given file, 
// calculates the content hash and registers it with the code hash in `HashReg`
// it uses address to send the transaction. 
// returns the content hash that we use to register a url
hash = admin.contractInfo.register(primaryAccount, address, contract, "~/dapps/shared/contracts/test/info.json")
// here you deploy ~/dapps/shared/contracts/test/info.json to a url
admin.contractInfo.registerUrl(primaryAccount, hash, url)
```

# Interacting with contracts

[`eth.contract`](https://github.com/ethereum/wiki/wiki/JavaScript-API#web3ethcontract) can be used to define a contract _class_ that will comply with the contract interface as described in its [ABI definition](https://github.com/ethereum/wiki/wiki/Ethereum-Contract-ABI).

```js
var Multiply7 = eth.contract(contract.info.abiDefinition);
var multiply7 = new Multiply7(address);
```

Now all the function calls specified in the abi are made available on the contract instance. You can just call those methods on the contract instance and chain `sendTransaction({from: address})` or `call()` to it. The difference between the two is that `call` performs a "dry run" locally, on your computer, while `sendTransaction` would actually submit your transaction for inclusion in the block chain and the results of its execution will eventually become part of the global consensus. In other words, use `call`, if you are interested only in the return value and use `sendTransaction` if you only care about "side effects" on the state of the contract.

In the example above, there are no side effects, therefore `sendTransaction` only burns gas and increases the entropy of the universe. All "useful" functionality is exposed by `call`:

```js
multiply7.multiply.call(6)
42
```

Now suppose this contract is not yours, and you would like documentation or look at the source code. 
This is made possible by making available the contract info bundle and register it in the blockchain.
The `admin.contractInfo` API provides convenience methods to fetch this bundle for any contract that chose to register.
To see how it works, read about [Contract Metadata](https://github.com/ethereum/wiki/wiki/Contract-metadata) or read the contract info deployment section of this document. 

```js
// get the contract info for contract address to do manual verification
var info = admin.contractInfo.get(address) // lookup, fetch, decode
var source = info.source;
var abiDef = info.abiDefinition
```

```js
// verify an existing contract in blockchain (NOT IMPLEMENTED)
admin.contractInfo.verify(address)
```

# NatSpec 

This section will further elaborate what you can do with contracts and transactions building on a protocol NatSpec. Solidity implements smart comments doxigen style which then can be used to generate various facades meta documents of the code. One such use case is to generate custom messages for transaction confirmation that clients can prompt users with. 

So we now extend the `multiply7` contract with a smart comment specifying a custom confirmation message  (notice).

```js
contract test {
   /// @notice Will multiply `a` by 7.
   function multiply(uint a) returns(uint d) {
       return a * 7;
   }
}
```

The comment has expressions in between backticks which are to be evaluated at the time the transaction confirmation message is presented to the user. The variables that refer to parameters of method calls then are instantiated in accordance with the actual transaction data sent by the user (or the user's dapp). NatSpec support for confirmation notices is fully implemented in `geth`. NatSpec relies on both the abi definition as well as the userDoc component to generate the proper confirmations. Therefore in order to access that, the contract needs to have registered its contract info as described above.

Let us see a full example. As a very conscientious smart contract dev, you first create your contract and deploy according to the recommended steps above:

```js
source = "contract test {
   /// @notice Will multiply `a` by 7.
   function multiply(uint a) returns(uint d) {
       return a * 7;
   }
}"
contract = eth.compile.solidity(source)
contentHash = admin.contractInfo.register(contract, "~/dapps/shared/contracts/test/info.json")
// put it up on your favourite site:
admin.contractInfo.registerUrl(contentHash, "http://dapphub.com/test/info.json")
```

For the purposes of a painless example just simply use the file url scheme (not exactly the cloud, but will show you how it works) without needing to deploy. `admin.contractInfo.registerUrl(contentHash, "file:///home/nirname/dapps/shared/contracts/test/info.json")`.

Now you are done as a dev, so swap seats as it were and pretend that you are a user who is sending a transaction to the infamous multiply7 contract. 

You need to start the client with the `--natspec` flag to enable smart confirmations and contractInfo fetching. You can also set it on the console with `admin.contractInfo.start()` and `admin.contractInfo.stop()`.

```
geth --natspec --unlock primary console 2>> /tmp/eth.log
```

Now at the console type:

```js
// obtain the abi definition for your contract
var info = admin.contractInfo.get(address)
var abiDef = info.abiDefinition
// instantiate a contract for transactions
var Multiply7 = eth.contract(abiDef);
var multiply7 = new Multiply7();
```

And now try to send an actual transaction:

```js
> multiply7.multiply.sendTransaction(6)
NatSpec: Will multiply 6 by 7. 
Confirm? [Y/N] y
>
```

When this transaction gets included in a block, somewhere on a lucky miner's computer, 6 will get multiplied by 7, with the result ignored.

```js
// assume an existing unlocked primary account
primary = eth.accounts[0];

// mine 10 blocks to generate ether
admin.miner.start();
admin.debug.waitForBlocks(eth.blockNumber+10);
admin.miner.stop()  ;

balance = web3.fromWei(eth.getBalance(primary), "ether");

admin.contractInfo.newRegistry(primary);

source = "contract test {\n" +
"   /// @notice will multiply `a` by 7.\n" +
"   function multiply(uint a) returns(uint d) {\n" +
"      return a * 7;\n" +
"   }\n" +
"} ";

contract = eth.compile.solidity(source);

contractaddress = eth.sendTransaction({from: primary, data: contract.code});

eth.getBlock("pending", true).transactions;

admin.miner.start()
// waits until block height is minimum the number given.
// basically a sleep function on variable block units of time.

admin.debug.waitForBlocks(eth.blockNumber+1);
admin.miner.stop()

code = eth.getCode(contractaddress);

abiDef = JSON.parse('[{"constant":false,"inputs":[{"name":"a","type":"uint256"}],"name":"multiply","outputs":[{"name":"d","type":"uint256"}],"type":"function"}]');
Multiply7 = eth.contract(abiDef);
multiply7 = new Multiply7(contractaddress);

fortytwo = multiply7.multiply.call(6);
console.log("multiply7.multiply.call(6) => "+fortytwo);
multiply7.multiply.sendTransaction(6, {from: primary})

admin.miner.start();
admin.debug.waitForBlocks(eth.blockNumber+1);
admin.miner.stop();

filename = "/tmp/info.json";
contenthash = admin.contractInfo.register(primary, contractaddress, contract, filename);

admin.contractInfo.registerUrl(primary, contenthash, "file://"+filename);

admin.miner.start();
admin.debug.waitForBlocks(eth.blockNumber+1);
admin.miner.stop();

info = admin.contractInfo.get(contractaddress);

admin.contractInfo.start();
abiDef = JSON.parse('[{"constant":false,"inputs":[{"name":"a","type":"uint256"}],"name":"multiply","outputs":[{"name":"d","type":"uint256"}],"type":"function"}]');
Multiply7 = eth.contract(abiDef);
multiply7 = new Multiply7(contractaddress);
fortytwo = multiply7.multiply.sendTransaction(6, { from: primary });

```