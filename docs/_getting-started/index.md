---
title: Getting Started with Geth
permalink: docs/getting-started
sort_key: A
---

This page explains how to set up Geth and execute some basic tasks using the command line tools. 
In order to use Geth, the software must first be installed. There are several ways Geth can be 
installed depending on the operating system and the user's choice of installation method, for 
example using a package manager, container or building from source. Instructions for installing 
Geth can be found on the ["Install and Build"](install-and-build/installing-geth) pages. 
The tutorial on this page assumes Geth and the associated developer tools have been installed successfully.

This page provides step-by-step instructions covering the fundamentals of using Geth. This includes 
generating accounts, joining an Ethereum network, syncing the blockchain and sending ether between accounts. 
It is considered best-practice to use [Clef](/docs/clef/introduction) for account management - this 
is explained in the [Geth with Clef](/docs/getting-started/geth_with_clef) tutorial. In this 
introductory tutorial, Geth's built-in account management tools are used instead.

{:toc}
-   this will be removed by the toc

## Prerequisites

In order to get the most value from the tutorials on this page, the following skills are 
necessary:

- Experience using the command line
- Basic knowledge about Ethereum and testnets
- Basic knowledge about HTTP and JavaScript

Users that need to revisit these fundamentals can find helpful resources relating to the command 
line [here][cli], Ethereum and its testnets [here](https://ethereum.org/en/developers/tutorials/), 
http [here](https://developer.mozilla.org/en-US/docs/Web/HTTP) and 
Javascript [here](https://www.javascript.com/learn).

{% include note.html content="If Geth was installed from source on Linux, `make` saves the 
binaries for Geth and the associated tools in `/build/bin`. To run these programs it is 
convenient to move them to the top level project directory (e.g. running `mv ./build/bin/* ./`) 
from `/go-ethereum`. Then `./` must be prepended to the commands in the code snippets in order to 
execute a particular program, e.g. `./geth` instead of simply `geth`. If the executables are not
moved then either navigate to the `bin` directory to run them (e.g. `cd ./build/bin` and `./geth`) 
or provide their path (e.g. `./build/bin/geth`). These instructions can be ignored for other installations." %}

## Background

Geth is an Ethereum client written in Go. This means running Geth turns a computer into an Ethereum node. 
Ethereum is a peer-to-peer network where information is shared directly between nodes rather than being 
managed by a central server. Nodes compete to generate new blocks of transactions to send to its peers 
because they are rewarded for doing so in Ethereum's native token, ether (ETH). On receiving a new block, 
each node checks that it is valid and adds it to their database. The sequence of discrete blocks is called 
a "blockchain". The information provided in each block is used by Geth to update its "state" - the ether 
balance of each account on Ethereum. There are two types of account: externally-owned accounts (EOAs) and 
contract accounts. Contract accounts execute contract code when they receive transactions. EOAs are accounts 
that users manage locally in order to sign and submit transactions. Each EOA is a public-private key pair, 
where the public key is used to derive a unique address for the user and the private key is used to protect 
the account and securely sign messages. Therefore, in order to use Ethereum, it is first necessary to generate 
an EOA (hereafter, "account"). This tutorial will guide the user through creating an account, funding it 
with ether and sending some to another address.

Read more about Ethereum accounts [here](https://ethereum.org/en/developers/docs/accounts/).


## Step 1: Generating accounts

To generate a new account in Geth:

```sh
geth account new
```

This returns a prompt for a password. Once provided, a new account will be created and added to the 
default keystore (`/datadir/keystore`). A custom keystore can also be provided by passing `--keystore <path>`. 
In this tutorial the keys will be stored in a new data directory `geth-tutorial`. Create that diredctory, then run:

```sh
geth account new --keystore geth-tutorial/keystore
```
The following will be returned to the console, confirming the new account has been created and 
added to the keystore.

```terminal
Your new account is locked with a password. Please give a password. Do not forget this password.
Password:
Repeat password:

Your new key was generated

Public address of the key:  0xca57F3b40B42FCce3c37B8D18aDBca5260ca72EC
Path of the secret key file: /home/go-ethereum/geth-tutorial/keystore/UTC--2022-07-25T08-27-59.433905560Z--ca57F3b40B42FCce3c37B8D18aDBca5260ca72EC

- You can share your public address with anyone. Others need it to interact with you.
- You must NEVER share the secret key with anyone! The key controls access to your funds!
- You must BACKUP your key file! Without the key, it's impossible to access account funds!
- You must REMEMBER your password! Without the password, it's impossible to decrypt the key!
```

It is important to save the account address and the password somewhere secure. They will be used 
again later in this tutorial. Please note that the account address shown in the code snippets 
above and later in this tutorials are examples - those generated by followers of this tutorial 
will be different. The account generated above can be used as the main account throughout the 
remainder of this tutorial. However in order to demonstrate transactions between accounts it is 
also necessary to have a second account. A second account can be added to the same keystore by 
precisely repeating the previous steps, providing the same password.

Notice that the path to the secret key includes a long filename that starts `UTC--`. This is the 
name of the file that contains the keys for the new account. It is **extremely important** that 
this file stays secure because it contains the secret key used to control access to any funds 
associated with the account. The file should be backed up securely along with the password 
used to encrypt it. If the file or the password is lost, then so is access to the funds in 
the account. If someone else gains access to the keyfile and password, they have access to any 
assets in the account. 

## Step 2:  Start Geth

Geth is the Ethereum client that will connect the computer to the Ethereum network. 
In this tutorial the network is Goerli, an Ethereum testnet. Testnets are used to test 
Ethereum client software and smart contracts in an environment where no real-world value 
is at risk. To start Geth, run the Geth executable file passing argument that define the 
data directory (where Geth should save blockchain data), the network ID and the sync mode. 
For this tutorial, snap sync is recommended 
(see [here](https://blog.ethereum.org/2021/03/03/geth-v1-10-0/) for reasons why).

The following command should be run in the terminal: 

```shell
geth --datadir geth-tutorial --goerli --syncmode snap
```
Running the above command starts Geth. The terminal should rapidly fill with status updates that look like the following:

```terminal
INFO [02-10|13:59:06.649] Starting Geth on goerli testnet...
INFO [02-10|13:59:06.649] Dropping default light client cache      provided=1024 updated=128
INFO [02-10|13:59:06.652] Maximum peer count                       ETH=50 LES=0 total=50
INFO [02-10|13:59:06.660] Set global gas cap                       cap=50,000,000
INFO [02-10|13:59:06.661] Allocated cache and file handles         database=/.../geth-tutorial/geth/chaindata cache=64.00MiB handles=5120
INFO [02-10|13:59:06.855] Persisted trie from memory database      nodes=361 size=51.17KiB time="643.54µs" gcnodes=0 gcsize=0.00B gctime=0s livenodes=1 livesize=0.00B
INFO [02-10|13:59:06.855] Initialised chain configuration          config="{ChainID: 5 Homestead: 0 DAO: nil DAOSupport: true EIP150: 0 EIP155: 0 EIP158: 0 Byzantium: 0 Constantinople: 0 Petersburg: 0 Istanbul: 1561651, Muir Glacier: nil, Berlin: 4460644, London: 5062605, Arrow Glacier: nil, MergeFork: nil, Engine: clique}"
INFO [02-10|13:59:06.862] Added trusted checkpoint                 block=5,799,935 hash=2de018..c32427
INFO [02-10|13:59:06.863] Loaded most recent local header          number=6,340,934 hash=483cf5..858315 td=9,321,576 age=2d9h29m
INFO [02-10|13:59:06.867] Configured checkpoint oracle             address=0x18CA0E045F0D772a851BC7e48357Bcaab0a0795D signers=5 threshold=2
INFO [02-10|13:59:06.867] Gasprice oracle is ignoring threshold set threshold=2
WARN [02-10|13:59:06.869] Unclean shutdown detected                booted=2022-02-08T04:25:08+0100 age=2d9h33m
INFO [02-10|13:59:06.870] Starting peer-to-peer node               instance=Geth/v1.10.15-stable/darwin-amd64/go1.17.5
INFO [02-10|13:59:06.995] New local node record                    seq=1,644,272,735,880 id=d4ffcd252d322a89 ip=127.0.0.1 udp=30303 tcp=30303
INFO [02-10|13:59:06.996] Started P2P networking                   self=enode://4b80ebd341b5308f7a6b61d91aa0ea31bd5fc9e0a6a5483e59fd4ea84e0646b13ecd289e31e00821ccedece0bf4b9189c474371af7393093138f546ac23ef93e@127.0.0.1:30303
INFO [02-10|13:59:06.997] IPC endpoint opened                      url=/.../geth-tutorial/geth.ipc
WARN [02-10|13:59:06.998] Light client mode is an experimental feature
INFO [02-10|13:59:08.793] Block synchronisation started
```

This indicates that Geth has started up and is searching for peers to connect to. Once it finds peers 
it can request block headers from them, starting at the genesis block for the Goerli blockchain. 
Geth continues to download blocks sequentially, saving the data in files in `/go-ethereum/geth-tutorial/geth/chaindata/`. 
This is confirmed by the logs printed to the terminal. There should be a rapidly-growing sequence of logs in the 
terminal with the following syntax:

```terminal

INFO [04-29][15:54:09.238] Looking for peers             peercount=2 tried=0 static=0
INFO [04-29][15:54:19.393] Imported new block headers    count=2 elapsed=1.127ms  number=996288  hash=09f1e3..718c47 age=13h9m5s
INFO [04-29][15:54:19:656] Imported new block receipts   count=698  elapsed=4.464ms number=994566 hash=56dc44..007c93 age=13h9m9s

```

These logs indicate that Geth is running as expected.

If there is no error message reported to the terminal, everything is OK. Geth must be running in 
order for a user to interact with the Ethereum network. If this terminal is closed down then Geth 
must be restarted again. Geth can be started and stopped easily, but it must be running for any 
interaction with Ethereum to take place. To shut down Geth, simply press `CTRL+C` in the Geth terminal. 
To start it again, run the previous command `geth --datadir ... ..`.

{% include note.html content="Snap syncing Goerli will take some time and until the sync is finished 
you can't use the node to transfer funds. You can also try doing a [light sync](interface/les) 
which will be much quicker but depends on light servers being available to serve your node the data it needs." %}

## Step 3:  Get Testnet Ether

In order to make some transactions, the user must fund their account with ether. On Ethereum mainnet, 
ether can only be obtained in three ways: 1) by receiving it as a reward for mining/validating; 2) 
receiving it in a transfer from another Ethereum user or contract; 3) receiving it from an exchange, 
having paid for it with fiat money. On Ethereum testnets, the ether has no real world value so it 
can be made freely available via faucets. Faucets allow users to request a transfer of testnet ether 
to their account.

The address generated by `geth account new` can be pasted into the Paradigm Multifaucet faucet 
[here](https://fauceth.komputing.org/?chain=1115511). This requires a Twitter login as proof of 
personhood. The faucets adds ether to the given address on multiple testnets simultaneously, 
including Goerli. In the next steps Geth will be used to check that the ether has been sent to 
the given address and send some of it to the second address created earlier.


## Step 4: Interact with Geth

For interacting with the blockchain, Geth provides JSON-RPC APIs. 
[JSON-RPC](https://ethereum.org/en/developers/docs/apis/json-rpc/) is a way to execute specific tasks 
by sending instructions to Geth in the form of [JSON](https://www.json.org/json-en.html) objects. 
RPC stands for "Remote Procedure Call" and it refers to the ability to send these JSON-encoded 
instructions from locations outside of those managed by Geth. It is possible to interact with Geth 
by sending these JSON encoded instructions directly to Geth using tools such as Curl. However, 
this is somewhat user-unfriendly and error-prone, especially for more complex instructions. For this 
reason, there are a set of libraries built on top of JSON-RPC that provide a more user-friendly 
interface for interacting with Geth. One of the most widely used is Web3.js. 

Geth provides a Javascript console that exposes the Web3.js API. This means that with Geth running in 
one terminal, a Javascript environment can be opened in another allowing the user to interact with 
Geth using Web3.js. There are three transport protocols that can be used to connect the Javascript 
environment to Geth:

- IPC (Inter-Process Communication): Provides unrestricted access to all APIs, but only works when the 
- console is run on the same host as the Geth node.
  
- HTTP: By default provides access to the `eth`, `web3` and `net` method namespaces.

- Websocket: By default provides access to the `eth`, `web3` and `net` method namespaces.

This tutorial will use the IPC option. To do this, the path to Geth's `ipc` file must be known. 
By default, this is the `datadir`, in this case `geth-tutorial`. In a new terminal, the following 
command can be run to start the Javascript console and connect it to Geth using the `geth.ipc` 
file from the datadir:

```shell
geth attach geth-tutorial/geth.ipc
```

The following welcome message will be displayed in the Javascript console:

```terminal
Welcome to the Geth JavaScript console!

instance: Geth/v1.10.15-stable/darwin-amd64/go1.17.5
at block: 6354736 (Thu Feb 10 2022 14:01:46 GMT+0100 (WAT))
 datadir: /home/go-ethereum/geth-tutorial
 modules: admin:1.0 clique:1.0 debug:1.0 eth:1.0 miner:1.0 net:1.0 personal:1.0 rpc:1.0 txpool:1.0 web3:1.0

To exit, press ctrl-d or type exit
```

The console is now active and connected to Geth. It can now be used to interact with the Ethereum (Goerli) network.


### List of accounts

Earlier in this tutorial, at least one account was created using `geth account new`. The following 
command will display the addresses of those two accounts and any others that might have been added 
to the keystore before or since. 

```javascript
eth.accounts
```

```terminal
["0xca57f3b40b42fcce3c37b8d18adbca5260ca72ec", "0xce8dba5e4157c2b284d8853afeeea259344c1653"]
```


### Checking account balance.

Having confirmed that the two addresses created earlier are indeed in the keystore and accessible 
through the Javascript console, it is possible to retrieve information about how much ether they 
own. The Goerli faucet should have sent 1 ETH to the address provided, meaning that the balance 
of one of the accounts should be 1 ether and the other should be 0. The following command displays 
the account balance in the console:

```javascript
web3.fromWei(eth.getBalance("0xca57F3b40B42FCce3c37B8D18aDBca5260ca72EC"), "ether")
```

There are actually two instructions sent in the above command. The inner one is the `getBalance` 
function from the `eth` namespace. This takes the account address as its only argument. By default, 
this returns the account balance in units of Wei. There are 10<sup>18</sup> Wei to one ether. To 
present the result in units of ether, `getBalance` is wrapped in the `fromWei` function from the 
`web3` namespace. Running this command should provide the following result (for the account that 
received faucet funds):

```terminal
1
```

Repeating the command for the other new account that was not funded from the faucet should yield:

```terminal
0
```

### Send ether to another account

The command `eth.sendTransaction` can be used to send some ether from one address to another. 
This command takes three arguments: `from`, `to` and `value`. These define the sender and 
recipient addresses (as strings) and the amount of Wei to transfer. It is far less error prone 
to enter the transaction value in units of ether rather than Wei, so the value field can take the 
return value from the `toWei` function. The following command, run in the Javascript console, 
sends 0.1 ether from one of the accounts in the keystore to the other. Note that the addresses 
here are examples - the user must replace the address in the `from` field with the address 
currently owning 1 ether, and the address in the `to` field with the address currently holding 0 ether.

```javascript
eth.sendTransaction({
    from: "0xca57f3b40b42fcce3c37b8d18adbca5260ca72ec",
    to: "0xce8dba5e4157c2b284d8853afeeea259344c1653",
    value: web3.toWei(0.1, "ether")
})
```

This command will return an error message indicating that `authentication is needed: password or unlock`. 
This is a security feature that prevents unauthorized access to sensitive account operations. 
There are two ways to unlock the account. The first is to start Geth with the account permanently 
unlocked (by passing `--unlock <address>` at startup). This is not recommended because the account 
remains unlocked all the time Geth is running, creating a security weakness. Instead, it is better 
to temporarily unlock the account for the specific transaction. This requires using the `sendTransaction` 
method from the `personal` namespace instead of the `eth` namespace. The password can be provided as a 
string in the method call as follows:

```sh
personal.sendTransaction({
    from: "0xca57f3b40b42fcce3c37b8d18adbca5260ca72ec",
    to: "0xce8dba5e4157c2b284d8853afeeea259344c1653",
    value: web3.toWei(0.1, "ether")
}, "password")
```

In the Javascript console, the transaction hash is displayed. This will be used in the next section 
to retrieve the transaction details.

```terminal
"0x99d489d0bd984915fd370b307c2d39320860950666aac3f261921113ae4f95bb"
```

It is also advised to check the account balances using Geth by repeating the instructions from earlier. 
At this point in the tutorial, the two accounts in the keystore should have balances just below 0.9 
ether (because 0.1 ether has been transferred out and some small amount paid in transaction gas) and 0.1 ether.

### Checking the transaction hash

The transaction hash is a unique identifier for this specific transaction that can be used later to 
retrieve the transaction details. For example, the transaction details can be viewed by pasting this 
hash into the [Goerli block explorer](https://goerli.etherscan.io/). The same information can also 
be retrieved directly from the Geth node. The hash returned in the previous step can be provided as 
an argument to `eth.getTransaction` to return the transaction information:

```javascript
eth.getTransaction("0x99d489d0bd984915fd370b307c2d39320860950666aac3f261921113ae4f95bb")
```

This returns the following response (although the actual values for each field will vary because they 
are specific to each transaction):

```terminal
{
  accessList: [],
  blockHash: "0x1c5d3f8dd997b302935391b57dc3e4fffd1fa2088ef2836d51f844f993eb39c4",
  blockNumber: 6355150,
  chainId: "0x5",
  from: "0xca57f3b40b42fcce3c37b8d18adbca5260ca72ec",
  gas: 21000,
  gasPrice: 2425000023,
  hash: "0x99d489d0bd984915fd370b307c2d39320860950666aac3f261921113ae4f95bb",
  input: "0x",
  maxFeePerGas: 2425000057,
  maxPriorityFeePerGas: 2424999967,
  nonce: 3,
  r: "0x66e5d23ad156e04363e68b986d3a09e879f7fe6c84993cef800bc3b7ba8af072",
  s: "0x647ff82be943ea4738600c831c4a19879f212eb77e32896c05055174045da1bc",
  to: "0xce8dba5e4157c2b284d8853afeeea259344c1653",
  transactionIndex: 630,
  type: "0x2",
  v: "0x0",
  value: 10000000000000000
}
```

## Using Curl

Up to this point this tutorial has interacted with Geth using the convenience library Web3.js. 
This library enables the user to send instructions to Geth using a more user-friendly interface 
compared to sending raw JSON objects. However, it is also possible for the user to send these JSON 
objects directly to Geth's exposed HTTP port. Curl is a command line tool that sends HTTP requests. 
This part of the tutorial demonstrates how to check account balances and send a transaction using Curl. 
This requires Geth to expose an HTTP port to listen for requests. This can be configured at startup 
by passing the `--http` flag. If no other commands are passed with it, `--http` will expose the 
default `localhost:8545` port. 

### Checking account balance

The command below returns the balance of the given account. This is a HTTP POST request to the local 
port 8545. The `-H` flag is for header information. It is used here to define the format of the incoming 
payload, which is JSON. The `--data` flag defines the content of the payload, which is a JSON object. 
That JSON object contains four fields: `jsonrpc` defines the spec version for the JSON-RPC API, `method` 
is the specific function being invoked, `params` are the function arguments, and `id` is used for ordering 
transactions. The two arguments passed to `eth_getBalance` are the account address whose balance to check 
and the block to query (here `latest` is used to check the balance in the most recently mined block).

```shell
curl -X POST http://127.0.0.1:8545 \
  -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0", "method":"eth_getBalance", "params":["0xca57f3b40b42fcce3c37b8d18adbca5260ca72ec","latest"], "id":1}'
```

A successful call will return a response like the one below:

```terminal
{"jsonrpc":"2.0","id":1,"result":"0xc7d54951f87f7c0"}
```

The balance is in the `result` field in the returned JSON object. However, it is denominated in Wei and 
presented as a hexadecimal string. There are many options for converting this value to a decimal in units 
of ether, for example by opening a Python console and running:

```python
0xc7d54951f87f7c0 / 1e18
```
This returns the balance in ether:

```terminal 
0.8999684999998321
```

### Checking the account list

The curl command below returns the list of all accounts.

```shell
curl -X POST http://127.0.0.1:8545 \
    -H "Content-Type: application/json" \
   --data '{"jsonrpc":"2.0", "method":"eth_accounts","params":[], "id":1}'
```

The following information is returned to the terminal:

```terminal
{"jsonrpc":"2.0","id":1,"result":["0xca57f3b40b42fcce3c37b8d18adbca5260ca72ec"]}
```

### Sending Transactions

It is possible to send transactions using raw curl requests too, but this requires unlocking the sender 
account. It is recommended to do this using Clef to manage access to accounts or to use `ipc` instead. The 
combination of HTTP and unlocked accounts pose a security risk.

## Summary

This tutorial has demonstrated how to generate accounts using Geth's built-in account management tool, 
fund them with testnet ether and use those accounts to interact with Ethereum (Goerli) through a Geth 
node. Checking account balances, sending transactions and retrieving transaction details were explained using 
the web3.js library via the Geth console and using the JSON-RPC directly using Curl. Note that this is an 
entry-level tutorial designed to help users get familiar with basic Geth processes, we strongly recommend 
following this with the [Geth with Clef](/docs/getting-started/geth_with_clef) tutorial which will help to 
adopt more secure account management practices than those outlined here.


[cli]: https://developer.mozilla.org/en-US/docs/Learn/Tools_and_testing/Understanding_client-side_tools/Command_line
