---
title: Getting Started with Geth
permalink: docs/getting-started
sort_key: A
---

To use Geth, you need to install it first. You can install Geth in a variety of ways that you can find in the “[Install and Build](install-and-build/installing-geth)” section. 
These include installing it via your favorite package manager, downloading a standalone pre-built binary, running it as a docker container, or building it yourself.

For this guide, we assume you have Geth installed and are ready to find out how to use it. 
The guide shows you how to create accounts, sync to a network, and then send transactions between accounts.
This guide uses [Clef](clef/tutorial), which is our preferred tool for signing transactions with Geth, and will replace Geth’s account management.

## Two Important terms in geth:

### Networks
You can connect a Geth node to several different networks using the network name as an argument. These include the main Ethereum network, [a private network](getting-started/private-net) you create, and three test networks that use different consensus algorithms:

-   Ropsten: Proof-of-work test network
-   Rinkeby: Proof-of-authority test network
-   Görli: Proof-of-authority test network

For this guide, you will use the Görli network. The default port is 8545, so you need to enable at least outgoing access from your node to that port.
### Sync modes
You can start Geth in one of three different sync modes using the `--syncmode "<mode>"` argument that determines what sort of node it is in the network.
These are:

- Full: Downloads all blocks (including headers, transactions, and receipts) and generates the state of the blockchain incrementally by executing every block.
- Snap (Default): Same functionality as fast, but with a faster algorithm.
- Light: Downloads all block headers, block data, and verifies some randomly.

For this tutorial, you will use a `light` sync:

### Prerequisites:

- Curl experience 
- Command line
- Basic Blockchain Knowledge

## Step 1: Open Terminal

You will need your system terminal to run the commands for this tutorial,

Use the command below to create an account 

![Create new account command](../../static/images/open_terminal.png)
## Step 2: Create accounts

Use the command below to create an account 
> **Note:** you will need to create two accounts for this guide

```javscript
clef newaccount --keystore geth-tutorial/keystore
```

It will give you the result below:

```
WARNING! 
Clef is an account management tool. It may, like any software, contain bugs. 

Please take care to 
— backup your keystore files,
 — verify that the keystore(s) can be opened with your password. 

Clef is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; 
without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR 
PURPOSE. See the GNU General Public License for more details. 

Enter 'ok' to proceed: 
> ok 
```

Enter “ok” and hit the enter key. Next, the system will request the below action.

**Please enter a password for the new account to be created (attempt 0 of 3)**

Enter your desired password and hit the enter key to get the result below:

```
INFO [02-07118:19:57.914] Your new key was generated                                       address=0xca57F3b40842FCce3c3713881848ca5260ce72EC
WAN. [02-07118:19:57.915] Please backup your key file, b40642fcce3c37b8d18adbca5260ca72ec  path=Users/wIsdommokochanocuments/LitHub/GethExample/getb-tutortalikeystore/UTC-2022-02-07717-19-58.5175380002—Ca57f3 
WAN! [02-07118:19:57.915] Please remember your password! 
Generated account 0xca57F3b40842FCce3c3713881848ca5260ce72EC 
wisdomnwokocha@wisdoms-MacBook-Pro GethExample %
```

Copy and save your password with the generated account somewhere safe; you will need it later in this tutorial.

**The Generated account:**

```javscript
0xca57F3b40B42FCce3c37B8D18aDBca5260ca72EC
```
## Step 3:  Start Clef

To start clef, open a new terminal and run the command below. Keeping clef running is required for the other steps to work.

```javscript
clef --keystore geth-tutorial/keystore --configdir geth-tutorial/clef --chainid 5
```

> Note:  geth-tutorial folder is the directory holding your keystore

after running the command above, the system will request you to type “ok” to proceed

A successful call will give you the result below:

```
INFO 102-07123:21:07.325] Using CLI as UI -channel 
INFO [02-07123:21:07.464] Loaded 4byte database                        embeds=146,841 locals=0 local=./4byte-custom.json 
NAM [02-07123:21:07.464] Failed to open master, rules disabled         err="failed stat on geth-tutorial/clef/masterseed.json: stat geth-tutorial/clef/masterseed.json no such file or directory"
INFO [02-07123:21:07.464] Starting signer                              chainld=5 keystore=geth-tutorial/keystore light-kdf=false advanced=false 
DEBUG[02-07123:21:07.4651 FS scan times                                 list=1.217485ms set="11.021ps. dIff=.3.3374s" 
DEBUG[02-07123:21:07.487] Ledger support enabled 
DEBUG[02-07123:21:07.489] Trezor support enabled via HID 
DEBUG[02-07123:21:07.492] Trezor support enabled via Nebusg 
INFO [02-07123:21:07.492] Audit logs configured                        file=audit.log 
DEBUG[02-07123:21:07.493] IPCs registered                              namespaces=account 
INFO [02-07123:21:07.494] IPC endpoint opened                          url=geth-tutorial/clef/clef.ipc   
------ Signer info -------- 
* extapi_version : 6.1.0 
* extapi_http : n/a
* extapi_ipc : geth-tutorial/clef/clef.ipc
* intapi_version : 7.0.1 
```

> **Note:** keep this terminal open.

## Step 4:  Start Geth
To start geth, open a new terminal and run the command below. It would be best if you did not close this terminal, always keep it running while working.

```javscript
geth --datadir geth-tutorial --signer=geth-tutorial/clef/clef.ipc --goerli --syncmode light --http
```


A successful call will give you the result below:

```

INFO (02-07 23:25:35.508] Starting Geth on Görli testnet...
INFO (02-07 123:25:35.508] Dropping default light client cache             provided=1024 updated=128 
INFO (02-07 23:25:35.510) Maximum peer count                               ETH=0 LES=10 total=50 
INFO (02-07 23:25:35.511] Using external signer                            url=geth-tutorial/clef/clef.ipc 
INFO (02-07 23:25:35.511] Set global gas cap                               cap=50,000,000
INFO (02-07 23:25:35.512] Allocated cache and file handles                 database=/Users/wisdomnwokocha/Documents/GitHub/GethExample/geth-tutorial/geth/lightchaindata cache=64.00MiB handles=5120 
INFO [02-07 23:25:35.546] Allocated cache and file handles                 database=/Users/wisdomnwokocha/Documents/GitHub/GethExample/geth-tutorial/geth/les.client cache=16.00MiB handles=16
INFO (02-07123:25:35.578) Writing custom genesis block 
INFO [02-07 23:25:35.584) Persisted trie from memory database              nodes=361 size=51.17KiB time=1.417193ms gcnodes=o gcsize=0.00B gctime=0s livenodes=1 livesize=0.00B
INFO [02-07 23:25:35.585] Initialised chain configuration                  config="{ChainID: 5 Homestead: 0 DAO: <nil> DAOSupport: true EIP150: 0 EIP155: 0 EIP158: 0 Byzantium: 0 Constantinople: 0

```

> **Note:** keep this terminal open.



## Step 5:  Get Goerli Testet Ether

The primary purpose of the faucet is to fund your testnet account to pay for gas fees for testing your project. 

The following sites gives free goerli faucets:

- [faucet 1](https://faucets.chain.link/goerli)
- [faucet 2](https://fauceth.komputing.org/?chain=5)

## Step 6: Interact with Geth via IPC or RPC

You can interact with Geth in two ways: Directly with the node using the JavaScript console over IPC or connecting to the node remotely over HTTP using RPC.

- IPC (Inter-Process Communication):
    allows you to do more, especially when creating and interacting with accounts, but you need direct access to the node.
- RPC (Remote Procedure Call):
     allows remote applications to access your node but has limitations and security considerations, and by default, only allows access to methods in the eth and shh namespaces. Find out how to override this setting [in the RPC docs](rpc/server#http-server).

## Step 7: Using IPC

**→ Connect to console**
Connect to the IPC console on a node from another terminal window, this will open the Geth javascript console
run the command below

```javscript
geth attach http://127.0.0.1:8545
```

Result after running the above command: 

```
Welcome to the Geth JavaScript console!
instance: Geth/v1.10.15-stable/darwin-amd64/go1.17.5 
at block: 6339763 (Mon Feb 07 2022 23:37:06 GMT+0100 (WAT) 
  modules: eth:1.0 net:1.0 rpc:1.0 web3:1.0

To exit, press ctrl-d or type exit
```


**→ Check account balance**

> **Note:** the value comes in wei
**Syntax:**

```javscript
web3.fromWei(eth.getBalance("<ADDRESS_1>"),"ether")
```

Run the command below to check your account balance

```javscript
web3.fromWei(eth.getBalance("0xca57F3b40B42FCce3c37B8D18aDBca5260ca72EC"),"ether")
```

**Result:**

```
> 0.1

```


**→ Check list of accounts**

**step 1:**
Run the command below to get the list of accounts in your keystore

 ```javascript
 eth.accounts
 ```

**step 2:** Accept request in your Clef terminal 

The command in step 1 will need approval from the terminal running clef, before showing the list of accounts.

```
-------- List Account request---- 
A request has been made to list all accounts. 
You can select which accounts the caller can see 
  [x] Oxca57F3b40B42FCce3c37B8D18aDBca5260ca72EC,
    URL: keystore:///Users/wisdomnwokocha/Documents/GitHub/GethExample/geth-tutorial/keystore/UTC--2022-02-07T17–19–56.517538000z--ca57f3b40b42fcce3c37b8d18adbca5260ca72ec!
------------------------------------------
Request context:
        NA -> ipc -> NA
Additional HTTP header data, provided by the external caller:
         User-Agent: ""
         Origin: "I 
Approve? [y/N]: 
> y

```


Approve the request by typing “y” and hit the enter key.

**Result:**

```
["0x92ac6226ccdb0d12003884c74d42a2436ebeb928"]
```


**→ Send ETH to account**

Send 0.01 ETH from the account that you added ETH to with the Görli faucet, to the second account you created.

**Syntax:**

```javscript
eth.sendTransaction({from:"<ADDRESS_1>",to:"<ADDRESS_2>", value: web3.toWei(0.01,"ether")})
```

**step 1:** 
Run the command below to transfer 0.01 ether to the other account you created

```javscript
eth.sendTransaction({
    from:"0xca57f3b40b42fcce3c37b8d18adbca5260ca72ec",
    to:"0x8EB19d8DF81a8B43a178207E23E9a57ff8cA61B1", 
    value: web3.toWei(0.01,"ether")
    })
```

**step 2:**
Accept request in your Clef terminal 

After running in step 1 command, Clef will prompt you to approve the transaction, and when you do, it will ask you for the password for the account you are sending the ETH from; if the password is correct, Geth proceeds with the transaction.

```
--------- Transaction request-------------
to:    0x1f7a76611939fbAcf7d2dAD2F864F6184BDCD690
from:               0xca57F3b40B42FCce3c37B8D18aDBca5260ca72EC [chksum ok]
value:              10000000000000000 wei
gas:                0x5208 (21000)
maxFeePerGas:          1500000014 wei
maxPriorityFeePerGas:  1500000000 wei
nonce:    0x2 (2)
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


After approving the transaction you will see the below screen in the Clef terminal

```
Transaction signed:
 {
    "type": "0x2",
    "nonce": "0x1",
    "gasPrice": null,
    "maxPriorityFeePerGas": "0x59682f00",
    "maxFeePerGas": "0x59682f0e",
    "gas": "0x5208",
    "value": "0x2386f26fc10000",
    "input": "0x",
    "v": "0x0",
    "r": "0xa54c83161e959bf8a03e30a5ed42a71563b0162fb4f9e5fc1bc426f312ef09e6",
    "s": "0x16cffa4d71274c6aa68c538d892a0b9a455ed28c504fa12f6b9fefc2ad92bfd0",
    "to": "0x8eb19d8df81a8b43a178207e23e9a57ff8ca61b1",
    "chainId": "0x5",
    "accessList": [],
    "hash": "0xa2b547d8742e345fa5f86f017d9da38c4a19cacee91e85191a57c0c7e420d187"
  }

```


**Step 1** Terminal Result, it will return a response that includes the transaction hash:

```
"Oxa2b547d8742e345fa5f86f017d9da38c4a19cacee91e85191a57c0c7e420d187"

```


**→ Check Transaction hash**

**Syntax:**

```javscript
eth.getTransaction("hash id")
```

A Transaction Hash (Tx Hash) is a record of successful transaction in a blockchain that can be accessed with unique address.

Run the command below.

```javscript
eth.getTransaction("0xa2b547d8742e345fa5f86f017d9da38c4a19cacee91e85191a57c0c7e420d187")
```

If successful, you will get the below response 

```
{
  accessList: [],
  blockHash: "0xf4e7f0a54dbc18e6777840a1fbdff8634b3e4923d09a62d7636ff923ebf280a8",
  blockNumber: 6336793,
  chainId: "0x5",
  from: "0x92ac6226ccdb0d12003884c74d42a2436ebeb928",
  gas: 21000,
  gasPrice: 1500000007,
  hash: "0xa2b547d8742e345fa5f86f017d9da38c4a19cacee91e85191a57c0c7e420d187",
  input: "0x",
  maxFeePerGas: 1500000014,
  maxPriorityFeePerGas: 1500000000,
  nonce: 0,
  r: "0x71480ea5bba6aa2f9c36568848db5afde0762a3ec7b45994139f06dbd137a6a7",
  s: "0x111979270ccfb300a7790a815b204a7913ba53ff23e469b4a2ae82b920fc565",
  to: "0x8eb19d8df81a8b43a178207e23e9a57ff8ca61b1",
  transactionIndex: 12,
  type: "0x2",
  v: "0x0",
  value: 10000000000000000
}


```

## Step 7: Using RPC

**→ Check account balance**

**Syntax:**

```javscript
    curl -X POST http://http://127.0.0.1:8545 \
        -H "Content-Type: application/json" \
       --data '{"jsonrpc":"2.0", "method":"eth_getBalance", "params":["<ADDRESS_1>","latest"], "id":1}'
```
 
 > **Note:** http://127.0.0.1:8545 this is the default address
 
 To check your account balance use the command below.

 ```javscript
  curl -X POST http://127.0.0.1:8545 \
    -H "Content-Type: application/json" \
   --data '{"jsonrpc":"2.0", "method":"eth_getBalance", "params":["0xca57f3b40b42fcce3c37b8d18adbca5260ca72ec","latest"], "id":5}'
   ```

A successful call will return a response below:

```
{"jsonrpc":"2.0","id":5,"result":"0xcc445d3d4b89390"}
```


So Geth returns the value without invoking Clef. Note that the value returned is in hexadecimal and WEI. To get the ETH value, convert to decimal and divide by 10^18.

**→ Check list of accounts**

Run the command below to get all the accounts.

```javscript
curl -X POST http://127.0.0.1:8545 \
    -H "Content-Type: application/json" \
   --data '{"jsonrpc":"2.0", "method":"eth_accounts","params":[], "id":5}'
```

Follow the same step as the IPC Check account balance

A successful call will return a response below:

```
{"jsonrpc":"2.0","id":5,"result":["0xca57f3b40b42fcce3c37b8d18adbca5260ca72ec"]}
```


**→ Send ETH to accounts**

**Syntax:**
```javscript
    curl -X POST http://http://127.0.0.1:8545 \
        -H "Content-Type: application/json" \
       --data '{"jsonrpc":"2.0", "method":"eth_sendTransaction", "params":[{"from": "<ADDRESS_1>","to": "<ADDRESS_2>","value": "0x9184e72a"}], "id":1}'
```

You need to convert eth to wei and get the hex value to send a transaction.

> **Example:**  0.0241 ether is 24100000000000000 wei, and would be encoded as the hex string "0x559ed283164000" in the JSON-RPC API.

**step 3:** Run the command below

```javscript
curl -X POST http://127.0.0.1:8545 \
    -H "Content-Type: application/json" \
   --data '{"jsonrpc":"2.0", "method":"eth_sendTransaction", "params":[{"from": "0xca57f3b40b42fcce3c37b8d18adbca5260ca72ec","to": "0x1f7a76611939fbAcf7d2dAD2F864F6184BDCD690","value": "0x2386F26FC10000"}], "id":5}'
```

A successful call will return a response below:

```
{"jsonrpc":"2.0","id":5,"result":"0xac8b347d70a82805edb85fc136fc2c4e77d31677c2f9e4e7950e0342f0dc7e7c"}
```


