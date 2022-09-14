---
title: Account Management
description: Guide to basic account management using Geth's built-in tools
---

The recommended practise for managing accounts in Geth is to use Clef. However, Geth also has its own, convenient account management tools. Eventually, these built in tools will be deprecated in favour of using Clef as the default account manager. This page describes account management using Geth's built-in tools. It is recommended to also visit the following pages that explain how to use Clef.

- [Getting started with Clef](/content/docs/getting_started/getting-started-with-clef.md)
- [Introduction to Clef](/content/docs/tools/Clef/Introduction.md)
- [Clef tutorial](/content/docs/tools/Clef/Tutorial.md)

## Account command

Geth's `account` command is used to interact with accounts:

```
geth account <command> [options...] [arguments...]
```

The account command enables the user to create new accounts, list existing accounts, import private keys into a new account, update key formats and update the passwords that lock each account. In interactive mode, the user is prompted for passwords in the console when the `account` functions are invoked, whereas in non-interactive mode passwords to unlock accounts are saved to text files whose path is passed to Geth at startup. Non-interactive mode is only intended for use on private networks or known safe environments.

The `account` subcommands are:

```
COMMANDS:
     list    Print summary of existing accounts
     new     Create a new account
     update  Update an existing account
     import  Import a private key into a new account
```

Information about the subcommands can be displayed in the terminal using `geth account <command> --help`. For example, for the `list` subcommand:

```
$ geth account list --help
list [command options] [arguments...]

Print a short summary of all accounts

OPTIONS:
  --datadir "/home/.ethereum"  Data directory for the databases and keystore
  --keystore                   Directory for the keystore (default = inside the datadir)
```

## Creating new accounts

New accounts can be created using `account new`. This generates a new key pair and adds them to the `keystore` directory in the `datadir`. To
create a new account in the default data directory:

```shell
$ geth account new
```

This returns the following to the terminal:

```terminal
Your new account is locked with a password. Please give a password. Do not forget this password.
Passphrase:
Repeat Passphrase:
Address: {168bc315a2ee09042d83d7c5811b533620531f67}
```

It is critical to backup the account password safely and securely as it cannot be retrieved or reset.

{% include note.html content=" If the password provided on account creation is lost or forgotten, there is no way to retrive it and the account will simply stay locked forever. The password MUST be backed up safely and securely!
**IT IS CRITICAL TO BACKUP THE KEYSTORE AND REMEMBER PASSWORDS**" %}

The newly generated key files can be viewed in `<datadir>/keystore/`. The file naming format is `UTC--<date>--<address>` where `date` is the date and time of key creation formatted according to [UTC 8601](https://www.iso.org/iso-8601-date-and-time-format.html) with zero time offset and seconds precise to eight decimal places. `address` is the 40 hexadecimal characters that make up the account address without a leading `0x`, for example:

`UTC--2022-05-19T12-34-36.47413510Z--0b85e5a13e118466159b1e1b6a4234e5f9f784bb`

## Listing Accounts

Listing all existing accounts is achieved using the `account list` command. If the keystore is located anywhere other than the default location its path should be included with the `keystore` flag. For example, if the datadir is `some-dir`:

```shell
geth account list --keystore some-dir/keystore
```

This command returns the following to the terminal for a keystore with two files:

```terminal
Account 0: {5afdd78bdacb56ab1dad28741ea2a0e47fe41331} keystore:///tmp/mykeystore/UTC--2017-04-28T08-46-27.437847599Z--5afdd78bdacb56ab1dad28741ea2a0e47fe41331
Account 1: {9acb9ff906641a434803efb474c96a837756287f} keystore:///tmp/mykeystore/UTC--2017-04-28T08-46-52.180688336Z--9acb9ff906641a434803efb474c96a837756287f
```

The ordering of accounts when they are listed is lexicographic, but is effectively chronological based on time of creation due to the timestamp in the file name. It is safe to transfer the entire `keystore` directory or individual key files between Ethereum nodes. This is important because when accounts are added from other nodes the order of accounts in the keystore may change. It is therefore important not to rely on account indexes in scripts or code snippets.

## Importing accounts

### Import a keyfile

It is also possible to create a new account by importing a private key. For example, a user might already have some ether at an address they created using a browser wallet and now wish to use a new Geth node to interact with their funds. In this case, the private key can be exported from the browser wallet and imported into Geth. Geth requires the private key to be stored as a file which contains the private key as unencrypted
canonical elliptic curve bytes encoded into hex (i.e. plain text key without leading 0x). The new account is then saved in encrypted format, protected by a passphrase the user provides on request. As always, this passphrase must be securely and safely backed up - there is no way to retrieve or reset it if it is forgotten!

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

This import/export process is not necessary for transferring accounts between Geth instances because the key files can simply be copied directly from one keystore to another.

It is also possible to import an account in non-interactive mode by saving the account password as plaintext in a `.txt` file and passing its path with the `--password` flag on startup.

```shell
geth account import --password path/password.txt path/keyfile
```

In this case, it is important to ensure the password file is not readable by anyone but the intended user. This can be achieved by changing the file permissions. On Linux, the following commands update the file permissions so only the current user has access:

```shell
chmod 700 /path/to/password
cat > /path/to/password
<type password here>
```

### Import a presale wallet

Assuming the password is known, importing a presale wallet is very easy. The `wallet import` commands are used, passing the path to the wallet.

```shell
geth wallet import /path/presale.wallet
```

## Updating accounts

The `account update` subcommand is used to unlock an account and migrate it to the newest format. This is useful for accounts that may have been created in a format that has since been deprecated. The same command can be used to update the account password. The current password and account address are needed in order to update the account, as follows:

```shell
geth account update a94f5374fce5edbc8e2a8697c15331677e6ebf0b
```

The following will be returned to the terminal:

```terminal
Unlocking account a94f5374fce5edbc8e2a8697c15331677e6ebf0b | Attempt 1/3
Passphrase:
0xa94f5374fce5edbc8e2a8697c15331677e6ebf0b
Account 'a94f5374fce5edbc8e2a8697c15331677e6ebf0b' unlocked.
Please give a new password. Do not forget this password.
Passphrase:
Repeat Passphrase:
0xa94f5374fce5edbc8e2a8697c15331677e6ebf0b
```

Alternatively, in non-interactive mode the path to a password file containing the account password in unencrypted plaintext can be passed with the `--password` flag:

```shell
geth account update a94f5374fce5edbc8e2a8697c15331677e6ebf0b --password path/password.txt
```

Updating the account replaces the original file with a new one - this means the original file is no longer available after it has been updated.

## Unlocking accounts

In Geth, accounts are locked unless they are explicitly unlocked. If an account is intended to be used by apps connecting to Geth via RPC then it can be unlocked in non-interactive mode by passing the `--unlock` flag with a comma-separated list of account addresses (or keystore indexes) to unlock. This unlocks the accounts for one session only. Including the `--unlock` flag without any account addresses defaults to unlocking the first account
in the keystore.

```shell
geth <other commands> --unlock 0xa94f5374fce5edbc8e2a8697c15331677e6ebf0b
```

Geth will start and prompt the user to input the account password in the terminal. Alternatively, the user can provide a password as a text file and pass its path to `--password`:

```shell
geth <other commands> --unlock 0xa94f5374fce5edbc8e2a8697c15331677e6ebf0b --password path/password.txt
```

{% include note.html content=" By default, account **unlocking is forbidden when HTTP or Websocket access is enabled** (i.e. by passing `--http` or `ws` flag). This is because an attacker that manages to access the node via the externally-exposed HTTP/WS port can then control the unlocked account.
It is possible to force account unlock by including the `--allow-insecure-unlock` flag but this is unsafe and **not recommended** except for expert users that completely understand how it can be used safely. This is not a hypothetical risk: **there are bots that continually scan for http-enabled Ethereum nodes to attack**" %}

## Accounts in the Javascript console

Account management can also be achieved in the Javascript console attached to a running Geth instance. Assuming Geth is already running, in a new terminal attach a Javascript console using the `geth.ipc` file. This file can be found in the data directory. Assuming the data directory is named `data` the console can be started using:

```shell
geth attach data/geth.ipc
```

### New accounts

New accounts can be generated using the Javascript console using `personal.newAccount()`. A new password is requested in the console and successful account creation is confirmed by the new account address being displayed.

```shell
personal.newAccount()
```

Accounts can also be created by importing private keys directly in the Javascript console. The private key is passed as an unencrypted hex-encoded string to `personal.importRawKey()` along with a passphrase that will be used to encrypt the key. A new key file will be generated from the private key and saved to the keystore.

```shell
personal.importRawKey("hexstringkey", "password")
```

### Listing accounts

The `accounts` function in the `eth` namespace can be used to list the accounts that currently exist in the keystore.:

```
eth.accounts
```

or alternatively the same is achieved using:

```
personal.listAccounts
```

This returns an array of account addresses to the terminal.

### Unlocking accounts

To unlock an account, the `personal.unlockAccount` function can be used:

```
personal.unlockAccount(eth.accounts[1])
```

The account passphrase is requested:

```terminal
Unlock account 0xa94f5374fce5edbc8e2a8697c15331677e6ebf0b
Passphrase:
true
```

This unlocked account can now be used to sign and send transactions. it is also possible to pass the passphrase as an argument to `personal.unlockAccount()` along with a duration after which the accout will automatically re-lock (in seconds), as follows:

```shell
personal.unlockAccount(eth.accounts[1], "passphrase", 60)
```

This unlocks the account for 60 seconds. However, this is not recommended because the command history is logged by the Javascript console which could compromise the security of the account. An unlocked account can be manually re-locked using `personal.lockAccount()`, passing the address as the sole argument.

### Unlocking for transactions

Sending transactions from the Javascript console also requires the sender account to be unlocked. There are two ways to send transactions: `eth.sendTransaction` and `personal.sendTransaction`. The difference between these two functions is that `eth.sendTransaction` requires the account to be
unlocked globally, at the node level (i.e., by unlocking it on the command line at the start of the Geth session). On the other hand, `personal.sendTransaction` takes a passphrase argument that unlocks the account temporarily in order to sign the transaction, then locks it again
immediately afterwards. For example, to send 5 ether between two accounts in the keystore:

```shell
var tx = {from: eth.accounts[1], to: eth.accounts[2], value: web3.toWei(5, "ether")}

# this requires global account unlock for eth.accounts[1]
eth.sendTransaction(tx)

# this unlocks eth.accounts[1] temporarily just to send the transaction
personal.sendTransaction(tx, "password")
```

## Summary

This page has demonstrated how to use Geth's built-in account management tools, both on the command line and in the Javascript console. Accounts are stored encrypted by a password. It is critical that the account passwords and the keystore directory are safely and securely backed up.
