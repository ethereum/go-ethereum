---
title: Puppeth
description: introduction to the private-network boot-strapping tool, Puppeth
---

Puppeth is a tool for quickly spinning up and managing private development networks. Puppeth gives fine-grained control over the network properties including the genesis block, signers, bootnodes, dashboards, etc but abstracts away the complexity of configuring it all manually.
The user is guided through the process by a command line wizard.

Puppeth comes bundled with Geth. The binary for Puppeth is built along with the other command line tools when the user runs `make all`. By default the binaries are saved to `build/bin`. This page demonstrates how to start a private proof-of-authority network with all the nodes running on the local machine. Other configurations are also possible, for example nodes can be spread over multiple (virtual) machines and the consensus mechanism can be proof-of-work.

## Creating accounts {#creating-accounts}

To run a Clique network, authorized nodes must seal each block. This requires accounts to exist that can be pre-authorized in the genesis block. These accounts should be created before Puppeth is started. The accounts can be created using Geth's built in account manager as follows:

```sh
geth account new --datadir NodeId
```

For each account, replace NodeId with Node 1, 2, 3 etc. This saves the details about each account to a new directory.

Geth will prompt for a password. Once provided, the public address of the new account and the location of the secret key file is displayed to the terminal. It is a good idea to copy these details down in a text document because they will be needed later. This account generation step should be repeated until the number of accounts is at least equal to the desired number of nodes on the proof-of-authority network plus a few extra that will act as non-sealing nodes. Make sure the account passwords are also securely backed up for each new account.

See more on the [account management pages](/docs/fundamentals/account-management).

## Starting Puppeth {#starting-puppeth}

Starting Puppeth is as easy as running `puppeth` in the terminal:

```sh
puppeth
```

This starts the wizard in the terminal.

```terminal
+-----------------------------------------------------------+
| Welcome to puppeth, your Ethereum private network manager |
|                                                           |
| This tool lets you create a new Ethereum network down to  |
| the genesis block, bootnodes, miners and ethstats servers |
| without the hassle that it would normally entail.         |
|                                                           |
| Puppeth uses SSH to dial in to remote servers, and builds |
| its network components out of Docker containers using the |
| docker-compose toolset.                                   |
+-----------------------------------------------------------+

Please specify a network name to administer (no spaces or hyphens, please)
```

The wizard prompts for a network name, in this case it can be called `testnetwork`. Typing `testnetwork` and Enter returns the following:

```terminal
Sweet, you can set this via --network=testnetwork next time!

INFO [08-15|12:40:39.643] Administering Ethereum network      name=testnetwork
WARN [08-15|12:40:39.643] No previous configurations found    path=/home/.puppeth/testnetwork

What would you like to do? (default = stats)
1. Show network stats
2. Configure new genesis block
3. Track new remote server
4. Deploy network components
```

There are four options displayed in the terminal. Select `2. Configure new genesis block` by typing `2` and Enter. The wizard presents the option to start a new genesis block from scratch or to import one. Select `1. Create new genesis block from scratch`. Then choose `Clique - proof-of-authority` when the wizard prompts for a choice of consensus engine, and then, when prompted, a block time in seconds (e.g. 15).

Next, the wizard prompts for the addresses of accounts that should be authorized to sign blocks. Here, the public addresses of the accoutjs created earlier can be pasted one-by-one. Note that the leading `0x` is preset, so do not repeat it when copy/pasting. Enter each address individually, separating them by pressing Enter (i.e. do not enter a list of addresses).

In a real network these would not be arbitrary accounts as they are in this demonstration, they would be trusted accounts belonging to specific node operators authorized to seal blocks.

After determining the sealer accounts, the wizard asks which accounts to prefund with ether. Provide all the sealer account addresses and the additional addresses generated earlier.

```terminal
Which accounts are allowed to seal? (mandatory at least one)

> 0xbb70c0073cb20d3b20cec14f2bfbe1b61a5b2bd1
> 0xbef818cf91f521012020ff1ec17c5e5e929b2bc6
...

Which accounts should be pre-funded? (advisable at least one)

> 0xbb70c0073cb20d3b20cec14f2bfbe1b61a5b2bd1
> 0xbef818cf91f521012020ff1ec17c5e5e929b2bc6
```

The final prompt is for a network ID. For this tutorial it is fine to skip this by hitting Enter 0 - this causes Puppeth to fallback to its default behaviour which is to randomly generate a network ID.

Puppeth will then display the following message to the terminal indicating that `testnetwork`'s genesis block has been configured.

```terminal
INFO [08-15|14:25:09.630] Configured new genesis block
```

Puppeth has also returned to the 'start menu' encountered earlier. Now, the second option on the menu has updated toread `2. manage existing genesis`. Selecting that option opens a new menu where the genesis configuration can be modified, removed or exported. Choose `2` again to export the config data as a set of json files to a user-defined directory.

```terminal
What would you like to do? (default = stats)
1. Shown network status
2. Manage existing genesis
3. Track new remote server
4. Deploy network components

> 2

1. Modify existing configurations
2. Export genesis configurations
3. Remove genesis configurations

> 2

Which folder to save the genesis specs into? (default = current)
  Will create testnetwork.json, testnetwork-aleth.json, testnetwork-harmony.json, testnetwork-parity.json

> /home/testnetwork
```

At this point a genesis configuration has been created and backed up. There are a few more componments that are required to start the network.

## Network components {#network-components}

Puppeth includes wizards for adding several network components:

```sh
1. Ethstats - Network monitoring tool
2. Bootnode - Entry point for a network
3. Sealer - Full node minting new blocks
4. Explorer - Chain analysis webservice
5. Faucet - Crypto faucet to give away funds
6. Dashboard - Website listing above web services
```

These are all accessed by starting Puppeth and selecting `4. Deploy network components` from the main menu. They should be deployed in the numerical order in which they are listed in the `Network components` submenu.

### Ethstats {#ethstats}

Ethstats is a network monitoring service. The Ethstats server must already be installed (see [instructions](https://github.com/cubedro/eth-netstats)) and running so that its IP address can be provided to Puppeth. The IP address of each node is also required. The wizard guides the user through providing the IP addresses and ports for Ethstats and the local nodes and setting a password for the Ethstats API.

### Bootnodes {#bootnodes}

Bootnodes are nodes with hardcoded addresses that allow new nodes entering the network to immediately find peers to connect to. This makes peer discovery faster. The wizard guides the user through providing IP addresses for nodes on the network that will be used as bootnodes.

### Sealer {#sealer}

The sealer nodes must be specified. These validate the network by sealing blocks. The wizard prompts the user to provide the IP addresses for the sealer nodes along with their keyfiles and unlock passwords. Some additional information is also set for the bootnodes including their gas limit - the higher the gas limit the more work the node has to do to validate each block. To match Ethereum set it to 15,000,000. The gas price can be anything, but since it is a private test network it may as well be small, say 1 GWei.

Puppeth will display the details of each node in a table in the terminal.

### Explorer {#explorer}

For proof-of-work networks a block explorer akin to [etherscan](https://etherscan.io/) can be created using the Puppeth wizard.

### Faucet {#faucet}

A faucet is an app that allows accounts to request ether to be sent to them. This can be created easily by following the wizard. The wizard prompts the user for details related to which node will act as a server for the faucet, how much ether to release per request, intervals between releases and some optional security features.

### Dashboard {#dashboard}

The dashboard wizard pulls together the pieces from the already-defined network components into a single dashboard that can be navigated to in a web browser. The wizard guides the user through the necessary steps. Optionally, the explorer and faucet apps can be deployed here too.

The dashboard can then be viewed by navigating to the node's ip address and the defined port in a web browser.

## Starting the network {#starting-network}

Start instances of Geth for each node

```sh
geth --datadir Node1 --port 30301 --bootnodes <enr> --networkid <testnetwork ID> -unlock <node 1 address> --mine
```

## Summary {#summary}

Puppeth is a command line wizard that guides a user through the various stages of setting up a private network using proof-of-authority or proof-of-work consensus engine. Various network components can be added that optimize the network or enable network monitoring.

[GitHub repository](https://github.com/ethereum/go-ethereum/tree/master/cmd/puppeth)
