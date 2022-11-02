---
title: Account Management
sort_key: C
---

Geth can use an external signer called [Clef](/docs/clef/introduction) to manage accounts. This is a standalone piece of software that runs independently of, but connects to, a Geth instance. Clef handles account creation, key management and signing transactions/data. This page explains how to use Clef to create and manage accounts for use with Geth. More information about Clef, including advanced setup options, are available in our dedicated Clef docs.

## Initialize Clef

The first time Clef is used it needs to be initialized with a master seed that unlocks Clef's secure vault and a path where the vault should be located. Clef will use the vault to store passwords for keystores, javascript auto-signing rules and hashes of rule files. To initialize Clef, pass a vault path to `clef init`, for example to store it in a new directory inside `/home/user/go-ethereum`:

```sh
clef init /home/user/go-ethereum/clefdata
```

It is extremely important to remember the master seed and keep it secure. It allows access to the accounts under Clef's management.


## Connecting Geth and Clef

The first time Clef is used it should be initialized by running `clef init`. This will prompt for a master password that is used to encrypt passwords, account data and attested rules in Clef. Once this is done, Clef is ready to use as an external signer for Geth.

Clef and Geth should be started separately but with complementary configurations so that they can communicate. This requires Clef to know the `chain_id` of the network Geth will connect to so that this information can be included in any signatures. Clef also needs to know the location of the keystore where accounts are (or will be) stored. This is usually in a subdirectory inside Geth's data directory. Clef is also given a data directory which is also often placed conveniently inside Geth's data directory. To enable communication with Clef using Curl, `--http` can be passed which will start an HTTP server on `localhost:8550` by default. To start Clef configured for a Geth node connecting to the Sepolia testnet:

```sh
clef --chainid 11155111 --keystore ~/.go-ethereum/sepolia-data/keystore --configdir ~/go-ethereum/sepolia-data/clef --http
```

Clef will now start running in the terminal, beginning with a disclaimer and a prompt to click "ok":

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

Geth can now be started in a separate terminal. To connect to Clef, ensure the data directory is consistent with the path provided to Clef and pass the location of the the Clef IPC file - which Clef saves to the path provided to its `--configdir` flag - in this case we set it to `~/go-ethereum/sepolia-data/clef`:

```sh
geth --sepolia --datadir sepolia <other flags> --signer=sepolia-data/clef/clef.ipc
```

Remember that it is also necessary to have a consensus client running too, which requires `--http` and several `authrpc` values to be provided to Geth. A complete set of startup commands for the Geth-Lodestar client combinaton plus Clef is provided as an example in this [Gist](https://gist.github.com/jmcook1186/ea5de9215ecedb1b0105bcfa9c30d44c) - adapt it for different client combinations and configurations.


## Interacting with Clef

There are two modes of interaction with Clef. One is direct interaction, which is achieved by passing requests by HTTP or IPC with JSON-RPC data as defined in Clef's external API. This is the way to do things in Clef that don't require Geth, such as creating and listing accounts, or signing data offline. The other way is via Geth. With Geth started with Clef as an external signer, requests made to Geth that touch account data will route via Clef for approval. By default, the user approves or denies interactions manually by typing `y` or `n` into the Clef console when prompted, but custom rules can also be created to automate common tasks. 

### Creating accounts

New accounts can be created using Clef's `account new` method. This generates a new key pair and adds them to the given `keystore` directory:

```sh
clef newaccount --keystore sepolia-data/keystore
```

Clef will request the new password in the terminal.

The same can be achieved using raw JSON requests (this example send the request to Clef's exposed HTTP port using curl):

```shell
curl -X POST --data '{"id": 0, "jsonrpc": "2.0", "method": "account_new", "params": []}' http://localhost:8550 -H "Content-Type: application/json"
```
The console will hang because Clef is waiting for manual approval. Switch to the Clef terminal and approve the action. Clef will prompt for a account password and then confirm the account creation in the terminal logs. A new keyfile has been added to the keystore in `go-ethereum/sepolia-data`. A JSON response is returned to the terminal the request originated from, containing the new account address in the `result` field.

```terminal
{"jsonrpc": "2.0", "id": 0, "result": "0x168bc315a2ee09042d83d7c5811b533620531f67"}
```

It is critical to backup the account password safely and securely as it cannot be retrieved or reset.

{% include note.html content=" If the password provided on account creation is lost or forgotten, there is no way to retrive it and the account will simply stay locked forever. The password MUST be backed up safely and securely! **IT IS CRITICAL TO BACKUP THE KEYSTORE AND REMEMBER PASSWORDS**" %}

The newly generated key files can be viewed in `<datadir>/keystore/`. The file naming format is `UTC--<date>--<address>` where `date` is the date and time of key creation formatted according to [UTC 8601](https://www.iso.org/iso-8601-date-and-time-format.html) with zero time offset and seconds precise to eight decimal places. `address` is the 40 hexadecimal characters that make up the account address without a leading `0x`, for example:

`UTC--2022-05-19T12-34-36.47413510Z--0b85e5a13e118466159b1e1b6a4234e5f9f784bb`


Note that there is also a Geth command for creating new accounts that will eventually be deprecated in favour of Clef. The following command will achieve the same as the RPC call suggested above:

```sh
geth account new
```

### Listing accounts

The accounts in the keystore can be listed to the terminal using `account_list` as follows:

```sh
curl -X POST --data '{"id": 0, "jsonrpc": "2.0", "method": "account_list", "params": []}' http://localhost:8550 -H "Content-Type: application/json"
```

This returns a JSON object with the account addresses in an array in the `result` field.

```terminal
{"jsonrpc": "2.0", "id": 0, "result": ["0x168bc315a2ee09042d83d7c5811b533620531f67", "0x0b85e5a13e118466159b1e1b6a4234e5f9f784bb"]}
```

The ordering of accounts when they are listed is lexicographic, but is effectively chronological based on time of creation due to the timestamp in the file name. It is safe to transfer the entire `keystore` directory or individual key files between Ethereum nodes. This is important because when accounts are added from other nodes the order of accounts in the keystore may change. It is therefore important not to rely on account indexes in scripts or code snippets.

Accounts can also be listed in the Javascript console using `eth.accounts`, which will defer to Clef for approval.


### Import a keyfile

It is also possible to create an account by importing an existing private key. For example, a user might already have some ether at an address they created using a browser wallet and now wish to use a new Geth node to interact with their funds. In this case, the private key can be exported from the browser wallet and imported into Geth. It is possible to do this using Clef, but currently the method is not externally exposed and requires implementing a UI. There is a Python UI on the Geth Github that could be used as an example or it can be done using the default console UI. However, for now the most straightforward way to import an accoutn from a private key is to use Geth's `account import`.

Geth requires the private key to be stored as a file which contains the private key as unencrypted canonical elliptic curve bytes encoded into hex (i.e. plain text key without leading 0x). The new account is then saved in encrypted format, protected by a passphrase the user provides on request. As always, this passphrase must be securely and safely backed up - there is no way to retrieve or reset it if it is forgotten!

```shell
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

```shell
geth account import --password path/password.txt path/keyfile
```

In this case, it is important to ensure the password file is not readable by anyone but the intended user. This can be achieved by changing the file permissions. On Linux, the following commands update the file permissions so only the current user has access:

```sh
chmod 700 /path/to/password
cat > /path/to/password
<type password here>
```

### Import a presale wallet

Assuming the password is known, importing a presale wallet is very easy. Geth's `wallet import` commands are used, passing the path to the wallet.

```sh
geth wallet import /path/presale.wallet
```

## Updating accounts

Clef can be used to set and remove passwords for an existing keystore file. To set a new password, pass the account address to `setpw`:

```sh
clef setpw a94f5374fce5edbc8e2a8697c15331677e6ebf0b
```

This will cause Clef to prompt for a new password, twice, and then the Clef master password to decrypt the keyfile. 

Geth's `account update` subcommand can also be used to update the account password:

```shell
geth account update a94f5374fce5edbc8e2a8697c15331677e6ebf0b
```

Alternatively, in non-interactive mode the path to a password file containing the account password in unencrypted plaintext can be passed with the `--password` flag:

```shell
geth account update a94f5374fce5edbc8e2a8697c15331677e6ebf0b --password path/password.txt
```

Updating the account using `geth account update` replaces the original file with a new one - this means the original file is no longer available after it has been updated. This can be used to update a key file to the latest format.

## Unlocking accounts

With Clef, indiscriminate account unlocking is no longer a feature. Instead, Clef unlocks are locked until actions are explicitly approved manually by a user, unless they conform to some specific scenario that has been encoded in a ruleset. Please refer to our Clef docs for instructions for how to create rulesets.


### Transactions

Transactions can be sent using raw JSON requests to Geth or using `web3js` in the Javascript console. Either way, with Clef acting as the signer the transactions will not get sent until approval is given in Clef. The following code snippet shows how a transaction could be sent between two accounts in the keystore using the Javascript console.

```shell
var tx = {from: eth.accounts[1], to: eth.accounts[2], value: web3.toWei(5, "ether")}

# this will hang until approval is given in the Clef console
eth.sendTransaction(tx)
```

## Summary

This page has demonstrated how to manage accounts using Clef and Geth's account management tools. Accounts are stored encrypted by a password. It is critical that the account passwords and the keystore directory are safely and securely backed up.
