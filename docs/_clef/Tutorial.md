---
title: Tutorial
sort_key: A
---

This page provides a step-by-step walkthrough tutorial demonstrating some common uses of Clef. This 
includes manual approvals and automated rules. Clef is presented both as a standalone general signer
with requests made via RPC and also as a backend signer for Geth.

{:toc}
-   this will be removed by the toc


## Initializing Clef

First things first, Clef needs to store some data itself. Since that data might be sensitive 
(passwords, signing rules, accounts), Clef's entire storage is encrypted. To support encrypting data, 
the first step is to initialize Clef with a random master seed, itself too encrypted with your chosen 
password:

```text
$ clef init

WARNING!

Clef is an account management tool. It may, like any software, contain bugs.

Please take care to
- backup your keystore files,
- verify that the keystore(s) can be opened with your password.

Clef is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY;
without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR
PURPOSE. See the GNU General Public License for more details.

Enter 'ok' to proceed:
> ok

The master seed of clef will be locked with a password.
Please specify a password. Do not forget this password!
Password:
Repeat password:

A master seed has been generated into /home/martin/.clef/masterseed.json

This is required to be able to store credentials, such as:
* Passwords for keystores (used by rule engine)
* Storage for JavaScript auto-signing rules
* Hash of JavaScript rule-file

You should treat 'masterseed.json' with utmost secrecy and make a backup of it!
* The password is necessary but not enough, you need to back up the master seed too!
* The master seed does not contain your accounts, those need to be backed up separately!
```

*For readability purposes, we'll remove the WARNING printout, user confirmation and the unlocking of the master seed in the rest of this document.*

## Remote interactions

This tutorial will use Clef with Geth on the Goerli testnet. The accounts used will be in the 
Goerli keystore with the path `~/go-ethereum/goerli-data/keystore`. The tutorial assumes there 
are two accounts in this keystore. Instructions for creating accounts can be found on the 
[Account managament page](/docs/interface/managing-your-accounts). Note that Clef can also interact 
with hardware wallets, although that is not demonstrated here.

Clef should be started before Geth, otherwise Geth will complain that it cannot find a Clef 
instance to connect to. Clef should be started with the correct `chainid` for Goerli. Clef 
itself does not connect to a blockchain, but the `chainID` parameter is included in the data 
that is aggregated to form a signature. Clef also needs a path to the correct keystore passed to 
the `--keystore` command. A custom path to the config directory can also be provided. This is where the
`ipc` file will be saved which is needed to connect Clef to Geth:

```sh
clef --keystore ~/go-ethereum/goerli-data/keystore --configdir ~/go-ethereum/goerli-data/clef --chainid=5
```

The following logs will be displayed in the console:

```terminal
INFO [07-01|11:00:46.385] Starting signer                          chainid=4 keystore= go-ethereum/goerli-data/keystore light-kdf=false advanced=false
DEBUG[07-01|11:00:46.389] FS scan times                            list=3.521941ms set=9.017µs diff=4.112µs
DEBUG[07-01|11:00:46.391] Ledger support enabled
DEBUG[07-01|11:00:46.391] Trezor support enabled via HID
DEBUG[07-01|11:00:46.391] Trezor support enabled via WebUSB
INFO [07-01|11:00:46.391] Audit logs configured                    file=audit.log
DEBUG[07-01|11:00:46.392] IPC registered                           namespace=account
INFO [07-01|11:00:46.392] IPC endpoint opened                      url=go-ethereum/goerli-data/clef/clef.ipc
------- Signer info -------
* intapi_version : 7.0.1
* extapi_version : 6.1.0
* extapi_http : n/a
* extapi_ipc : go-ethereum/goerli-data/clef/clef.ipc
```

Clef starts up in CLI (Command Line Interface) mode by default. Arbitrary remote 
processes may *request* account interactions (e.g. sign a transaction), which the user 
can individually *confirm* or *deny*.

The code snippet below shows a request made to Clef via its *External API endpoint* using 
[NetCat](http://netcat.sourceforge.net/). The request invokes the 
["account_list"](/docs/_clef/apis#accountlist) endpoint which lists the accounts in the keystore. 
This command should be run in a new terminal.

```sh
echo '{"id": 1, "jsonrpc": "2.0", "method": "account_list"}' | nc -U ~/.clef/clef.ipc
```

The terminal used to send the command will now hang. This is because the process is awaiting 
confirmation from Clef. Switching to the Clef console reveals Clef's prompt to the user to 
confirm or deny the request:

```terminal
-------- List Account request--------------
A request has been made to list all accounts.
You can select which accounts the caller can see
  [x] 0xD9C9Cd5f6779558b6e0eD4e6Acf6b1947E7fA1F3
    URL: keystore://go-ethereum/goerli-data/keystore/UTC--2017-04-14T15-15-00.327614556Z--d9c9cd5f6779558b6e0ed4e6acf6b1947e7fa1f3
  [x] 0x086278A6C067775F71d6B2BB1856Db6E28c30418
    URL: keystore://go-ethereum/goerli-data/keystore/UTC--2018-02-06T22-53-11.211657239Z--086278a6c067775f71d6b2bb1856db6e28c30418
-------------------------------------------
Request context:
	NA - ipc - NA

Additional HTTP header data, provided by the external caller:
	User-Agent:
	Origin:
Approve? [y/N]:
```

Depending on whether the request is approved or denied, the NetCat process in the other terminal 
will receive one of the following responses:

```terminal
{"jsonrpc":"2.0","id":1,"result":["0xd9c9cd5f6779558b6e0ed4e6acf6b1947e7fa1f3","0x086278a6c067775f71d6b2bb1856db6e28c30418"]}
```

or

```terminal
{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"Request denied"}}
```

Apart from listing accounts, you can also *request* creating a new account, signing transactions 
and data or recovering signatures. The available methods are documented in the Clef 
[External API Spec](https://github.com/ethereum/go-ethereum/tree/master/cmd/clef#external-api-1) 
and the [External API Changelog](https://github.com/ethereum/go-ethereum/blob/master/cmd/clef/extapi_changelog.md).

*Note, the number of things you can do from the External API is deliberately small to limit 
the power of remote calls as much as possible! Clef has an 
[Internal API](https://github.com/ethereum/go-ethereum/tree/master/cmd/clef#ui-api-1) 
too for the UI (User Interface) which is much richer and can support custom interfaces on top. 
But that's out of scope here.*

The example above used Clef completely independently of Geth. However, by defining Clef as 
the signer when Geth is started imposes Clef's `request - confirm - result` pattern to any 
interaction with the local Geth node that touches accounts, including requests made using 
RPC or an attached Javascript console. To demonstrate this, Geth can be started,
with Clef as the signer:

```sh
geth --goerli --datadir goerli-data --signer=goerli-data/clef/clef.ipc
```

With Geth running, open a new terminal and attach a Javascript console:

```sh
geth attach goerli-data/geth.ipc
```

A simple request to list the accounts in the keystore will cause the Javascript console to hang.

```js
eth.accounts
```

Switching to the Clef terminal reveals that this is because the request is awaiting explicit 
confirmation from the user. The log is identical to the one shown above, when the same request 
for account information was made to Clef via Netcat:

```terminal
-------- List Account request--------------
A request has been made to list all accounts.
You can select which accounts the caller can see
  [x] 0xD9C9Cd5f6779558b6e0eD4e6Acf6b1947E7fA1F3
    URL: keystore://go-ethereum/goerli-data/keystore/UTC--2017-04-14T15-15-00.327614556Z--d9c9cd5f6779558b6e0ed4e6acf6b1947e7fa1f3
  [x] 0x086278A6C067775F71d6B2BB1856Db6E28c30418
    URL: keystore://go-ethereum/goerli-data/keystore/UTC--2018-02-06T22-53-11.211657239Z--086278a6c067775f71d6b2bb1856db6e28c30418
-------------------------------------------
Request context:
	NA - ipc - NA

Additional HTTP header data, provided by the external caller:
	User-Agent:
	Origin:
Approve? [y/N]:
```

In this mode, the user is required to manually confirm every action that touches account data, 
including querying accounts, signing and sending transactions.

The example below shows an ether transaction between the two accounts in the keystore 
using `eth.sendTransaction` in the attached Javascript console.

```js
// this command requires 2x approval in Clef because it loads account data via eth.accounts[0]
// and eth.accounts[1]
var tx = {from: eth.accounts[0], to: eth.accounts[1], value: web3.toWei(0.1, "ether")}

// then send the transaction
eth.sendTransaction(tx)
```

This example demonstrates the power of Clef much more clearly than the account-listing example. 
In the Clef terminal, all the details of the transaction are presented to the user so that they 
can be reviewed before being confirmed. This gives the user an opportunity to review the fine 
details and make absolutely sure they really want to sign the transaction. `eth.sendTransaction`
returns the following confirmation prompt in the Clef terminal:


```terminal
-------- Transaction request----------------
to:     0x086278A6C067775F71d6B2BB1856Db6E28c30418
from:               0xD9C9Cd5f6779558b6e0eD4e6Acf6b1947E7fA1F3 [chksum ok]
value:              100000000000000000 wei
gas:                0x5208 (21000)
maxFeePerGas:           1500000016 wei
maxPriorityFeePerGas:   1500000000 wei
nonce:  0x0 (0)
chainid: 0x5
Accesslist

Request context:
        NA - ipc - NA

Additional HTTP header data, provided by the external caller:
    User-Agent: ""
    Origin: ""
---------------------------------------------

Approve? [y/N]

```

Approving this transaction causes Clef to prompt the user to provide the password for
the sender account. Providing the password enables the transaction to be signed and sent to
Geth for broadcasting to the network. The details of the signed transaction are displayed 
in the console. Account passwords can also be stored in Clef's encrypted vault so that they
do not have to be manually entered - [more on this below](#account-passwords).


## Automatic rules

For most users, manually confirming every transaction is the right way to use Clef because a 
human-in-the-loop can review every action. However, there are cases when it makes sense to 
set up some rules which permit Clef to sign a transaction without prompting the user. 

For example, well defined rules such as:

* Auto-approve transactions with Uniswap v2, with value between 0.1 and 0.5 ETH 
  per 24h period
* Auto-approve transactions to address `0xD9C9Cd5f6779558b6e0eD4e6Acf6b1947E7fA1F3` 
  as long as gas < 44k and gasPrice < 80Gwei

can be encoded and intepreted by Clef's built-in ruleset engine. 

### Rule files

Rules are implemented as Javascript code in `js` files. The ruleset engine includes the 
same methods as the JSON_RPC defined in the [UI Protocol](/docs/_clef/datatypes.md). 
The following code snippet demonstrates a rule file that approves a transaction if it 
satisfies the following conditions:

* the recipient is `0xae967917c465db8578ca9024c205720b1a3651a9`
* the value is less than 50000000000000000 wei (0.05 ETH)

and approves account listing if:

* the request has arrived via ipc

```js
//ancillary function for formatting numbers
function asBig(str) {
	if (str.slice(0, 2) == "0x") {
		return new BigNumber(str.slice(2), 16)
	}
	return new BigNumber(str)
}

// Approve transactions to a certain contract if value is below a certain limit
function ApproveTx(req) {
	var limit = big.Newint("0xb1a2bc2ec50000")
	var value = asBig(req.transaction.value);

	if (req.transaction.to.toLowerCase() == "0xae967917c465db8578ca9024c205720b1a3651a9") 
        && value.lt(limit)) {
	return "Approve"
	}
    else{
        return "Reject"
    }
}

// Approve listings if request made from IPC
function ApproveListing(req){
    if (req.metadata.scheme == "ipc"){ return "Approve"}
}
// returning nothing passes the decision to the next UI for manual assessment
```

There are three possible outcomes to this ruleset that are handled in different ways:

| Return value      |    Action   |
| ----------- | ----------- |
| "Approve"      | Auto-approve request       |
| "Reject"   | Auto-approve request        |
| Error |  Pass decision to UI for manual approval  |
| Unexpected value | Pass decision to UI for manual approval   |
| Nothing   | Pass decision to UI for manual approval  |


### Attestations

Clef will not just accept and run arbitrary scripts - that would create an attack vector because a malicious party could
change the rule file. Instead, the user explicitly *attests* to a rule file, which involves injecting the file's SHA256 
hash into Clef's secure store. The following code snippet shows how to calculate a SHA256 hash for a file named `rules.js` 
and pass it to Clef. Note that Clef will prompt the user to provide the master password because the Clef store has to 
be decrypted in order to add the attestation to it.

```sh
# calculate hash
sha256sum rules.js

# attest to rules.js in Clef
clef attest 645b58e4f945e24d0221714ff29f6aa8e860382ced43490529db1695f5fcc71c
```

Once this attestation has been added to the Clef store, it can be used to automatically approve 
interactions that satisfy the conditions encoded in `rules.js` in Clef. 


### Account passwords

The rules described in `rules.js` above require access to the accounts in the Clef keystore which 
are protected by user-defined passwords. The signer therefore requires access to these passwords 
in order to automatically unlock the keystore and sign data and transactions using the accounts.

This is done using `clef setpw`, passing the account address as the sole argument:

```sh
clef setpw 0xd9c9cd5f6779558b6e0ed4e6acf6b1947e7fa1f3
```

which displays the following in the terminal:

```terminal
Please enter a password to store for this address:
Password:
Repeat password:

Decrypt master seed of clef
Password:
INFO [07-01|14:05:56.031] Credential store updated   key=0xd9c9cd5f6779558b6e0ed4e6acf6b1947e7fa1f3
```

Note that Clef does not really 'unlock' an account, it just abstracts the process of providing the
password away from the end-user in specific, predefined scenarios. If an account password 
exists in the Clef vault and the rule evaluates to "Approve" then Clef decrypts the password, 
uses it to decrypt the key, does the requested signing and then re-locks the account. 


### Implementing rules

Clef can be instructed to run an attested rule file simply by passing the path to `rules.js` 
to the `--rules` flag:

```sh
clef --keystore go-ethereum/goerli-data/ --configdir go-ethereum/goerli-data/clef --chainid 5 --rules rules.js
```

The following logs will be displayed in the terminal:

```
INFO [07-01|13:39:49.726] Rule engine configured                   file=rules.js
INFO [07-01|13:39:49.726] Starting signer                          chainid=5 keystore=$go-ethereum/goerli-data/ light-kdf=false advanced=false
DEBUG[07-01|13:39:49.726] FS scan times                            list=35.15µs set=4.251µs diff=2.766µs
DEBUG[07-01|13:39:49.727] Ledger support enabled
DEBUG[07-01|13:39:49.727] Trezor support enabled via HID
DEBUG[07-01|13:39:49.727] Trezor support enabled via WebUSB
INFO [07-01|13:39:49.728] Audit logs configured                    file=audit.log
DEBUG[07-01|13:39:49.728] IPC registered                           namespace=account
INFO [07-01|13:39:49.728] IPC endpoint opened                      url=go-ethereum/goerli-data/clef/clef.ipc
------- Signer info -------
* intapi_version : 7.0.0
* extapi_version : 6.0.0
* extapi_http : n/a
* extapi_ipc : go-ethereum/goerli-data/clef/clef.ipc
```

Any request that satisfies the ruleset will now be auto-approved by the rule file, for example 
the following request to sign a transaction made using the Geth Javascript console 
(note that the password for account `0xd9c9cd5f6779558b6e0ed4e6acf6b1947e7fa1f3`
has already been provided to `setpw` and the recipient and value comply with the rules in `rules.js`):

```js
var tx = {to: "0xae967917c465db8578ca9024c205720b1a3651a9", from: "0xd9c9cd5f6779558b6e0ed4e6acf6b1947e7fa1f3", value: web3.toWei(0.01, "ether")}
eth.sendTransaction(tx)
```

By contrast, the following transactions *do not* satisfy the rules in `rules.js`:

```js
// violate maximum transaction value condition
var tx = {to: "0xae967917c465db8578ca9024c205720b1a3651a9", from: "0xd9c9cd5f6779558b6e0ed4e6acf6b1947e7fa1f3", value: web3.toWei(1, "ether")}
eth.sendTransaction(tx)
```

```js
// violate recipient condition
var tx = {to: "0xae967917c465db8578ca9024c205720b1a3651a9", from: "0xd4c4bb7d6889453c6c6ea3e9eab3c4177b4fbcc3", value: web3.toWei(0.01, "ether")}
eth.sendTransaction(tx)
```

These latter two transactions, that do not satisfy the encoded rules in `rules.js`, are not automatically approved, but instead pass the 
decision back to the UI for manual approval by the user.


### Summary of basic usage

To summarize, the steps required to run Clef with an automated ruleset that requires account access is as follows:

**1)** Define rules as Javascript and save as a `.js` file, e.g. `rules.js`
 
**2)** Calculate hash of rule file using `sha256sum rules.js`
 
**3)** Attest the rules in Clef using `clef attest <hash>`
 
**4)** Set account passwords in Clef using `clef --setpw <address>`
 
**5)** Start Clef with rule file enabled using `clef --keystore <path-to-keystore> --chainid <chainID> --rules rules.js`

**6)** Make requests directly to Clef using the external API or connect to Geth by passing `--signer=<path to clef.ipc>` at Geth startup
 
 
## More rules

Since rules are defined as Javascript code, rulesets of arbitrary complexity can be created and they can
impose conditions on any part of a transaction, not only the recipient and value.

A simple example is implementing a "whitelist" of recipients where transactions that have those
accounts in the `to` field are automatically signed (for example perhaps transactions between 
a user's own accounts might be whitelisted):

```js
function ApproveTx(r) {
	if (r.transaction.to.toLowerCase() == "0xd4c4bb7d6889453c6c6ea3e9eab3c4177b4fbcc3") {
		return "Approve"
	}
	if (r.transaction.to.toLowerCase() == "0xae967917c465db8578ca9024c205720b1a3651a9") {
		return "Reject"
	}
	// Otherwise goes to manual processing
}
```

In addition to addresses and values, other properties of a request can also be incorporated 
into a ruleset. The example below demonstrates a ruleset for `approve_signData` imposing 
the following conditions on a transaction's sender and message data.

1. The sender must be `0xd9c9cd5f6779558b6e0ed4e6acf6b1947e7fa1f3`
2. The transaction message must include the text `wen-merge`, which is `77656E2D6D65726765` in hex.

If these conditions are satisfied then the transaction is auto-approved (assuming the password for 
`0xd9c9cd5f6779558b6e0ed4e6acf6b1947e7fa1f3` has been provided to `setpw`).

```js
function ApproveListing() {
    return "Approve"
}

function ApproveSignData(req) {
    if (req.address.toLowerCase() == "0xd9c9cd5f6779558b6e0ed4e6acf6b1947e7fa1f3") {
        if (req.messages[0].value.indexOf("wen-merge") >= 0) {
            return "Approve"
        }
        return "Reject"
    }
    // Otherwise goes to manual processing
}
```

This file should be saved as a `.js` file, hashed and attested in Clef:

```sh
sha256sum rules.js
```

which returns:

```terminal
84d9e70aa30d0e5ffb3c4b376c9490f428390a196bfdc1d36770ffd2bbe66845  rules.js
```

then:

```sh
clef attest 84d9e70aa30d0e5ffb3c4b376c9490f428390a196bfdc1d36770ffd2bbe66845
```

which returns:

```terminal
Decrypt master seed of clef
Password:
INFO [07-01|14:11:28.509] Ruleset attestation updated    sha256=84d9e70aa30d0e5ffb3c4b376c9490f428390a196bfdc1d36770ffd2bbe66845
```

Then, Clef can be restarted with the new rules in place:

```sh
clef --keystore go-ethereum/goerli-data/clef --configdir go-ethereum/goerli-data/clef --chainid 5 --rules rules.js
```

```terminal
INFO [07-01|14:12:41.636] Rule engine configured                   file=rules.js
INFO [07-01|14:12:41.636] Starting signer                          chainid=5 keystore=go-ethereum/goerli-data/clef/keystore light-kdf=false advanced=false
DEBUG[07-01|14:12:41.636] FS scan times                            list=46.722µs set=4.47µs diff=2.157µs
DEBUG[07-01|14:12:41.637] Ledger support enabled
DEBUG[07-01|14:12:41.637] Trezor support enabled via HID
DEBUG[07-01|14:12:41.638] Trezor support enabled via WebUSB
INFO [07-01|14:12:41.638] Audit logs configured                    file=audit.log
DEBUG[07-01|14:12:41.638] IPC registered                           namespace=account
INFO [07-01|14:12:41.638] IPC endpoint opened                      url=go-ethereum/goerli-data/clef/clef.ipc
------- Signer info -------
* intapi_version : 7.0.0
* extapi_version : 6.0.0
* extapi_http : n/a
* extapi_ipc : go-ethereum/goerli-data/clef/clef.ipc
```

Finally, a request can be submitted to test that the rules are being applied as expected. 
Here, Clef is used independently of Geth by making a request via RPC, but the same logic 
would be imposed if the request was made via a connected Geth node. Some arbitrary text 
will be included in the message data that includes the term `wen-merge`. The plaintext 
`clefdemotextthatincludeswen-merge` is `636c656664656d6f7465787474686174696e636c7564657377656e2d6d65726765` 
when represented as a hexadecimal string. This can be passed as data to an `account_signData` 
request as follows:

```sh
echo '{"id": 1, "jsonrpc":"2.0", "method":"account_signData", "params":["data/plain", "0x636c656664656d6f7465787474686174696e636c7564657377656e2d6d65726765"]}' | nc -U ~/go-ethereum.goerli-data/clef/clef.ipc
```

This will be automatically signed, returning a result that looks like the following:

```terminal
{"jsonrpc":"2.0","id":1,"result":"0x4f93e3457027f6be99b06b3392d0ebc60615ba448bb7544687ef1248dea4f5317f789002df783979c417d969836b6fda3710f5bffb296b4d51c8aaae6e2ac4831c"}
```

Alternatively, a request that does not include the phrase `wen-merge` will not automatically approve. For example, the following request passes the hexadecimal
string representing the plaintext `clefdemotextwithoutspecialtext`:

```sh
echo '{"id": 1, "jsonrpc":"2.0", "method":"account_signData", "params":["data/plain", "0x636c656664656d6f74657874776974686f75747370656369616c74657874"]}' | nc -U ~/go-ethereum.goerli-data/clef/clef.ipc
```
This returns a `Request denied` message as follows:

```terminal
{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"Request denied"}}
```

Meanwhile, in the output logs in the Clef terminal you can see:
```text
INFO [02-21|14:42:41] Op approved
INFO [02-21|14:42:56] Op rejected
```

The signer also stores all traffic over the external API in a log file. 
The last 4 lines shows the two requests and their responses:

```text
$ tail -n 4 audit.log
t=2022-07-01T15:52:14+0300 lvl=info msg=SignData   api=signer type=request  metadata="{\"remote\":\"NA\",\"local\":\"NA\",\"scheme\":\"NA\",\"User-Agent\":\"\",\"Origin\":\"\"}" addr="0xd9c9cd5f6779558b6e0ed4e6acf6b1947e7fa1f3 [chksum INVALID]" data=0x202062617a6f6e6b2062617a2067617a0a content-type=data/plain
t=2022-07-01T15:52:14+0300 lvl=info msg=SignData   api=signer type=response data=0x636c656664656d6f7465787474686174696e636c7564657377656e2d6d65726765 error=nil
t=2022-07-01T15:52:23+0300 lvl=info msg=SignData   api=signer type=request  metadata="{\"remote\":\"NA\",\"local\":\"NA\",\"scheme\":\"NA\",\"User-Agent\":\"\",\"Origin\":\"\"}" addr="0xd9c9cd5f6779558b6e0ed4e6acf6b1947e7fa1f3 [chksum INVALID]" data=0x636c656664656d6f74657874776974686f75747370656369616c74657874     content-type=data/plain
t=2022-07-01T15:52:23+0300 lvl=info msg=SignData   api=signer type=response data=                                     error="Request denied"
```

More examples, including a ruleset for a rate-limited window, are available on the [Clef Github][rate-limited-window-example]
and on the [Rules page](/docs/clef/rules). 


## Under the hood

The examples on this page have provided step-by-step instructions for verious operations using Clef. 
However, they have not provided much detail as to what is happening under the hood. 
This section will provide some more details about how Clef organizes itself locally.

Initializing Clef with a master password and providing an account password to `clef setpw` 
and attesting a ruleset creates the following files in the directory `~/.clef/` 
(this path is independent of the paths provided to `--keystore` and `--configdir` on startup):

```terminal
# displayed using $ ls -laR ~/.clef/

/home/user/.clef/:
total 24
drwxr-x--x   3 user user  4096 Jul  1 13:45 .
drwxr-xr-x 102 user user 12288 Jul  1 13:39 ..
drwx------   2 user user  4096 Jul  1 13:25 02f90c0603f4f2f60188
-r--------   1 user user   868 Jun 28 13:55 masterseed.json

/home/user/.clef/02f90c0603f4f2f60188:
total 12
drwx------ 2 user user 4096 Jul  1 13:25 .
drwxr-x--x 3 user user 4096 Jul  1 13:45 ..
-rw------- 1 user user  159 Jul  1 13:25 config.json
-rw------- 1 user user  115 Jul  1 13:35 credentials.json
```

The file `masterseed.json` includes a json object containing the masterseed which was used to derive
the vault directory (in this case `02f90c0603f4f2f60188`). The vault is encrypted using a password
which is also derived from the masterseed. Inside the vault are two subdirectories: 

`credentials.json`
 
`config.json`
 

Inside `credentials.json` are the confidential `ksp` data (standing for "keystore pass" - these
are the account passwords used to unlock the keystore).

The `config.json` file contains encrypted key/value pairs for configuration data. Usually 
this is only the `sha256` hashes of any attested rulesets. 

Vault locations map uniquely to masterseeds so that multiple instances of Clef can co-exist 
each with their own attested rules and their own set of keystore passwords. This is useful for, 
for example, maintaining separate setups for Mainnet and testnets.

The contents of each of these json files can be viewed using `cat` and should look something 
like the following:

For `config.json`:

```sh
cat ~/.clef/02f90c0603f4f2f60188/config.json
```

```terminal
{"ruleset_sha256":{"iv":"SWWEtnl+R+I+wfG7","c":"I3fjmwmamxVcfGax7D0MdUOL29/rBWcs73WBILmYK0o1CrX7wSMc3y37KsmtlZUAjp0oItYq01Ow8VGUOzilG91tDHInB5YHNtm/YkufEbo="}}
```

and for `credentials.json`:

```sh
cat ~/.clef/02f90c0603f4f2f60188/config.json
```

```terminal
{"0xd9c9cd5f6779558b6e0ed4e6acf6b1947e7fa1f3": {"iv": "6SC062CfaUW8uSqH","c":"C+S5kaJyrarrxrAESs4EmPjL5zmg5tRh0Q=="}}
```

## Geth integration

This tutorial has bounced back and forth between demonstrating Clef as a standalone tool by making 
'manual` JSON RPC requests from the terminal and integrating it as a backend singer for Geth. 
Using Clef for account management is considered best practise for Geth users because of the additional 
security benefits it offers over and above what it offered by Geth's built-in accounts module. Clef is
far more flexible and composable than Geth's built-in account management tool and can interface directly 
with hardware wallets, while Apps and wallets can request signatures directly from Clef.

Ultimately, the goal is to deprecate Geth's account management tools completely and replace them with 
Clef. Until then, users are simply encouraged to choose to use Clef as an optional backend signer for Geth. 
In addition to the examples on this page, the [Getting started tutorial](/docs/_getting-started/index.md) 
also demonstrates Clef/Geth integration.


## Summary

This page includes step-by-step instructions for basic and intermediate uses of Clef, including using 
it as a standalone app and a backend signer for Geth. Further information is available on our other 
Clef pages, including [Introduction](/docs/clef/introduction), [Setup](/docs/clef/setup), 
[Rules](/docs/clef/rules), [Communication Datatypes](/docs/clef/datatypes) and [Communication APIs](/docs/clef/apis). 
Also see the [Clef Github](https://github.com/ethereum/go-ethereum/tree/master/cmd/clef) for further reading.


[rate-limited-window-example]:https://github.com/ethereum/go-ethereum/blob/master/cmd/clef/rules.md#example-1-ruleset-for-a-rate-limited-window