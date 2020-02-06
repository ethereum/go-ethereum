* * *

title: Getting Started with Geth
permalink: docs/getting-started

## sort_key: A

    Sync to a test net, probably goerli
    create account
    get eth for that account
    send transaction to another account

* * *

To use Geth, you need to install it first. You can install Geth in a variety
of ways that you can find in the "[Install and Build](install-and-build/installing-geth)" section. These include installing it via your favorite package manager, downloading a
standalone pre-built binary, running it as a docker container, or building it yourself.

For this guide, we assume you have Geth installed and are ready to find  out how to use it.

## Sync your local node to a test network

To begin, start your node by connecting it to a network and setting a sync mode.

```shell
geth --goerli --syncmode "light"
```

### Networks

You can connect a Geth node to several different networks using its name as an argument. These include the main Ethereum network, [a private network](getting-started/private-net) you  create, and three test networks that use different consensus algorithms:

-   **Ropsten**: Proof-of-work test network
-   **Rinkeby**: Proof-of-authority test network
-   **Görli**: Proof-of-authority test network

For this guide we use the Görli network.

### Sync modes

You can start Geth in one of three different sync modes using the `--syncmode "{mode}"`
argument that determines what sort of node it is in the network.

These are:

-   **Full**: Downloads all blocks (including headers, transactions and receipts) and
    generates the state of the blockchain incrementally by executing every block.
-   **Fast** (Default): Downloads all blocks (including headers, transactions and
    receipts), verifies all headers, and downloads the state and verifies it against the
    headers.
-   **Light**: Downloads all block headers, block data, and verifies some randomly.

## Connect to Geth

You can interact with Geth in two ways: Directly with the node using the JavaScript console over IPC, or connecting to the node remotely over HTTP using RPC.

RPC allows remote applications to  access your node, but has limitations and security considerations.

### Using the console

IPC  allows you to do more, especially when it comes to creatig, and interactinng with accounts, but you need direct  access to the node.

You can open the console in  two ways, first when you start the node:

```shell
geth --goerli --syncmode "light" console
```

However the node also  outputs sync information, so using the console can be difficult.

You  can also open a console on the node from another terminal using:

```shell
geth attach {IPC_LOCATION}
```

<!-- TODO: Add note about geth attach ~/Library/Ethereum/goerli/geth.ipc -->

### Using RPC

You can use standard HTTP requests to connect to a Geth node using the RPC APIs, for examples:

```shell
curl -X POST http://{IP_ADDRESS}:8545 --data \
    '{"jsonrpc":"2.0",
    "method":"{METHOD}",
    "params":[],
    "id":1}' \ // TODO: What's this?
    -H "Content-Type:application/json"
```

## Create an account

Next create an account that represents a key pair. 

<!-- TODO: Maybe more on the above -->

Use the following command in the JavaScript console to create a new account and set a password for that account:

```javascript
personal.newAccount()
```

<!-- TODO: And for RPC? -->

_[Read this guide](./interface/managing-your-accounts) for more details on importing
existing Ethereum accounts and other uses of the `account` command._

## Send Eth to accounts

Unless you have Ether in another account on the Görli network, you can use  a [faucet](https://goerli-faucet.slock.it/) to send ETH to your new account(s).

<!-- TODO: On the below, why? -->

Before you can  interact with an account that was secured with a  password, you need to unlock it. With the console, use the following command:

```javascript
personal.unlockAccount("{ACCOUNT_ADDRESS}", "{PASSWORD}", {SECONDS})
```

<!-- TODO: And with RPC? -->

Now you can send ETH to the account and check  its balance with:

```javascript
web3.fromWei(eth.getBalance("{ADDRESS}"),"ether")
```

## Send ETH between accounts

Now  you have ETH in an account, let's send it to another account. First  create a new account using the same method as above.

With two accounts, transfer 0.01 ETH from to  the new  account:

<!-- TODO: Placeholders -->

```javascript
eth.sendTransaction({from:eth.accounts[0],to:"0x21b2228d9522b57f8c1e60508ef97ff7106868f3", value: web3.toWei(0.01,"ether")})
```

* * *

### Javascript Console

Once you have an account and Geth is running, you can interact with it by opening another
terminal and using the following command to open a JavaScript console:

```shell
geth attach
```

If you get the error 'unable to attach to remote geth', try connecting via HTTP as shown below:

```shell
geth attach http://127.0.0.1:8545
```

In the console you can issue any of the Geth commands, for example, to list all the
accounts on the node, use:

```js
> eth.accounts
```

You can also enter the console directly when you start the node with the `console` command:

```shell
geth console --syncmode "light"
```

* * *

## Console

geth --goerli --syncmode "light"
geth attach ~/Library/Ethereum/goerli/geth.ipc
personal.newAccount()

Use faucet two two addresses

web3.fromWei(eth.getBalance("0xa5b4ee82d326de3321c926030e5abce8ab610bfb"),"ether")

personal.unlockAccount("0x21b2228d9522b57f8c1e60508ef97ff7106868f3", "password", 300)

eth.sendTransaction({from:eth.accounts[0],to:"0x21b2228d9522b57f8c1e60508ef97ff7106868f3", value: web3.toWei(0.01,"ether")})

## RPCs

Create new accounnt is possible, but not recommended, so stick to IPC for these steps.  then how to unlock?

curl -X POST <http://localhost:8545> --data '{"jsonrpc":"2.0","method":"eth_getBalance","params":["0xa5b4ee82d326de3321c926030e5abce8ab610bfb","latest"],"id":1}' -H "Content-Type:application/json"

{"jsonrpc":"2.0","id":1,"result":"0x8e08b04d79b000"}

curl -X POST <http://localhost:8545> --data '{"jsonrpc":"2.0","method":"eth_sendTransaction","params":[{"from": "0xa5b4ee82d326de3321c926030e5abce8ab610bfb","to": "0x21b2228d9522b57f8c1e60508ef97ff7106868f3","value": "0x9184e72a"}],"id":1}' -H "Content-Type:application/json"
