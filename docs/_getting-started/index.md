---
title: Getting Started with Geth
permalink: docs/getting-started
sort_key: A
---


To use Geth, you need to install it first. You can install Geth in various ways that you can find in the “[Install and Build](install-and-build/installing-geth)” section. These include installing it via your favorite package manager, downloading a standalone pre-built binary, running it as a docker container, or building it yourself.

We assume you have Geth installed for this guide and are ready to find out how to use it. The guide shows you how to create accounts, sync to a network, and send transactions between accounts. This guide uses [Clef](clef/tutorial), our preferred tool for signing transactions with Geth, and will replace Geth's account management.

## Two Important Terms In Geth:

### Networks
You can connect a Geth node to several different networks using the network name as an argument. These include the main Ethereum network, [a private network](getting-started/private-net) you create, and three test networks that use different consensus algorithms:

-   **Ropsten:** Proof-of-work test network
-   **Rinkeby:** Proof-of-authority test network
-   **Görli:** Proof-of-authority test network

For this guide, you will use the Görli network and the default port is 8545, so you need to enable at least outgoing access from your node to that port.
### Sync modes
You can start Geth in one of three different sync modes using the `--syncmode "<mode>"` argument that determines what sort of node it is in the network.
These are:

- **Full:** Downloads all blocks (including headers, transactions, and receipts) and generates the state of the blockchain incrementally by executing every block.
- **Snap:** (Default): Same functionality as fast, but with a faster algorithm.
- **Light:** When using the "light" synchronization, the node only downloads a few recent block headers, block data and syncs quickly..

For this guide, you will use a `light` sync:

### Requirements:

- Experience using the command line
- Basic knowledge about Ethereum and testnets
- Basic knowledge about HTTP and JavaScript

## Step 1: Generate account

Use the command below to generate an account.
> **Note:** you will need to create two accounts for this guide.

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

Enter “ok” and press the enter key. Next, the system will request the below action.

**Please enter a password for the new account to be created (attempt 0 of 3)**

Enter your desired password and hit the enter key to get the result below:

```terminal
-----------------------
DEBUG[02-10|13:46:46.436] FS scan times                            list="92.081µs" set="12.629µs" diff="2.129µs"
INFO [02-10|13:46:46.592] Your new key was generated               address=0xCe8dBA5e4157c2B284d8853afEEea259344C1653
WARN [02-10|13:46:46.595] Please backup your key file!             path=/Users/wisdomnwokocha/Documents/GitHub/GethExample/geth-tutorial/keystore/UTC--2022-02-10T12-46-45.265592000Z--ce8dba5e4157c2b284d8853afeeea259344c1653
WARN [02-10|13:46:46.595] Please remember your password! 
Generated account 0xCe8dBA5e4157c2B284d8853afEEea259344C1653
```

Save your password and the generated account because you will need them later in this tutorial.

**The Generated account:**

```shell
0xCe8dBA5e4157c2B284d8853afEEea259344C1653
```
## Step 2:  Start Clef

To start clef, open a new terminal and run the command below. Keeping clef running is required for the other steps because it signs transactions.

```shell
clef --keystore geth-tutorial/keystore --configdir geth-tutorial/clef --chainid 5
```

> Note:  geth-tutorial folder is holding your keystore.

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
To start geth, open a new terminal and run the command below. Keeping geth running is required for the other steps because the command below starts the HTTP server.

```shell
geth --datadir geth-tutorial --signer=geth-tutorial/clef/clef.ipc --goerli --syncmode light --http
```


A successful call will give you the result below:

```terminal
INFO [02-10|13:59:06.649] Starting Geth on Görli testnet... 
INFO [02-10|13:59:06.649] Dropping default light client cache      provided=1024 updated=128
INFO [02-10|13:59:06.652] Maximum peer count                       ETH=0 LES=10 total=50
INFO [02-10|13:59:06.655] Using external signer                    url=geth-tutorial/clef/clef.ipc
INFO [02-10|13:59:06.660] Set global gas cap                       cap=50,000,000
INFO [02-10|13:59:06.661] Allocated cache and file handles         database=/Users/wisdomnwokocha/Documents/GitHub/GethExample/geth-tutorial/geth/lightchaindata cache=64.00MiB handles=5120
INFO [02-10|13:59:06.794] Allocated cache and file handles         database=/Users/wisdomnwokocha/Documents/GitHub/GethExample/geth-tutorial/geth/les.client cache=16.00MiB handles=16
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
INFO [02-10|13:59:06.997] IPC endpoint opened                      url=/Users/wisdomnwokocha/Documents/GitHub/GethExample/geth-tutorial/geth.ipc
INFO [02-10|13:59:06.998] HTTP server started                      endpoint=127.0.0.1:8545 prefix= cors= vhosts=localhost
WARN [02-10|13:59:06.998] Light client mode is an experimental feature 
WARN [02-10|13:59:06.999] Failed to open wallet                    url=extapi://geth-tutorial/clef/cle.. err="operation not supported on external signers"
INFO [02-10|13:59:08.793] Block synchronisation started 
```

> **Note:** keep this terminal open.

## Step 4:  Get Goerli Testnet Ether

The following sites give free goerli ether:

- [faucet 1](https://faucets.chain.link/goerli)
- [faucet 2](https://fauceth.komputing.org/?chain=5)

## Step 5: Interact with Geth via IPC or RPC

You can interact with Geth in two ways: Directly with the node using the JavaScript console over IPC or connecting to the node remotely over HTTP using RPC.

- IPC (Inter-Process Communication):
    This allows you to do more, especially when creating and interacting with accounts, but you need direct access to the node.
- RPC (Remote Procedure Call):
    This allows remote applications to access your node but has limitations and security considerations. By default, it only allows access to the eth and shh namespaces methods. Find out how to override this setting [in the RPC docs](rpc/server#http-server).

## Step 6: Using IPC

**→ Connect to console**

To connect to the IPC console, open a new terminal and run the command below. 

```shell
geth attach http://127.0.0.1:8545
```
The `attach` subcommand attaches to the console to an already-running geth instance and open the Geth javascript console as shown below.

```terminal
Welcome to the Geth JavaScript console!

instance: Geth/v1.10.15-stable/darwin-amd64/go1.17.5
at block: 6354736 (Thu Feb 10 2022 14:01:46 GMT+0100 (WAT))
 modules: eth:1.0 net:1.0 rpc:1.0 web3:1.0

To exit, press ctrl-d or type exit
```


**→ Check account balance**

> **Note:** the value comes in wei.

**Syntax:**

```javascript
web3.fromWei(eth.getBalance("<ADDRESS_1>"),"ether")
```

Run the command below to check your account balance.

```javascript
web3.fromWei(eth.getBalance("0xca57f3b40b42fcce3c37b8d18adbca5260ca72ec"),"ether")
```

**Result:**

```terminal
> 0.1
```


**→ Check list of accounts**

**step 1:**

Run the command below to get the list of accounts in your keystore.

 ```javascript
 eth.accounts
 ```

**step 2:** 
Accept request in your Clef terminal. 

The command in step 1 will request approval from the clef terminal before showing the list of accounts.

```terminal
-------- List Account request--------------
A request has been made to list all accounts. 
You can select which accounts the caller can see
  [x] 0xca57F3b40B42FCce3c37B8D18aDBca5260ca72EC
    URL: keystore:///Users/wisdomnwokocha/Documents/GitHub/GethExample/geth-tutorial/keystore/UTC--2022-02-07T17-19-56.517538000Z--ca57f3b40b42fcce3c37b8d18adbca5260ca72ec
  [x] 0xCe8dBA5e4157c2B284d8853afEEea259344C1653
    URL: keystore:///Users/wisdomnwokocha/Documents/GitHub/GethExample/geth-tutorial/keystore/UTC--2022-02-10T12-46-45.265592000Z--ce8dba5e4157c2b284d8853afeeea259344c1653
-------------------------------------------
Request context:
        NA -> ipc -> NA

Additional HTTP header data, provided by the external caller:
        User-Agent: ""
        Origin: ""
Approve? [y/N]:
> y

```

**Result:**

```terminal
["0xca57f3b40b42fcce3c37b8d18adbca5260ca72ec", "0xce8dba5e4157c2b284d8853afeeea259344c1653"]
```


**→ Send ETH to account**

Send 0.01 ETH from the account you added the free eth to the second account you created.

**Syntax:**

```javascript
eth.sendTransaction({from:"<ADDRESS_1>",to:"<ADDRESS_2>", value: web3.toWei(0.01,"ether")})
```

**step 1:** 

Run the command below to transfer 0.01 ether to the other account you created.

```javascript
eth.sendTransaction({
    from:"0xca57f3b40b42fcce3c37b8d18adbca5260ca72ec",
    to:"0xce8dba5e4157c2b284d8853afeeea259344c1653", 
    value: web3.toWei(0.01,"ether")
    })
```

**step 2:** Accept request in your Clef terminal.

Clef will prompt you to approve the transaction, and when you do, it will ask you for the password for the account you are sending the ETH from; if the password is correct, Geth proceeds with the transaction.

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
Transaction signed:
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


**Step 3** Your Terminal Result, 
You will get a transaction hash as a response after approving the transaction in the clef terminal.

```terminal
"0x99d489d0bd984915fd370b307c2d39320860950666aac3f261921113ae4f95bb"
```


**→ Check Transaction hash**

**Syntax:**

```javascript
eth.getTransaction("hash id")
```

To get the transaction hash, Run the command below.

```javascript
eth.getTransaction("0xa2b547d8742e345fa5f86f017d9da38c4a19cacee91e85191a57c0c7e420d187")
```

If successful, you will get the below response.

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

## Step 7: Using RPC

**→ Check account balance**

**Syntax:**

```
    curl -X POST http://http://127.0.0.1:8545 \
        -H "Content-Type: application/json" \
       --data '{"jsonrpc":"2.0", "method":"eth_getBalance", "params":["<ADDRESS_1>","latest"], "id":1}'
```
 
 > **Note:** http://127.0.0.1:8545 this is the default address.
 
 To check your account balance, use the command below.

 ```
  curl -X POST http://127.0.0.1:8545 \
    -H "Content-Type: application/json" \
   --data '{"jsonrpc":"2.0", "method":"eth_getBalance", "params":["0xca57f3b40b42fcce3c37b8d18adbca5260ca72ec","latest"], "id":5}'
   ```

A successful call will return a response below:

```terminal
{"jsonrpc":"2.0","id":5,"result":"0xcc445d3d4b89390"}
```


So Geth returns the value without invoking Clef. Note that the value returned is in hexadecimal and WEI. To get the ETH value, convert to decimal and divide by 10^18.

**→ Check list of accounts**

Run the command below to get all the accounts.

```
curl -X POST http://127.0.0.1:8545 \
    -H "Content-Type: application/json" \
   --data '{"jsonrpc":"2.0", "method":"eth_accounts","params":[], "id":5}'
```

Follow the same step as the IPC Check account balance.

A successful call will return a response below:

```terminal
{"jsonrpc":"2.0","id":5,"result":["0xca57f3b40b42fcce3c37b8d18adbca5260ca72ec"]}
```


**→ Send ETH to accounts**

**Syntax:**
```
    curl -X POST http://http://127.0.0.1:8545 \
        -H "Content-Type: application/json" \
       --data '{"jsonrpc":"2.0", "method":"eth_sendTransaction", "params":[{"from": "<ADDRESS_1>","to": "<ADDRESS_2>","value": "0x9184e72a"}], "id":1}'
```

You need to convert eth to wei and get the hex value to send a transaction.

> **Example:**  0.0241 ether is 24100000000000000 wei, and would be encoded as the hex string "0x559ed283164000" in the JSON-RPC API.

**step 3:** Run the command below

```
curl -X POST http://127.0.0.1:8545 \
    -H "Content-Type: application/json" \
   --data '{"jsonrpc":"2.0", "method":"eth_sendTransaction", "params":[{"from": "0xca57f3b40b42fcce3c37b8d18adbca5260ca72ec","to": "0xce8dba5e4157c2b284d8853afeeea259344c1653","value": "0x2386F26FC10000"}], "id":5}'
```

A successful call will return a response below:

```terminal
{"jsonrpc":"2.0","id":5,"result":"0xac8b347d70a82805edb85fc136fc2c4e77d31677c2f9e4e7950e0342f0dc7e7c"}
```


