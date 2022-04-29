---
title: Getting Started with Geth
permalink: docs/getting-started
sort_key: A
---

This page explains how to set up Geth and execute some basic tasks using the command line tools. In order to use Geth, the software must first be installed. There are several ways Geth can be installed depending on the operating system and the user's choice of installation method, for example using a package manager, container or building from source. Instructions for installing Geth can be found on the ["Install and Build"](install-and-build/installing-geth) pages. The tutorial on this page assumes Geth and the associated developer tools have been installed.

This page provides step-by-step instructions covering the fundamentals of using Geth. This includes generating accounts, joining an Ethereum network, syncing the blockchain and sending ether between accounts. This tutorial also uses [Clef](clef/tutorial). Clef is an account management tool external to Geth itself that allows users to sign transactions. It is developed and maintained by the Geth team and is intended to eventually replace the account management tool built in to Geth.

## Prerequisites

In order to get the msot value from the tutorials on this page, the following skills are necessary:

- Experience using the command line
- Basic knowledge about Ethereum and testnets
- Basic knowledge about HTTP and JavaScript

Users that need to revisit these fundamentals can find helpful resources relating to the command line [here](https://developer.mozilla.org/en-US/docs/Learn/Tools_and_testing/Understanding_client-side_tools/Command_line), Ethereum and its testnets [here](https://ethereum.org/en/developers/tutorials/), http [here](https://developer.mozilla.org/en-US/docs/Web/HTTP) and javascript [here](https://www.javascript.com/learn).


## Background

Geth is an Ethereum client written in Go. This means running Geth turns a computer into an Ethereum node. Ethereum is a peer-to-peer network where information is shared directly between nodes rather than being managed by a central server. The information shared between nodes is packaged into discrete blocks. Nodes compete to generate a new block because they are rewarded for doing so in Ethereum's native token, ether (ETH). On receiving a new block, each node adds it to their database. The sequence of discrete blocks is called a "blockchain". The information provided in each block is used by Geth to update its "state" - the ether balance of each account on Ethereum. There are two types of account: externally-owned accounts (EOAs) and contract accounts. Contract accounts execute contract code when they receive transactions. EOAs are accounts that users manage locally in order to sign and submit transactions. Each EOA is a public-private key pair, where the public key is used to derive a unique address for the user and the private key is used to protect the account and securely sign messages. Therefore, in order to use Ethereum, it is first necessary to generate an EOA (hereafter, "account").

Read more about Ethereum accounts [here](https://ethereum.org/en/developers/docs/accounts/).


## Step 1: Generating accounts

There are several methods for generating accounts in Geth. This tutorial demonstrates how to generate accounts using Clef, as this is considered best practise, largely because it decouples the users key management from a specific Geth implementation, making it more modular and flexible. It can also be run from secure USB sticks or virtual machines, offering security benefits. For convenience, this tutorial will execute Clef on the same computer that will also run Geth, although more secure options are available (see [here](https://github.com/ethereum/go-ethereum/blob/master/cmd/clef/docs/setup.md)).

If you installed Geth from the source code, it is necessary to run `make all` as opposed to `make geth` to build executable files for the developer tools, including Clef. These executables are saved in `~/go-ethereum/build/bin`. In order to run these executables it is first necessary to navigate to that directory or move the files to a more convenient location to run from, e.g. the top level project directory.

The following commands move the files to the top level directory:

Linux: 

```shell

cd ~/go-ethereum
mv ./build/bin/* ./

```

Windows (Powershell):

```shell

Set-Location -Path C:\go-ethereum
Move-Item C:\go-ethereum\build\bin\* C:\go-ethereum\

```


A new account can now be created using Clef. An account is a pair of keys (public and private). Clef needs to know where to save these keys to so that they can be retrieved later. This information is passed to Clef as an argument. This is achieved using the following command:

```shell

./clef newaccount --keystore geth-tutorial/keystore

```

The specific function from Clef that generates new accounts is `newaccount` and it accepts a parameter, `--keystore`, that tells it where to store the newly generated keys. In this example the keystore location is a new directory that will be created automatically: `geth-tutorial/keystore`. Clef will return the following result in the terminal:

```terminal
WARNING!

Clef is an account management tool. It may, like any software, contain bugs.

Please take care to
- backup your keystore files,
- verify that the keystore(s) can be opened with your password.

Clef is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY;
without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR
PURPOSE. See the GNU General Public License for more details.

Enter 'ok' to proceed:
>
```

This is important information. The `geth-tutorial/keystore` directory will soon contain a secret key that can be used to access any funds held in the new account. If it is compromised, the funds can be stolen. If it is lost, there is no way to retrieve the funds. This tuitorial will only use dummy funds with no real world value, but when these steps are repeated on Ethereum mainnet is critical that the keystore is kept secure and backed up.


Typing `ok` into the terminal and pressign `enter` causes Clef to prompt for a password. Clef requires a password that is at least 10 characters long, and best practise would be to use a combination of numbers, characters and special characters. Entering a suitable password and pressing `enter` returns the following result to the terminal:


```terminal
-----------------------
DEBUG[02-10|13:46:46.436] FS scan times                            list="92.081µs" set="12.629µs" diff="2.129µs"
INFO [02-10|13:46:46.592] Your new key was generated               address=0xCe8dBA5e4157c2B284d8853afEEea259344C1653
WARN [02-10|13:46:46.595] Please backup your key file!             path=keystore:///.../geth-tutorial/keystore/UTC--2022-02-07T17-19-56.517538000Z--ca57f3b40b42fcce3c37b8d18adbca5260ca72ec
WARN [02-10|13:46:46.595] Please remember your password!
Generated account 0xCe8dBA5e4157c2B284d8853afEEea259344C1653
```

It is important to save the account address and the password somewhere secure. They will be used again later in this tutorial. Please note that the account address shown in the code snippets above and later in this tutorials are examples - those generated by followers of this tutorial will be different. The account generated above can be used as the main account throughout the remainder of this tutorial. However in order to demonstrate transactions between accounts it is also necessary to have a second account. A second account can be added to the same keystore by precisely repeating the previous steps, providing the same password.

## Step 2:  Start Clef

The previous commands used Clef's `newaccount` function to add new key pairs to the keystore. Clef uses the private key(s) saved in the keystore is used to sign transactions. In order to do this, Clef needs to be started and left running while Geth is running simultaneously, so that the two programs can communicate between one another. 

To start Clef, run the Clef executable passing as arguments the keystore file location, config directory location and a chain ID. The config dirctory was automatically created inside the `geth-tutorial` directory during the previous step. The [chain ID](https://chainlist.org/) is an integer that defines which Ethereum network to connect to. Ethereum mainnet has chain ID 1. In this tutorial the Chain ID is 11155111 which is for the Sepolia testnet. It is very important that this chain ID parameter is set to 11155111. The following command starts Clef on Sepolia:

```shell

./clef --keystore geth-tutorial/keystore --configdir geth-tutorial/clef --chainid 11155111

```

After running the command above, Clef requests the user to type “ok” to proceed. On typing "ok" and pressing enter, Clef returns the following to the terminal:  

A successful call will give you the result below:

```terminal
INFO [02-10|13:55:30.812] Using CLI as UI-channel
INFO [02-10|13:55:30.946] Loaded 4byte database                    embeds=146,841 locals=0 local=./4byte-custom.json
WARN [02-10|13:55:30.947] Failed to open master, rules disabled    err="failed stat on geth-tutorial/clef/masterseed.json: stat geth-tutorial/clef/masterseed.json: no such file or directory"
INFO [02-10|13:55:30.947] Starting signer                          chainid=11155111 keystore=geth-tutorial/keystore light-kdf=false advanced=false
DEBUG[02-10|13:55:30.948] FS scan times                            list="133.35µs" set="5.692µs" diff="3.262µs"
DEBUG[02-10|13:55:30.970] Ledger support enabled
DEBUG[02-10|13:55:30.973] Trezor support enabled via HID
DEBUG[02-10|13:55:30.976] Trezor support enabled via WebUSB
INFO [02-10|13:55:30.978] Audit logs configured                    file=audit.log
DEBUG[02-10|13:55:30.981] IPCs registered                          namespaces=account
INFO [02-10|13:55:30.984] IPC endpoint opened                      url=geth-tutorial/clef/clef.ipc
------- Signer info -------
* intapi_version : 7.0.1
* extapi_version : 6.1.0
* extapi_http : n/a
* extapi_ipc : geth-tutorial/clef/clef.ipc
```

This result indicates that Clef is running. This terminal should be left running for the duration of this tutorial. If the tutorial is stopped and restarted later Clef must also be restarted by running the previous command.

## Step 3:  Start Geth

Geth is the Ethereum client that will connect the computer to the Ethereum network. In this tutorial the network is Sepolia, an Ethereum testnet. testnets are used to test Ethereum client software and smart contracts in an environment where no real-world value is at risk. To start Geth, run the Geth executable file passing argument that define the data directory (where Geth should save blockchain data), signer (points Geth to Clef), the network ID and the sync mode. For this tutorial, snap sync is recommended (see [here](https://blog.ethereum.org/2021/03/03/geth-v1-10-0/) for reasons why). The final argument passed to Geth is the `--http` flag. This enables the http-rpc server that allows external programs to interact with Geth by sending it http requests. By default the http server is only exposed locally using port 8545: `localhost:8545`.

The following command should be run in a new terminal, separate to the one running Clef: 

```shell

./geth --datadir geth-tutorial --signer=geth-tutorial/clef/clef.ipc --sepolia --syncmode light --http

```

Running the above command starts Geth. The terminal should rapidly fill with status updates, starting with:


```terminal
INFO [02-10|13:59:06.649] Starting Geth on Sepolia testnet...
INFO [02-10|13:59:06.649] Dropping default light client cache      provided=1024 updated=128
INFO [02-10|13:59:06.652] Maximum peer count                       ETH=50 LES=0 total=50
INFO [02-10|13:59:06.655] Using external signer                    url=geth-tutorial/clef/clef.ipc
INFO [02-10|13:59:06.660] Set global gas cap                       cap=50,000,000
INFO [02-10|13:59:06.661] Allocated cache and file handles         database=/.../geth-tutorial/geth/chaindata cache=64.00MiB handles=5120
INFO [02-10|13:59:06.855] Persisted trie from memory database      nodes=361 size=51.17KiB time="643.54µs" gcnodes=0 gcsize=0.00B gctime=0s livenodes=1 livesize=0.00B
INFO [02-10|13:59:06.855] Initialised chain configuration          config="{ChainID: 11155111 Homestead: 0 DAO: <nil> DAOSupport: true EIP150: 0 EIP155: 0 EIP158: 0 Byzantium: 0 Constantinople: 0 Petersburg: 0 Istanbul: 1561651, Muir Glacier: <nil>, Berlin: 4460644, London: 5062605, Arrow Glacier: <nil>, MergeFork: <nil>, Engine: clique}"
INFO [02-10|13:59:06.862] Added trusted checkpoint                 block=5,799,935 hash=2de018..c32427
INFO [02-10|13:59:06.863] Loaded most recent local header          number=6,340,934 hash=483cf5..858315 td=9,321,576 age=2d9h29m
INFO [02-10|13:59:06.867] Configured checkpoint oracle             address=0x18CA0E045F0D772a851BC7e48357Bcaab0a0795D signers=5 threshold=2
INFO [02-10|13:59:06.867] Gasprice oracle is ignoring threshold set threshold=2
WARN [02-10|13:59:06.869] Unclean shutdown detected                booted=2022-02-08T04:25:08+0100 age=2d9h33m
INFO [02-10|13:59:06.870] Starting peer-to-peer node               instance=Geth/v1.10.15-stable/darwin-amd64/go1.17.5
INFO [02-10|13:59:06.995] New local node record                    seq=1,644,272,735,880 id=d4ffcd252d322a89 ip=127.0.0.1 udp=30303 tcp=30303
INFO [02-10|13:59:06.996] Started P2P networking                   self=enode://4b80ebd341b5308f7a6b61d91aa0ea31bd5fc9e0a6a5483e59fd4ea84e0646b13ecd289e31e00821ccedece0bf4b9189c474371af7393093138f546ac23ef93e@127.0.0.1:30303
INFO [02-10|13:59:06.997] IPC endpoint opened                      url=/.../geth-tutorial/geth.ipc
INFO [02-10|13:59:06.998] HTTP server started                      endpoint=127.0.0.1:8545 prefix= cors= vhosts=localhost
WARN [02-10|13:59:06.998] Light client mode is an experimental feature
WARN [02-10|13:59:06.999] Failed to open wallet                    url=extapi://geth-tutorial/clef/cle.. err="operation not supported on external signers"
INFO [02-10|13:59:08.793] Block synchronisation started
```

This indicates that Geth has started up and is searching for peers to connect to. Once it finds peers it can request block headers from them, starting at the genesis block for the Goerli blockchain. Geth continues to download blocks sequentially, saving the data in leveldb files in `/go-ethereum/geth-tutorial/geth/chaindata/`. This is confirmed by the logs printed to the terminal. There should be a rapidly-growing sequence of logs in the terminal with the following syntax:

```terminal

INFO [04-29][15:54:09.238] Looking for peers             peercount=2 tried=0 static=0
INFO [04-29][15:54:19.393] Imported new block headers    count=2 elapsed=1.127ms  number=996288  hash=09f1e3..718c47 age=13h9m5s
INFO [04-29][15:54:19:656] Imported new block receipts   count=698  elapsed=4.464ms number=994566 hash=56dc44..007c93 age=13h9m9s

```

These logs indicate that Geth is running as expected. Sending an empty curl request to the http server provides a quick way to confirm that this too has been started without any issues. In a third terminal, run:

```shell

curl http://localhost:8545

```

If there is no error message reported to the terminal, everything is OK. Geth must be running in order for a user to interact with the Ethereum network. If this terminal is closed down then Geth must be restarted in a new terminal. Geth can be started and stopped easily, but it must be running for any interaction with Ethereum to take place. To shut down Geth, simply press `CTRL+C` in the Geth terminal. To start it again, run the previous command `geth --datadir ... ...`.


## Step 4:  Get Goerli Testnet Ether

In order to make some transactions, the user must fund their account with ether. On Ethereum mainnet, ether can only be obtained in three ways: 1) by receiving it as a reward for mining/validating; 2) receiving it in a transfer from another Ethereum user or contract; 3) sending it from an exchange, having paid for it with fiat money. On Ethereum testnets, the ether has no real world value so it can be made freely available via faucets. Faucets allow users to request a transfer of testnet ether to their account.

The address generated by Clef in Step 1 can be pasted into the Goerli ether faucet [here](https://fauceth.komputing.org/?chain=1115511). 


## Step 5: Interact with Geth via IPC or RPC

For interacting with the blockchain, Geth provides JSON-RPC APIs. A good way to get
started with the API is by using the Geth JavaScript console. The console gives you a
JavaScript environment similar to node.js and comes with Geth.

You can connect to the Geth node using HTTP or IPC.

- IPC (Inter-Process Communication): This provides unrestricted access to all APIs,
  but only works when you are running the console on the same host as the geth node.
- HTTP: This connection method provides access to the `eth`, `web3` and `net` method
  namespaces. We will be using HTTP for this guide.

To connect to Geth using the console, open a new terminal and run this command:

```shell
geth attach http://127.0.0.1:8545
```

The `attach` subcommand starts the console and should print a welcome message similar to
the text shown below:

```terminal
Welcome to the Geth JavaScript console!

instance: Geth/v1.10.15-stable/darwin-amd64/go1.17.5
at block: 6354736 (Thu Feb 10 2022 14:01:46 GMT+0100 (WAT))
 modules: eth:1.0 net:1.0 rpc:1.0 web3:1.0

To exit, press ctrl-d or type exit
```

### Checking your test account balance.

Run this command in the JavaScript console to check the ether balance of the test account:

```javascript
web3.fromWei(eth.getBalance("0xca57F3b40B42FCce3c37B8D18aDBca5260ca72EC"), "ether")
```

**Result:**

```terminal
> 0.1
```

### Getting the list of accounts

Run the command below to get the list of accounts in your keystore.

 ```javascript
 eth.accounts
 ```

**Note: Since the accounts are provided by Clef in this tutorial, you must accept the
account list request in the terminal window running Clef:***

```terminal
-------- List Account request--------------
A request has been made to list all accounts.
You can select which accounts the caller can see
  [x] 0xca57F3b40B42FCce3c37B8D18aDBca5260ca72EC
    URL: keystore:///.../geth-tutorial/keystore/UTC--2022-02-07T17-19-56.517538000Z--ca57f3b40b42fcce3c37b8d18adbca5260ca72ec
  [x] 0xCe8dBA5e4157c2B284d8853afEEea259344C1653
    URL: keystore:///.../geth-tutorial/keystore/UTC--2022-02-10T12-46-45.265592000Z--ce8dba5e4157c2b284d8853afeeea259344c1653
-------------------------------------------
Request context:
        NA -> ipc -> NA

Additional HTTP header data, provided by the external caller:
        User-Agent: ""
        Origin: ""
Approve? [y/N]:
> y

```

Now you should get a result in the JavaScript console.

**Result:**

```terminal
["0xca57f3b40b42fcce3c37b8d18adbca5260ca72ec", "0xce8dba5e4157c2b284d8853afeeea259344c1653"]
```

If you didn't get a result (e.g. an exception was raised), it may be because the account
listing request timed out while you were entering the password. Just try again in this
case.

### Send ether to another account

Run the command below to transfer 0.01 ether to the other account you created.

```javascript
eth.sendTransaction({
    from: "0xca57f3b40b42fcce3c37b8d18adbca5260ca72ec",
    to: "0xce8dba5e4157c2b284d8853afeeea259344c1653",
    value: web3.toWei(0.01, "ether")
})
```

**Again, since the test account is stored by Clef, you must confirm the request in the Clef
terminal window.**

Clef will prompt you to approve the transaction, and when you do, it will ask you for the
password for the account you are sending the ether from. If the password is correct, Geth
proceeds with the transaction.

```terminal
--------- Transaction request-------------
to:    0xCe8dBA5e4157c2B284d8853afEEea259344C1653
from:               0xca57F3b40B42FCce3c37B8D18aDBca5260ca72EC [chksum ok]
value:              10000000000000000 wei
gas:                0x5208 (21000)
maxFeePerGas:          2425000057 wei
maxPriorityFeePerGas:  2424999967 wei
nonce:    0x3 (3)
chainid:  0x5
Accesslist

Request context:
        NA -> ipc -> NA

Additional HTTP header data, provided by the external caller:
        User-Agent: ""
        Origin: ""
-------------------------------------------
Approve? [y/N]:
> y
## Account password

Please enter the password for account 0xca57F3b40B42FCce3c37B8D18aDBca5260ca72EC
>
```

After approving the transaction, you will see the below screen in the Clef terminal.

```terminal
-----------------------
Transaction signed:
 {
    "type": "0x2",
    "nonce": "0x3",
    "gasPrice": null,
    "maxPriorityFeePerGas": "0x908a901f",
    "maxFeePerGas": "0x908a9079",
    "gas": "0x5208",
    "value": "0x2386f26fc10000",
    "input": "0x",
    "v": "0x0",
    "r": "0x66e5d23ad156e04363e68b986d3a09e879f7fe6c84993cef800bc3b7ba8af072",
    "s": "0x647ff82be943ea4738600c831c4a19879f212eb77e32896c05055174045da1bc",
    "to": "0xce8dba5e4157c2b284d8853afeeea259344c1653",
    "chainId": "0x5",
    "accessList": [],
    "hash": "0x99d489d0bd984915fd370b307c2d39320860950666aac3f261921113ae4f95bb"
  }

```

**Result:**

You will get the transaction hash response in the Geth JavaScript console after approving
the transaction in Clef.

```terminal
"0x99d489d0bd984915fd370b307c2d39320860950666aac3f261921113ae4f95bb"
```

### Checking the transaction hash

To get the transaction hash, Run the command below.

```javascript
eth.getTransaction("0x99d489d0bd984915fd370b307c2d39320860950666aac3f261921113ae4f95bb")
```

If successful, you will get a response like the one below:

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

## Access using low-level HTTP

In this part of the tutorial, we will show how to access the JSON-RPC API using curl.

### Checking account balance

To check account balance, use the command below.

```shell
curl -X POST http://127.0.0.1:8545 \
  -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0", "method":"eth_getBalance", "params":["0xca57f3b40b42fcce3c37b8d18adbca5260ca72ec","latest"], "id":1}'
```

A successful call will return a response like the one below:

```terminal
{"jsonrpc":"2.0","id":1,"result":"0xcc445d3d4b89390"}
```

Note that the value returned is in hexadecimal and WEI. To get the balance in ether,
convert to decimal and divide by 10^18.

### Checking the account list

Run the command below to get the list of all accounts.

```shell
curl -X POST http://127.0.0.1:8545 \
    -H "Content-Type: application/json" \
   --data '{"jsonrpc":"2.0", "method":"eth_accounts","params":[], "id":1}'
```

Note: you will need to confirm this request in the Clef terminal window.a

**Response:**

```terminal
{"jsonrpc":"2.0","id":1,"result":["0xca57f3b40b42fcce3c37b8d18adbca5260ca72ec"]}
```

### Sending Transactions

```shell
curl -X POST http://127.0.0.1:8545 \
    -H "Content-Type: application/json" \
   --data '{"jsonrpc":"2.0", "method":"eth_sendTransaction", "params":[{"from": "0xca57f3b40b42fcce3c37b8d18adbca5260ca72ec","to": "0xce8dba5e4157c2b284d8853afeeea259344c1653","value": "0x2386F26FC10000"}], "id":1}'
```

A successful call will return a response containing the transaction hash.

```terminal
{"jsonrpc":"2.0","id":5,"result":"0xac8b347d70a82805edb85fc136fc2c4e77d31677c2f9e4e7950e0342f0dc7e7c"}
```
