---
title: Dev mode
sort_key: B
---

It is often convenient for developers to work in an environment where changes to client or application software can be deployed and tested rapidly and without putting real-world users or assets at risk. For this purpose, Geth has a `--dev` flag that spins up Geth in "developer mode". This creates a single-node Ethereum test network with no connections to any external peers. It exists solely on the local machine. Starting Geth in developer mode does the following:

-   Initializes the data directory with a testing genesis block
-   Sets max peers to 0 (meaning Geth does not search for peers)
-   Turns off discovery by other nodes (meaning the node is invisible to other nodes)
-   Sets the gas price to 0 (no cost to send transactions)
-   Uses the Clique proof-of-authority consensus engine with which allows blocks to be mined as-needed without excessive CPU and memory consumption
-   Uses on-demand block generation, producing blocks when transactions are waiting to be mined

This configuration enables developers to experiment with Geth's source code or develop new applications without having to sync to a pre-existing public network. Blocks are only mined when there are pending transactions. Developers can break things on this network without affecting other users. This page will demonstrate how to spin up a local Geth testnet and a simple smart contract will be deployed to it using the Remix online integrated development environment (IDE).

## Prerequisites

It is assumed that the user has a working Geth installation (see [installation guide](../_install-and-build/)).
It would also be helpful to have basic knowledge of Geth and the Geth console. See [Getting Started](../_getting-started/).
Some basic knowledge of [Solidity](https://docs.soliditylang.org/) and [smart contract deployment](https://ethereum.org/en/developers/tutorials/deploying-your-first-smart-contract/) would be useful.

## Start Geth in Dev Mode

Starting Geth in developer mode is as simple as providing the `--dev` flag. It is also possible to create a realistic block creation frequency by setting `--dev.period 13` instead of creating blocks only when transactions are pending. There are also additional configuration options required to follow this tutorial. First, a data directory will be created so that the local blockchain state can be maintained between sessions - without this the local blockchain is ephemeral and only existing in memory for the duration the node is running. In the command below this is achieved by providing the directory name, `dev-chain` as an argument following the `--datadir` flag. Next, `http` must be enabled so that the Javascript console can be attached to the Geth node, and some namespaces must be specified so that certain functions can be executed from the Javascript console, specifically `eth`, `web3` and `personal`. Finally, Remix will be used to deploy a smart contract to the node which requires information to be exchanged externally to Geth's own domain. To permit this, the `net` namespace must be enabled and the Remix URL must be provided to `--http.corsdomain`. The full command is as follows:

```shell

geth --datadir dev-chain --dev --http --http.api eth,web3,personal,net --http.corsdomain "http://remix.ethereum.org"

```

The terminal will display the following logs, confirming Geth has started successfully in developer mode:

```terminal


ADD GETH STARTUP LOGS!!!


```

This terminal must be left running throughout the entire tutorial. In a second terminal, attach a Javascript console:


```shell

geth attach http://127.0.0.1:8545

```

The Javascript terminal will open with the following welcome message:

```terminal

Welcome to the Geth Javascript console!

instance: Geth/v1.10.18-unstable-8d84a701-20220503/linux-amd64/go.1.18.1
coinbase: 0x540dbaeb2390f2eb005f7a6dbf3436a0959197a9
at block: 0 (Thu Jan 01 1970 01:00:00 GMT+0100 (BST))
 modules: eth:1.0 personal:1.0 rpc:1.0 web3:1.0

To exit, press ctrl-d or type exit
>

```

In the [Getting Started](../_getting-started/) tutorial it was explained that using the external signing and account management tool, Clef, was best practise for generating and securing user accounts. However, Clef cannot be used on a local development network. Therefore, this tutorial will use Geth's built-in account management. First, the existing accounts can be displayed using `eth.accounts`:

```shell

eth.accounts

```

An array containing a single address will be displayed in the terminal, despite no accounts having yet been explicitly created. This is the "coinbase" account. The coinbase address is the recipient of the total amount of ether created at the local network genesis. Querying the ether balance of the coinbase account will return a very large number. The coinbase account can be invoked as `eth.accounts[0]` or as `eth.coinbase`:

```terminal

> eth.coinbase==eth.accounts[0]
true

```

The following command can be used to query the balance. The return value is in units of Wei, which is divided by 1e18 to give units of ether. This can be done explicitly or by calling the `web3.FromWei` function:

```shell

eth.getBalance(eth.coinbase)/1e18

// or

web3.fromWei(eth.getBalance(eth.coinbase))

```

Using `web3.fromWei` is less error prone because the correct multiplier is built in. These commands both return the following:

```terminal

1.157920892373162e+59

```

Now, a new account can be created and some of the ether from the coinbase transferred across to it. A new account is generated using the `newAccount` function in the `personal` namespace:

```shell

personal.newAccount()

```

The terminal will display a request for a password, twice. Once provided, a new account will be created and its address printed to the terminal. The account creation is also logged in the Geth terminal, including the location of the keyfile in the keystore. It is a good idea to back up the password somewhere at this point. If this were an account on a live network, intended to own assets of real-world value, it would be critical to back up the account password and the keyfiles in a secure manner.

To reconfirm the account creation, running `eth.accounts` in the Javascript console should now display an array containing two account addresses, one being the coinbae and othe other being the newly generated address. The following command transfers 50 ETH from the coinbase to the new account:

```shell

eth.sendTransaction({from: eth.coinbase, to: eth.accounts[1], value: web3.toWei(50, "ether")})

```

A transaction hash will be returned to the console. This transaction hash will also be displayed in the logs in the Geth console, followed by logs confirming that a new block was mined (remember in the local development network blocks are mined when transactions are pending). The transaction details can be displayed in the Javascript console by passing the transaction hash to `eth.getTransaction()`:

```shell

eth.getTransaction("0x62044d2cab405388891ee6d53747817f34c0f00341cde548c0ce9834e9718f27")

```

The transaction details are displayed as follows:

```terminal

TODO: ADD TRANSACTION HASH DETAILS


```

Now that the user account is funded with ether, a contract can be created ready to deploy to the Geth node.

## A simple smart contract

This tutorial will make use of a classic example smart contract, `Storage.sol`. This contract exposes two public functions, one to add a value to the contract storage and one to view the stored value. The contract, written in Solidity, is provided below:


```Solidity

pragma solidity >=0.7.0;

contract Storage{


    uint256 number;

    function store(uint256 num) public{

        number = num;
    }

    function retrieve() public view returns (uint256){
        return number;
    
    }
}

```

Solidity is a high-level language that makes code executable by the Ethereum virtual machine (EVM) readable to humans. This means that there is an intermediate step between writing code in Solidity and deploying it to Ethereum. This step is called "compilation" and it converts human-readable code into EVM-executable byte-code. This byte-code is then included in a transaction sent from the Geth node during contract deployment. This can all be done directly from the Geth Javascript console; however this tutorial uses an online IDE called Remix to handle the compilation and deployment of the contract to the local Geth node.



## Compile and deploy using Remix

In a web browser, open <https://remix.ethereum.org>. This opens an online smart contract development environment. On the left-hand side of the screen there is a side-bar menu that toggles between several toolboxes that are displayed in a vertical panel. On the right hand side of the screen there is an editor and a terminal. This layout is similar to the default layout of many other IDEs such as [VSCode](https://code.visualstudio.com/). The contract defined in the previous section, `Storage.sol` is already available in the `Contracts` directory in Remix. It can be opened and reviewed in the editor.

The Solidity logo is present as an icon in the Remix side-bar. Clicking this icon opens the Solidity compiler wizard. This can be used to compile `Storage.sol` ready. With `Solidity.sol` open in the editor window, simply click the `Compile 1_Storage.sol` button. A green tick will appear next to the Solidity icon to confirm that the contract has compiled successfully. This means the contract bytecode is available.

Below the Solidit icon is a fourth icon that includes the Ethereum logo. Clicking this opens the Deploy menu. In this menu, Remix can be configured to connect to the local Geth node. In the drop-down menu labelled `ENVIRONMENT `, select `Injected Web3`. This will open an information pop-up with instructions for configuring Geth - these cna be ignored as they were completed earlier in this tutorial. However, at the bottom of this pop-up is a box labelled `Web3 Provider Endpoint`. This should be set to Geth's 8545 port on `localhost` (`127.0.0.1:8545`). Click OK. The `ACCOUNT` field should automatically populate with the address of the account created earlier using the Geth Javascript console.

To deploy `Storage.sol`, click `DEPLOY`.

The following logs in the Geth terminal confirm that the contract was successfully deployed.


```terminal

TODO: ADD DEPLOY LOGS HERE

```

## Interact with contract using Remix

The contract is now deployed on a local testnet version of the Etheruem blockchain. This means there is a contract address that contains executable bytecode that can be invoked by sending transactions with instructions, also in bytecode, to that address. Again, this can all be achieved by constructing transactions directly in the Geth console or even by making external http requests using tools such as Curl. However, this tutorial will use Remix in order to abstract away some complexity. 

After deploying the contract in Remix, the `Deployed Contracts` tab in the sidebar automatically populates with the public functions exposed by `Storage.sol`. To send a value to the contract storage, type a number in the field adjacent to the `store` button, then click the button. In the Geth terminal, the following logs confirm that the transaction was successful (the actual values will vary from the example below):

```terminal

ADD CONTRACT FUNCTION EXECUTION LOGS HERE

```

The transaction hash can be used to retrieve the transaction details using the Geth Javascript console, which will return the following information:

```terminal

TODO: ADD CONTRACT TRANSACTION LOG HERE!!

```

The `from` address is the account that sent the transaction, the `to` address is the deployment address of the contract. The value entered into Remix is now in storage at that contract address. This can be retrieved using Remix by calling the `retrieve` function - to do this simpyl click the `retrieve` button. Alternatively, it can be retrieved using `web3.getStorageAt` using the Geth Javascript console. The following command returns the value in the contract storage (replace the given address with the correct one displayed in the Geth logs).

```shell

web3.eth.getStorageAt("0x407d73d8a49eeb85d32cf465507dd71d507100c1", 0)

```

This returns a value that looks like the following:

```terminal

"0x000000000000000000000000000000000000000000000000000000000000000038"

```


The returned value is a left-padded hexadecimal value. For example, the return value `0x000000000000000000000000000000000000000000000000000000000000000038` corresponds to a value of `56` entered as a uint256 to Remix. After convertinfrom hexadecimal string to decimal number the returned value should be equal to that provided to Remix in the previosu step.

## Summary

This tutorial has demonstrated how to spin up a local developer network usign Geth. Having started this development network, a simple contract was deployed to the developer network. Then, Remix was connected to the local Geth node and used to deploy and interact with a contract. Remix was used to add a value to the contract storage and then the value was retrieved using Remix and also using the lower level commands in the Javascript console.