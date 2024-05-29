---
title: Developer mode
description: Instructions for setting up Geth in developer mode
---

It is often convenient for developers to work in an environment where changes to client or application software can be deployed and tested rapidly and without putting real-world users or assets at risk. For this purpose, Geth has a `--dev` flag that spins up Geth in "developer mode". This creates a single-node Ethereum test network with no connections to any external peers. It exists solely on the local machine. Starting Geth in developer mode does the following:

- Initializes the data directory with a testing genesis block
- Sets max peers to 0 (meaning Geth does not search for peers)
- Turns off discovery by other nodes (meaning the node is invisible to other nodes)
- Sets the gas price to 0 (no cost to send transactions)
- Simulates a consensus client which allows blocks to be mined as-needed without excessive CPU and memory consumption
- Uses on-demand block generation, producing blocks when transactions are waiting to be mined

This configuration enables developers to experiment with Geth's source code or develop new applications without having to sync to a pre-existing public network. Blocks are only mined when there are pending transactions. Developers can break things on this network without affecting other users. This page will demonstrate how to spin up a local Geth testnet and a simple smart contract will be deployed to it using the Remix online integrated development environment (IDE).

## Prerequisites {#prerequisites}

It is assumed that the user has a working Geth installation (see [installation guide](/docs/getting-started/installing-geth)).
It would also be helpful to have basic knowledge of Geth and the Geth console. See [Getting Started](/docs/getting-started).
Some basic knowledge of [Solidity](https://docs.soliditylang.org/) and [smart contract deployment](https://ethereum.org/en/developers/tutorials/deploying-your-first-smart-contract/) would be useful.

## Start Geth in Dev Mode {#start-geth-in-dev-mode}

Starting Geth in developer mode is as simple as providing the `--dev` flag. It is also possible to create a realistic block creation frequency by setting `--dev.period 12` instead of creating blocks only when transactions are pending. Additional configuration options required to follow this tutorial.

Remix will be used to deploy a smart contract to the node which requires information to be exchanged externally with Geth's own domain. To permit this, enable `http` and the `net` namespace must be enabled and the Remix URL must be provided to `--http.corsdomain`. Some other namespaces will also be enabled for this tutorial. The full command is as follows:

```sh
geth --dev --http --http.api eth,web3,net --http.corsdomain "https://remix.ethereum.org"
```

The terminal will display the following logs, confirming Geth has started successfully in developer mode:

```terminal
INFO [05-09|10:49:02.951] Starting Geth in ephemeral dev mode...
INFO [05-09|10:49:02.952] Maximum peer count                       ETH=50 LES=0 total=50
INFO [05-09|10:49:02.952] Smartcard socket not found, disabling    err="stat /run/pcscd/pcscd.comm: no such file or directory"
INFO [05-09|10:49:02.953] Set global gas cap                       cap=50,000,000
INFO [05-09|10:49:03.133] Using developer account                  address=0x7Aa16266Ba3d309e3cb278B452b1A6307E52Fb62
INFO [05-09|10:49:03.196] Allocated trie memory caches             clean=154.00MiB dirty=256.00MiB
INFO [05-09|10:49:03.285] Writing custom genesis block
INFO [05-09|10:49:03.286] Persisted trie from memory database      nodes=13 size=1.90KiB time="180.524µs" gcnodes=0 gcsize=0.00B gctime=0s livenodes=1 livesize=0.00B
INFO [05-09|10:49:03.287] Initialised chain configuration          config="{ ChainID: 1337 Homestead: 0 DAO: nil DAOSupport: false EIP150: 0 EIP155: 0 EIP158: 0 Byzantium: 0 Constantinople: 0 Petersburg: 0 Istanbul: 0, Muir Glacier: 0, Berlin: 0, London: 0, Arrow Glacier: nil, MergeFork: nil, Terminal TD: nil, Engine: clique}"
INFO [05-09|10:49:03.288] Initialising Ethereum protocol           network=1337 dbversion= nil
INFO [05-09|10:49:03.289] Loaded most recent local header          number=0 hash=c9c3de..579bb8 td=1 age=53y1mo1w
INFO [05-09|10:49:03.289] Loaded most recent local full block      number=0 hash=c9c3de..579bb8 td=1 age=53y1mo1w
INFO [05-09|10:49:03.289] Loaded most recent local fast block      number=0 hash=c9c3de..579bb8 td=1 age=53y1mo1w
WARN [05-09|10:49:03.289] Failed to load snapshot, regenerating    err="missing or corrupted snapshot"
INFO [05-09|10:49:03.289] Rebuilding state snapshot
INFO [05-09|10:49:03.290] Resuming state snapshot generation       root=ceb850..0662cb accounts=0 slots=0 storage=0.00B elapsed="778.089µs"
INFO [05-09|10:49:03.290] Regenerated local transaction journal    transactions=0 accounts=0
INFO [05-09|10:49:03.292] Gasprice oracle is ignoring threshold set threshold=2
INFO [05-09|10:49:03.292] Generated state snapshot                 accounts=10 slots=0 storage=412.00B elapsed=2.418ms
WARN [05-09|10:49:03.292] Error reading unclean shutdown markers   error="leveldb: not found"
INFO [05-09|10:49:03.292] Starting peer-to-peer node               instance=Geth/v1.10.18-unstable-8d84a701-20220503/linux-amd64/go1.18.1
WARN [05-09|10:49:03.292] P2P server will be useless, neither dialing nor listening
INFO [05-09|10:49:03.292] Stored checkpoint snapshot to disk       number=0 hash=c9c3de..579bb8
INFO [05-09|10:49:03.312] New local node record                    seq=1,652,089,743,311 id=bfedca74bea20733 ip=127.0.0.1 udp=0 tcp=0
INFO [05-09|10:49:03.313] Started P2P networking                   self=enode://0544de6446dd5831daa5a391de8d0375d93ac602a95d6a182d499de31f22f75b6645c3f562932cac8328d51321b676c683471e2cf7b3c338bb6930faf6ead389@127.0.0.1:0
INFO [05-09|10:49:03.314] IPC endpoint opened                      url=/tmp/geth.ipc
INFO [05-09|10:49:03.315] HTTP server started                      endpoint=127.0.0.1:8545 auth=false prefix= cors=http:remix.ethereum.org vhosts=localhost
INFO [05-09|10:49:03.315] Transaction pool price threshold updated price=0
INFO [05-09|10:49:03.315] Updated mining threads                   threads=0
INFO [05-09|10:49:03.315] Transaction pool price threshold updated price=1
INFO [05-09|10:49:03.315] Etherbase automatically configured       address=0x7Aa16266Ba3d309e3cb278B452b1A6307E52Fb62
INFO [05-09|10:49:03.316] Commit new sealing work                  number=1 sealhash=2372a2..7fb8e7 uncles=0 txs=0 gas=0 fees=0 elapsed="202.366µs"
WARN [05-09|10:49:03.316] Block sealing failed                     err="sealing paused while waiting for transactions"
INFO [05-09|10:49:03.316] Commit new sealing work                  number=1 sealhash=2372a2..7fb8e7 uncles=0 txs=0 gas=0 fees=0 elapsed="540.054µs"
```

This terminal must be left running throughout the entire tutorial. In a second terminal, attach a Javascript console. By default, the `ipc` file is saved in the `datadir`:

```sh
geth attach <datadir>/geth.ipc
```

The Javascript terminal will open with the following welcome message:

```terminal
Welcome to the Geth JavaScript console!

instance: Geth/v1.14.3-stable-ab48ba42/linux-arm64/go1.22.3
at block: 0 (Thu Jan 01 1970 00:00:00 GMT+0000 (UTC))
 datadir: 
 modules: admin:1.0 debug:1.0 dev:1.0 eth:1.0 miner:1.0 net:1.0 rpc:1.0 txpool:1.0 web3:1.0

To exit, press ctrl-d or type exit
> 
```

For simplicity this tutorial uses Geth's built-in account management. First, the existing accounts can be displayed using `eth.accounts`:

```sh
eth.accounts
```

An array containing a single address will be displayed in the terminal, despite no accounts having yet been explicitly created. This is the "developer" account. The developer address is the recipient of the total amount of ether created at the local network genesis. Querying the ether balance of the developer account will return a very large number. The developer account can be invoked as `eth.accounts[0]`.

The following command can be used to query the balance. The return value is in units of Wei, which is divided by 1<sup>18</sup> to give units of ether. This can be done explicitly or by calling the `web3.FromWei()` function:

```sh
eth.getBalance(eth.accounts[0])/1e18

// or

web3.fromWei(eth.getBalance(eth.accounts[0]))
```

Using `web3.fromWei()` is less error-prone because the correct multiplier is built in. These commands both return the following:

```terminal
1.157920892373162e+59
```

A new account can be created using Clef. Some of the ether from the developer account can then be transferred across to it. A new account is generated using the `newaccount` function on the command line:

```sh
clef newaccount --keystore <path-to-keystore>
```

The terminal will display a request for a password, twice. Once provided, a new account will be created, and its address will be printed to the terminal. The account creation is also logged in the Geth terminal, including the location of the keyfile in the keystore. It is a good idea to back up the password somewhere at this point. If this were an account on a live network, intended to own assets of real-world value, it would be critical to back up the account password and the keystore in a secure manner.

To reconfirm the account creation, running `eth.accounts` in the Javascript console should display an array containing two account addresses, one being the developer account and the other being the newly generated address. The following command transfers 50 ETH from the developer account to the new account:

```sh
eth.sendTransaction({from: eth.accounts[0], to: eth.accounts[1], value: web3.toWei(50, "ether")})
```

A transaction hash will be returned to the console. This transaction hash will also be displayed in the logs in the Geth console, followed by logs confirming that a new block was mined (remember, in the local development network, blocks are mined when transactions are pending). The transaction details can be displayed in the Javascript console by passing the transaction hash to `eth.getTransaction()`:

```sh
eth.getTransaction("0x62044d2cab405388891ee6d53747817f34c0f00341cde548c0ce9834e9718f27")
```

The transaction details are displayed as follows:

```terminal
{
  accessList: [],
  blockHash: "0xdef68762539ebfb247e31d749acc26ab5df3163aabf9d450b6001c200d17da8a",
  blockNumber: 1,
  chainId: "0x539",
  from: "0x540dbaeb2390f2eb005f7a6dbf3436a0959197a9",
  gas: 21000,
  gasPrice: 875000001,
  hash: "0x2326887390dc04483d435a6303dc05bd2648086eab15f24d7dcdf8c26e8af4b8",
  input: "0x",
  maxFeePerGas: 2000000001,
  maxPriorityFeePerGas: 1,
  nonce: 0,
  r: "0x3f7b54f095b837ec13480eab5ac7de082465fc79f43b813cca051394dd028d5d",
  s: "0x167ef271ae8175239dccdd970db85e06a044d5039252d6232d0954d803bb4e3e",
  to: "0x43e3a14fb8c68caa7eea95a02693759de4513950",
  transactionIndex: 0,
  type: "0x2",
  v: "0x0",
  value: 50000000000000000000,
  yParity: "0x0"
}
```

Now that the user account is funded with ether, a contract can be created and deployed to the Geth node.

## A simple smart contract {#simple-smart-contract}

This tutorial will make use of a classic example smart contract, `Storage.sol`. This contract exposes two public functions, one to add a value to the contract storage and one to view the stored value. The contract, written in Solidity, is provided below:

```solidity
pragma solidity >=0.8.2 <0.9.0;

contract Storage {
    uint256 number;

    function store(uint256 num) public {
        number = num;
    }

    function retrieve() public view returns (uint256) {
        return number;
    }
}
```

Solidity is a high-level language that makes code executable by the Ethereum virtual machine (EVM) readable to humans. This means that there is an intermediate step between writing code in Solidity and deploying it to Ethereum. This step is called "compilation", and it converts human-readable code into EVM-executable bytecode. This byte-code is then included in a transaction sent from the Geth node during contract deployment. This can all be done directly from the Geth Javascript console; however, this tutorial uses an online IDE called Remix to handle the compilation and deployment of the contract to the local Geth node.

## Compile and deploy using Remix {#compile-and-deploy}

In a web browser, open <https://remix.ethereum.org>. This opens an online smart contract development environment. On the left-hand side of the screen there is a side-bar menu that toggles between several toolboxes that are displayed in a vertical panel. On the right-hand side of the screen, there is an editor and a terminal. This layout is similar to the default layout of many other IDEs such as [VSCode](https://code.visualstudio.com/). The contract defined in the previous section, `Storage.sol` is already available in the `Contracts` directory in Remix. It can be opened and reviewed in the editor.

![Remix](/images/docs/remix.png)

The Solidity logo is present as an icon in the Remix sidebar. Clicking this icon opens the Solidity compiler wizard, which can be used to compile `Storage.sol` ready. With `Solidity.sol` open in the editor window, simply click the `Compile 1_Storage.sol` button. A green tick will appear next to the Solidity icon to confirm that the contract has been compiled successfully. This means the contract bytecode is available.

![Remix-compiler](/images/docs/remix-compiler.png)

Below the Solidity icon is a fourth icon that includes the Ethereum logo. Clicking this opens the Deploy menu. In this menu, Remix can be configured to connect to the local Geth node. In the drop-down menu labelled `ENVIRONMENT`, select `Custom - External Http Provider`. This will open an information pop-up with instructions for configuring Geth - these can be ignored as they were completed earlier in this tutorial. However, at the bottom of this pop-up is a box labelled `External HTTP Provider Endpoint`. This should be set to Geth's 8545 port on `localhost` (`http://127.0.0.1:8545`). Click OK. The `ACCOUNT` field should automatically populate with the address of the account created earlier using the Geth Javascript console.

![Remix-deploy](/images/docs/remix-deploy.png)

To deploy `Storage.sol`, click `DEPLOY`.

The following logs in the Geth terminal confirm that the contract was successfully deployed.

```terminal
INFO [05-09|12:27:09.680] Setting new local account                address=0x7Aa16266Ba3d309e3cb278B452b1A6307E52Fb62
INFO [05-09|12:27:09.680] Submitted contract creation              hash=0xbf2d2d1c393a882ffb6c90e6d1713906fd799651ae683237223b897d4781c4f2 from=0x7Aa16266Ba3d309e3cb278B452b1A6307E52Fb62 nonce=1 contract=0x4aA11DdfD817dD70e9FF2A2bf9c0306e8EC450d3 value=0
INFO [05-09|12:27:09.681] Commit new sealing work                  number=2 sealhash=845a53..f22818 uncles=0 txs=1 gas=125,677 fees=0.0003141925 elapsed="335.991µs"
INFO [05-09|12:27:09.681] Successfully sealed new block            number=2 sealhash=845a53..f22818 hash=e927bc..f2c8ed elapsed="703.415µs"
INFO [05-09|12:27:09.681] 🔨 mined potential block                  number=2 hash=e927bc..f2c8ed
```

## Interact with contract using Remix {#interact-with-contract}

The contract is now deployed on a local testnet version of the Ethereum blockchain. This means there is a contract address that contains an executable bytecode that can be invoked by sending transactions with instructions, also in bytecode, to that address. Again, this can all be achieved by constructing transactions directly in the Geth console or even by making external http requests using tools such as Curl. Here, Remix is used to retrieve the value; then the same action is taken using the Javascript console.

After deploying the contract in Remix, the `Deployed Contracts` tab in the sidebar automatically populates with the public functions exposed by `Storage.sol`. To send a value to the contract storage, type a number in the field adjacent to the `store` button, then click the button.

![Remix-func](/images/docs/remix-func.png)

In the Geth terminal, the following logs confirm that the transaction was successful (the actual values will vary from the example below):

```terminal
INFO [05-09|13:41:58.644] Submitted transaction                    hash=0xfa3cd8df6841c5d3706d3bacfb881d2b985d0b55bdba440f1fdafa4ed5b5cc31 from=0x7Aa16266Ba3d309e3cb278B452b1A6307E52Fb62 nonce=2 recipient=0x4aA11DdfD817dD70e9FF2A2bf9c0306e8EC450d3 value=0
INFO [05-09|13:41:58.644] Commit new sealing work                  number=3 sealhash=5442e3..f49739 uncles=0 txs=1 gas=43724   fees=0.00010931   elapsed="334.446µs"
INFO [05-09|13:41:58.645] Successfully sealed new block            number=3 sealhash=5442e3..f49739 hash=c076c8..eeee77 elapsed="581.374µs"
INFO [05-09|13:41:58.645] 🔨 mined potential block                  number=3 hash=c076c8..eeee77
```

The transaction hash can be used to retrieve the transaction details using the Geth Javascript console, which will return the following information:

```terminal
{
  accessList: [],
  blockHash: "0xc076c88200618f4cbbfb4fe7c3eb8d93566724755acc6c4e9a355cc090eeee77",
  blockNumber: 3,
  chainId: "0x539",
  from: "0x7aa16266ba3d309e3cb278b452b1a6307e52fb62",
  gas: 43724,
  gasPrice: 3172359839,
  hash: "0xfa3cd8df6841c5d3706d3bacfb881d2b985d0b55bdba440f1fdafa4ed5b5cc31",
  input: "0x6057361d0000000000000000000000000000000000000000000000000000000000000038",
  maxFeePerGas: 4032048134,
  maxPriorityFeePerGas: 2500000000,
  nonce: 2,
  r: "0x859b88062715c5d66b9a188886ad51b68a1e4938d5932ce2dac874c104d2b26",
  s: "0x61ef6bc454d5e6a76c414f133aeb6321197a61e263a3e270a16bd4a65d94da55",
  to: "0x4aa11ddfd817dd70e9ff2a2bf9c0306e8ec450d3",
  transactionIndex: 0,
  type: "0x2",
  v: "0x1",
  value: 0,
  yParity: "0x1"
}
```

The `from` address is the account that sent the transaction, and the `to` address is the deployment address of the contract. The value entered into Remix is now in storage at that contract address. This can be retrieved using Remix by calling the `retrieve` function - to do this simply click the `retrieve` button. Alternatively, it can be retrieved using `web3.getStorageAt` using the Geth Javascript console. The following command returns the value in the contract storage (replace the given address with the correct one displayed in the Geth logs).

```sh
web3.eth.getStorageAt("0x407d73d8a49eeb85d32cf465507dd71d507100c1", 0)
```

This returns a value that looks like the following:

```terminal
"0x000000000000000000000000000000000000000000000000000000000000000038"
```

The returned value is a left-padded hexadecimal value. For example, the return value `0x000000000000000000000000000000000000000000000000000000000000000038` corresponds to a value of `56` entered as a uint256 to Remix. After converting from hexadecimal string to decimal number the returned value should be equal to that provided to Remix in the previous step.

## Reusing --datadir {#reusing-datadir}

This tutorial used an ephemeral blockchain that is completely destroyed and started afresh during each dev-mode session. However, it is also possible to create persistent blockchain and account data that can be reused across multiple sessions. This is done by providing the `--datadir` flag and a directory name when starting Geth in dev-mode.

```sh
geth --datadir dev-chain --dev --http --http.api web3,eth,net --http.corsdomain "https://remix.ethereum.org"
```

## Re-using accounts {#reusing-accounts}

Geth will fail to start in dev-mode if keys have been manually created or imported into the keystore in the `--datadir` directory. This is because the account cannot be automatically unlocked. To resolve this issue, the password defined when the account was created can be saved to a text file and its path passed to the `--password` flag on starting Geth, for example, if `password.txt` is saved in the top-level `go-ethereum` directory:

```sh
geth --datadir dev-chain --dev --http --http.api web3,eth,net --http.corsdomain "https://remix.ethereum.org" --password password.txt
```

## Using a Custom Genesis Configuration

It is possible to use a custom genesis block configuration in development mode. To obtain a compatible configuration, run `geth --dev dumpgenesis`. The resulting genesis has proof-of-stake and all pre-merge hard forks activated at block 0. Precompile addresses are funded to prevent them being removed from the state per EIP158.

Users are free to modify the generated template provided they keep the pre-merge hard-forks and proof-of-stake transition activated at block 0.

## Summary {#summary}

This tutorial has demonstrated how to spin up a local developer network using Geth. Having started this development network, a simple contract was deployed to the developer network. Then, Remix was connected to the local Geth node and used to deploy and interact with a contract. Remix was used to add a value to the contract storage, and then the value was retrieved using Remix and also using the lower-level commands in the Javascript console.
