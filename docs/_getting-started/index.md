---
title: Getting Started with Geth
permalink: docs/getting-started
sort_key: A
---


To use Geth, you need to install it first. You can install Geth in various ways that you
can find in the “[Install and Build](install-and-build/installing-geth)” section. These
include installing it via your favorite package manager, downloading a standalone
pre-built binary, running it as a docker container, or building it yourself.

We assume you have Geth installed for this guide and are ready to find out how to use it.
The guide shows you how to create accounts, sync to a network, and send transactions
between accounts. This guide also uses [Clef](clef/tutorial), our preferred tool for
signing transactions with Geth.

#### Networks

You can connect a Geth node to several different networks using the network name as an
argument. These include the main Ethereum network, [a private network](getting-started/private-net) you create,
and three test networks:

- **Ropsten:** Proof-of-work test network
- **Rinkeby:** Proof-of-authority test network
- **Görli:** Proof-of-authority test network

#### Sync modes

You can start Geth in one of three different sync modes using the `--syncmode "<mode>"`
argument that determines what sort of node it is in the network. These are:

- **Full:** Downloads all blocks (including headers, transactions, and receipts) and
  generates the state of the blockchain incrementally by executing every block.
- **Snap:** (Default): Downloads all blocks and a recent version of the state.
- **Light:** The node only downloads a few recent block headers, and downloads
  other data on-demand. See this [page](../interface/les) for more info.

For this guide, we will use `light` sync.

### Requirements for following this guide

- Experience using the command line
- Basic knowledge about Ethereum and testnets
- Basic knowledge about HTTP and JavaScript

## Step 1: Generate accounts

Use the command below to generate an account. When you create a new account with Clef, it
will generate a new private key, encrypt it according to the [web3 keystore spec](https://github.com/ethereum/wiki/wiki/Web3-Secret-Storage-Definition),
and store it in the keystore directory.

```shell
clef newaccount --keystore geth-tutorial/keystore
```

It will give you the result below:

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

Type “ok” and press the enter key. Next, the Clef will prompt for a password. Enter your
desired password and hit the enter key to get the result below:

```terminal
-----------------------
DEBUG[02-10|13:46:46.436] FS scan times                            list="92.081µs" set="12.629µs" diff="2.129µs"
INFO [02-10|13:46:46.592] Your new key was generated               address=0xCe8dBA5e4157c2B284d8853afEEea259344C1653
WARN [02-10|13:46:46.595] Please backup your key file!             path=keystore:///.../geth-tutorial/keystore/UTC--2022-02-07T17-19-56.517538000Z--ca57f3b40b42fcce3c37b8d18adbca5260ca72ec
WARN [02-10|13:46:46.595] Please remember your password!
Generated account 0xCe8dBA5e4157c2B284d8853afEEea259344C1653
```

Save your password and the generated account address because you will need them later in
this tutorial.

**In the remainder of this guide, we will use this account as the main account for
testing:**

```shell
0xca57F3b40B42FCce3c37B8D18aDBca5260ca72EC
```

When you follow the tutorial locally, your account address will be different and you will
need to supply your own account address in all command invocations.

**Please generate a second account by repeating this step one more time.**

## Step 2:  Start Clef

To start clef, open a new terminal and run the command below. Keeping clef running is
required for the other steps because it signs transactions.

```shell
clef --keystore geth-tutorial/keystore --configdir geth-tutorial/clef --chainid 5
```

> Note: geth-tutorial is the directory holding your keystore.

After running the command above, clef will request you to type “ok” to proceed.

A successful call will give you the result below:

```terminal
INFO [02-10|13:55:30.812] Using CLI as UI-channel
INFO [02-10|13:55:30.946] Loaded 4byte database                    embeds=146,841 locals=0 local=./4byte-custom.json
WARN [02-10|13:55:30.947] Failed to open master, rules disabled    err="failed stat on geth-tutorial/clef/masterseed.json: stat geth-tutorial/clef/masterseed.json: no such file or directory"
INFO [02-10|13:55:30.947] Starting signer                          chainid=5 keystore=geth-tutorial/keystore light-kdf=false advanced=false
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

## Step 3:  Start Geth

To start geth, open a new terminal and run the command below. Keeping geth running is
required for the other steps because the command below starts the HTTP server that we will
be interacting with.

```shell
geth --datadir geth-tutorial --signer=geth-tutorial/clef/clef.ipc --goerli --syncmode light --http
```

A successful call will print log messages like the following:

```terminal
INFO [02-10|13:59:06.649] Starting Geth on Görli testnet...
INFO [02-10|13:59:06.649] Dropping default light client cache      provided=1024 updated=128
INFO [02-10|13:59:06.652] Maximum peer count                       ETH=0 LES=10 total=50
INFO [02-10|13:59:06.655] Using external signer                    url=geth-tutorial/clef/clef.ipc
INFO [02-10|13:59:06.660] Set global gas cap                       cap=50,000,000
INFO [02-10|13:59:06.661] Allocated cache and file handles         database=/.../geth-tutorial/geth/lightchaindata cache=64.00MiB handles=5120
INFO [02-10|13:59:06.794] Allocated cache and file handles         database=/.../geth-tutorial/geth/les.client cache=16.00MiB handles=16
INFO [02-10|13:59:06.855] Persisted trie from memory database      nodes=361 size=51.17KiB time="643.54µs" gcnodes=0 gcsize=0.00B gctime=0s livenodes=1 livesize=0.00B
INFO [02-10|13:59:06.855] Initialised chain configuration          config="{ChainID: 5 Homestead: 0 DAO: <nil> DAOSupport: true EIP150: 0 EIP155: 0 EIP158: 0 Byzantium: 0 Constantinople: 0 Petersburg: 0 Istanbul: 1561651, Muir Glacier: <nil>, Berlin: 4460644, London: 5062605, Arrow Glacier: <nil>, MergeFork: <nil>, Engine: clique}"
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

> **Note:** keep this terminal window open.

## Step 4:  Get Goerli Testnet Ether

We will now top up the test account ether balance, so that it can send transactions. You
can get Goerli testnet ether at several faucet sites. We recommend you try one of these
faucets:

- <https://faucets.chain.link/goerli>
- <https://fauceth.komputing.org/?chain=5>

Just enter your address on one of those sites and follow the instructions provided.

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
