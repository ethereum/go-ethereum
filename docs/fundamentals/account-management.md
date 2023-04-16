---
title: Account Management with Clef
description: Guide to basic account management using Geth's built-in tools
---

Geth uses an external signer called [Clef](/docs/tools/clef/introduction) to manage accounts. This is a standalone piece of software that runs independently of - but connects to - a Geth instance. Clef handles account creation, key management and signing transactions/data. This page explains how to use Clef to create and manage accounts for use with Geth. More information about Clef, including advanced setup options, are available in our dedicated Clef docs.

## Initialize Clef {#initializing-clef}

The first time Clef is used it needs to be initialized with a master seed that unlocks Clef's secure vault and a path where the vault should be located. Clef will use the vault to store passwords for keystores, javascript auto-signing rules and hashes of rule files. To initialize Clef, pass a vault path to `clef init`, for example to store it in a new directory inside `/home/user/go-ethereum`:

```sh
clef init /home/user/go-ethereum/clefdata
```

It is extremely important to remember the master seed and keep it secure. It allows access to the accounts under Clef's management.

## Connecting Geth and Clef {#connecting-geth-and-clef}

Clef and Geth should be started separately but with complementary configurations so that they can communicate. This requires Clef to know the `chain_id` of the network Geth will connect to so that this information can be included in any signatures. Clef also needs to know the location of the keystore where accounts are (or will be) stored. This is usually in a subdirectory inside Geth's data directory. Clef is also given a data directory which is also often placed conveniently inside Geth's data directory. To enable communication with Clef using Curl, `--http` can be passed which will start an HTTP server on `localhost:8550` by default. To start Clef configured for a Geth node connecting to the Sepolia testnet:

```sh
clef --chainid 11155111 --keystore ~/.go-ethereum/sepolia-data/keystore --configdir ~/go-ethereum/sepolia-data/clef --http
```

Clef will start running in the terminal, beginning with a disclaimer and a prompt to click "ok":

```terminal
WARNING!

Clef is an account management tool. It may, like any software, contain bugs.

Please take care to
- backup your keystore files,
- verify that the keystore(s) can be opened with your password.

Clef is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY
without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR
PURPOSE. See the GNU General Public License for more details.

Enter 'ok' to proceed:
>
```

Geth can be started in a separate terminal. To connect to Clef, ensure the data directory is consistent with the path provided to Clef and pass the location of the the Clef IPC file - which Clef saves to the path provided to its `--configdir` flag - in this case we set it to `~/go-ethereum/sepolia-data/clef`:

```sh
geth --sepolia --datadir sepolia <other flags> --signer=sepolia-data/clef/clef.ipc
```

Remember that it is also necessary to have a consensus client running too, which requires `--http` and several `authrpc` values to be provided to Geth. A complete set of startup commands for the Geth-Lodestar client combinaton plus Clef is provided as an example in this [Gist](https://gist.github.com/jmcook1186/ea5de9215ecedb1b0105bcfa9c30d44c) - adapt it for different client combinations and configurations.

## Interacting with Clef {#interacting-with-clef}

There are two modes of interaction with Clef. One is direct interaction, which is achieved by passing requests by HTTP or IPC with JSON-RPC data as defined in Clef's external API. This is the way to do things in Clef that don't require Geth, such as creating and listing accounts, or signing data offline. The other way is via Geth. With Geth started with Clef as an external signer, requests made to Geth that touch account data will route via Clef for approval. By default, the user approves or denies interactions manually by typing `y` or `n` into the Clef console when prompted, but custom rules can also be created to automate common tasks.

### Creating accounts {#creating-accounts}

New accounts can be created using Clef's `account new` method. This generates a new key pair and adds them to the given `keystore` directory:

```sh
clef newaccount --keystore sepolia-data/keystore
```

Clef will request the new password in the terminal.

The same can be achieved using raw JSON requests (this example send the request to Clef's exposed HTTP port using curl):

```sh
curl -X POST --data '{"id": 0, "jsonrpc": "2.0", "method": "account_new", "params": []}' http://localhost:8550 -H "Content-Type: application/json"
```

The console will hang because Clef is waiting for manual approval. Switch to the Clef terminal and approve the action. Clef will prompt for an account password and then confirm the account creation in the terminal logs. A new keyfile has been added to the keystore in `go-ethereum/sepolia-data`. A JSON response is returned to the terminal the request originated from, containing the new account address in the `result` field.

```terminal
{"jsonrpc": "2.0", "id": 0, "result": "0x168bc315a2ee09042d83d7c5811b533620531f67"}
```

It is critical to backup the account password safely and securely as it cannot be retrieved or reset.

<Note>If the password provided on account creation is lost or forgotten, there is no way to retrive it and the account will simply stay locked forever. The password MUST be backed up safely and securely! **IT IS CRITICAL TO BACKUP THE KEYSTORE AND REMEMBER PASSWORDS!**</Note>

The newly generated key files can be viewed in `<datadir>/keystore/`. The file naming format is `UTC--<date>--<address>` where `date` is the date and time of key creation formatted according to [UTC 8601](https://www.iso.org/iso-8601-date-and-time-format.html) with zero time offset and seconds precise to eight decimal places. `address` is the 40 hexadecimal characters that make up the account address without a leading `0x`, for example:

`UTC--2022-05-19T12-34-36.47413510Z--0b85e5a13e118466159b1e1b6a4234e5f9f784bb`

An account can also be created by importing a raw private key (hex string) using `clef importraw` as follows:

```sh
clef importraw <hexkey>
```

The terminal will respond with the following message, indicating the account has been created successfully:

```terminal
## Info
Key imported:
  Address 0x9160DC9105f7De5dC5E7f3d97ef11DA47269BdA6
  Keystore file: /home/user/.ethereum/keystore/UTC--2022-10-28T12-03-13.976383602Z--9160dc9105f7de5dc5e7f3d97ef11da47269bda6

The key is now encrypted; losing the password will result in permanently losing
access to the key and all associated funds!

Make sure to backup keystore and passwords in a safe location.
```

### Listing accounts {#listing-accounts}

The accounts in the keystore can be listed to the terminal using a simple CLI command as follows:

```sh
clef list-accounts --keystore <path-to-keystore>
```

or using `account_list` in a POST request as follows:

```sh
curl -X POST --data '{"id": 0, "jsonrpc": "2.0", "method": "account_list", "params": []}' http://localhost:8550 -H "Content-Type: application/json"
```

This returns a JSON object with the account addresses in an array in the `result` field.

```terminal
{"jsonrpc": "2.0", "id": 0, "result": ["0x168bc315a2ee09042d83d7c5811b533620531f67", "0x0b85e5a13e118466159b1e1b6a4234e5f9f784bb"]}
```

The ordering of accounts when they are listed is lexicographic, but is effectively chronological based on time of creation due to the timestamp in the file name. It is safe to transfer the entire `keystore` directory or individual key files between Ethereum nodes. This is important because when accounts are added from other nodes the order of accounts in the keystore may change. It is therefore important not to rely on account indexes in scripts or code snippets.

Accounts can also be listed in the Javascript console using `eth.accounts`, which will defer to Clef for approval.

As well as individual accounts, any wallets managed by Clef can be listed (which will also print the wallet status and the address and URl of any accounts they contain. This uses the `list-wallets` CLI command.

```sh
clef list-wallets --keystore <path-to-keystore>
```

which returns:

```terminal
- Wallet 0 at keystore:///home/user/Code/go-ethereum/testdata/keystore/UTC--2022-11-01T17-05-01.517877299Z--4f4094babd1a8c433e0f52a6ee3b6ff32dee6a9c (Locked )
  - Account 0: 0x4f4094BaBd1A8c433e0f52A6ee3B6ff32dEe6a9c (keystore:///home/user/go-ethereum/testdata/keystore/UTC--2022-11-01T17-05-01.517877299Z--4f4094babd1a8c433e0f52a6ee3b6ff32dee6a9c)
- Wallet 1 at keystore:///home/user/go-ethereum/testdata/keystore/UTC--2022-11-01T17-05-11.100536003Z--8ef15919f852a8034688a71d8b57ab0187364009 (Locked )
  - Account 0: 0x8Ef15919F852A8034688a71d8b57Ab0187364009 (keystore:///home/user/go-ethereum/testdata/keystore/UTC--2022-11-01T17-05-11.100536003Z--8ef15919f852a8034688a71d8b57ab0187364009)
```

### Import a keyfile {#importing-a-keyfile}

It is also possible to create an account by importing an existing private key. For example, a user might already have some ether at an address they created using a browser wallet and now wish to use a new Geth node to interact with their funds. In this case, the private key can be exported from the browser wallet and imported into Geth. It is possible to do this using Clef, but currently the method is not externally exposed and requires implementing a UI. There is a Python UI on the Geth GitHub that could be used as an example or it can be done using the default console UI. However, for now, the most straightforward way to import an account from a private key is to use Geth's `account import`.

Geth requires the private key to be stored as a file which contains the private key as unencrypted canonical elliptic curve bytes encoded into hex (i.e. plain text key without leading 0x). The new account is then saved in encrypted format, protected by a passphrase the user provides on request. As always, this passphrase must be securely and safely backed up - there is no way to retrieve or reset it if it is forgotten!

```sh
$ geth account import --datadir /some-dir ./keyfile
```

The following information will be displayed in the terminal, indicating a successful import:

```terminal
Please enter a passphrase now.
Passphrase:
Repeat Passphrase:
Address: {7f444580bfef4b9bc7e14eb7fb2a029336b07c9d}
```

This import/export process is **not necessary** for users transferring accounts between Geth instances because the key files can simply be copied directly from one keystore to another.

It is also possible to import an account in non-interactive mode by saving the account password as plaintext in a `.txt` file and passing its path with the `--password` flag on startup.

```sh
geth account import --password path/password.txt path/keyfile
```

In this case, it is important to ensure the password file is not readable by anyone but the intended user. This can be achieved by changing the file permissions. On Linux, the following commands update the file permissions so only the current user has access:

```sh
chmod 700 /path/to/password
cat > /path/to/password
<type password here>
```

### Import a presale wallet {#import-presale-wallet}

Assuming the password is known, importing a presale wallet is very easy. Geth's `wallet import` commands are used, passing the path to the wallet.

```sh
geth wallet import /path/presale.wallet
```

## Updating accounts {#updating-accounts}

Clef can be used to set and remove passwords for an existing keystore file. To set a new password, pass the account address to `setpw`:

```sh
clef setpw a94f5374fce5edbc8e2a8697c15331677e6ebf0b
```

This will cause Clef to prompt for a new password, twice, and then the Clef master password to decrypt the keyfile.

Geth's `account update` subcommand can also be used to update the account password:

```sh
geth account update a94f5374fce5edbc8e2a8697c15331677e6ebf0b
```

Alternatively, in non-interactive mode the path to a password file containing the account password in unencrypted plaintext can be passed with the `--password` flag:

```sh
geth account update a94f5374fce5edbc8e2a8697c15331677e6ebf0b --password path/password.txt
```

Updating the account using `geth account update` replaces the original file with a new one - this means the original file is no longer available after it has been updated. This can be used to update a key file to the latest format.

## Unlocking accounts {#unlocking-accounts}

With Clef, indiscriminate account unlocking is no longer a feature. Instead, Clef unlocks are locked until actions are explicitly approved manually by a user, unless they conform to some specific scenario that has been encoded in a ruleset. Please refer to our Clef docs for instructions for how to create rulesets.

### Transactions {#transactions}

Transactions can be sent using raw JSON requests to Geth or using `web3js` in the Javascript console. Either way, with Clef acting as the signer the transactions will not get sent until approval is given in Clef. The following code snippet shows how a transaction could be sent between two accounts in the keystore using the Javascript console.

```sh
var tx = {from: eth.accounts[1], to: eth.accounts[2], value: web3.toWei(5, "ether")}

# this will hang until approval is given in the Clef console
eth.sendTransaction(tx)
```

## Summary {#summary}

This page has demonstrated how to manage accounts using Clef and Geth's account management tools. Accounts are stored encrypted by a password. It is critical that the account passwords and the keystore directory are safely and securely backed up.
