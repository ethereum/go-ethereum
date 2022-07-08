---
title: Tutorial
sort_key: A
---

This page provides a step-by-step walkjthrough tutorial demonstrating basic usage of Clef. This 
includes manual approvals and automated rules.

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

This tutorial will use Clef with Geth on the Goerli testnet. The accounts used will be in the Goerli keystore with the
path `~/go-ethereum/goerli-data/keystore`. The tutorial assumes there are two accounts in this keystore. Instructions
for creating accounts can be found on the [Account managament page](/docs/interface/managing-your-accounts). Note that Clef can also interact with hardware wallets, although that is
not demonstrated here.

Clef should be started before Geth, otherwise Geth will complain that it cannot find a Clef instance to connect to. 
Clef should be started with the correct `chainid` for Goerli. Clef itself does not connect to a blockchain, but the
`chainID` parameter is included in the data that is aggregated to form a signature. Clef also needs a path to the correct
keystore passed to the `--keystore` command. A custom path to the config directory can also be provided. This is where the
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

Clef starts up in CLI (Command Line Interface) mode by default. Arbitrary remote processes may *request* account 
interactions (e.g. sign a transaction), which the user can individually *confirm* or *deny*.

The code snippet below shows a request made to Clef via its *External API endpoint* using 
[NetCat](http://netcat.sourceforge.net/). The request invokes the ["account_list"](/docs/_clef/apis#accountlist) 
endpoint which lists the accounts in the keystore. This command should be run in a new terminal.

```sh
echo '{"id": 1, "jsonrpc": "2.0", "method": "account_list"}' | nc -U ~/.clef/clef.ipc
```

The terminal used to send the command will now hang. This is because the process is awaiting confirmation from
Clef. Switching to the Clef console reveals Clef's prompt to the user to confirm or deny the request:

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

Depending on whether the request is approved or denied, the NetCat process in the other terminal will receive one
of the following responses:

```text
{"jsonrpc":"2.0","id":1,"result":["0xd9c9cd5f6779558b6e0ed4e6acf6b1947e7fa1f3","0x086278a6c067775f71d6b2bb1856db6e28c30418"]}

or

{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"Request denied"}}
```

Apart from listing accounts, you can also *request* creating a new account, signing transactions and data or recovering signatures.
The available methods are available in the Clef [External API Spec](https://github.com/ethereum/go-ethereum/tree/master/cmd/clef#external-api-1) 
and the [External API Changelog](https://github.com/ethereum/go-ethereum/blob/master/cmd/clef/extapi_changelog.md).

*Note, the number of things you can do from the External API is deliberately small to limit the power of remote calls as much as possible! 
Clef has an [Internal API](https://github.com/ethereum/go-ethereum/tree/master/cmd/clef#ui-api-1) too for the UI (User Interface) which 
is much richer and can support custom interfaces on top. But that's out of scope here.*

The example above used Clef completely independently of Geth. However, the pattern of `request - confirm - result` 
demonstrated above using NetCat also applies to any interaction with the local Geth node that touches accounts, 
including requests made using RPC or an attached Javascript console. To demonstrate this, Geth can be started,
specifying Clef as the signer:

```
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

Switching to the Clef terminal reveals that this is because the request is awaiting explicit confirmation from the user.
The log is identical to the one shown above, when the same request for account information was made to Clef via Netcat.

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

In this mode, the user is required to manually confirm every action that touches account data, including querying accounts
signing and sending transactions.

The example below shows an ether transaction between the two accounts in the keystore using `eth.sendTransaction` in the attached
Javascript console.

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
in the console.


## Automatic rules

For most users, manually confirming every transaction is the right way to use Clef because a 
human-in-the-loop can review every action. However, there are cases when it makes sense to 
set up some rules which permit Clef to sign a transaction without prompting the user. 

For example, well defined rules such as:

* Auto-approve transactions with Uniswap v2, with value between 0.1 and 0.5 ETH per 24h period
* Auto-approve transactions to address `0xD9C9Cd5f6779558b6e0eD4e6Acf6b1947E7fA1F3` as long as gas < 44k and gasPrice < 80Gwei

can be encoded and intepreted by Clef's built-in ruleset engine. 

### Rule files

Rules are implemented as Javascript code in `js` files. The ruleset engine includes the same methods as the 
JSON_RPC defined in the [UI Protocol](/docs/_clef/datatypes.md). The following code snippet demonstrates
a rule file that approves a transaction if it satisfies the following conditions:

* the recipient is `0xae967917c465db8578ca9024c205720b1a3651a9`
* the value is less than 50000000000000000 wei (0.05 ETH)
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

	if (req.transaction.to.toLowerCase() == "0xae967917c465db8578ca9024c205720b1a3651a9") && value.lt(limit)) {
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
 
* **Javascript returns "Approve"**:
 
    Auto-approve request
 
* **Javascript returns "Reject"**:
 
    Auto-reject request
 
* **Javascript returns Error, unexpected value or nothing**:
 
    Pass on to next ui: the regular UI channel
 
 
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

The rules described in `rules.js` above require access to the accounts in the Clef keystore which are protected
by user-defined passwords. The signer therefore requires access to these passwords in order to automatically 
unlock the keystore and sign data and transactions using the accounts.

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


### Implementing rules

Clef can be instructed to run the attested rule file simply by passing the path to `rules.js` 
to the `--rules` flag:

```sh
clef --keystore go-ethereum/goerli-data/ --chainid 5 --rules rules.js
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

Any request that satisfies the ruleset will now be auto-approved by the rule file, for example the following request to 
sign a transaction made in the Geth Javascript console (note that the password for account `0xd9c9cd5f6779558b6e0ed4e6acf6b1947e7fa1f3`)
has been provided to `--setpw` and the recipient and value comply with the rules in `rules.js`:

```js
var tx = {to: "0xae967917c465db8578ca9024c205720b1a3651a9", from: "0xd9c9cd5f6779558b6e0ed4e6acf6b1947e7fa1f3", value: web3.toWei(0.01, "ether")}
eth.sendTransaction(tx)
```

By contrast, the following transactions will be rejected because they do not satisfy the rules in `rules.js`:

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

These latter two transactions, that do not satisfy the encoded rules in `rules.js`, pass the decision back to the UI for manual approval.


### Summary of basic usage

To summarize, the steps required to run Clef with an automated ruleset that requires account access is as follows:

**1)** define rules as Javascript and save as a `.js` file, e.g. `rules.js`
 
**2)** calculate hash of rule file using `sha256sum rules.js`
 
**3)** attest the rules in Clef using `clef attest <hash>`
 
**4)** set account passwords in Clef using `clef --setpw <address>`
 
**5)** start Clef with rule file enabled using `clef --keystore <path-to-keystore> --chainid <chainID> --rules rules.js`
 
 
## Advanced rules

In order to make more useful rules - like signing transactions - the signer needs access to the passwords needed
to unlock keys from the keystore. You can inject an unlock password via `clef setpw`.

```text
$ clef setpw 0xd9c9cd5f6779558b6e0ed4e6acf6b1947e7fa1f3

Please enter a password to store for this address:
Password:
Repeat password:

Decrypt master seed of clef
Password:
INFO [07-01|14:05:56.031] Credential store updated                 key=0xd9c9cd5f6779558b6e0ed4e6acf6b1947e7fa1f3
```

Now let's update the rules to make use of the new credentials:

```js
function ApproveListing() {
    return "Approve"
}

function ApproveSignData(req) {
    if (req.address.toLowerCase() == "0xd9c9cd5f6779558b6e0ed4e6acf6b1947e7fa1f3") {
        if (req.messages[0].value.indexOf("bazonk") >= 0) {
            return "Approve"
        }
        return "Reject"
    }
    // Otherwise goes to manual processing
}
```

In this example:

- Any requests to sign data with the account `0xd9c9...` will be:
    - Auto-approved if the message contains `bazonk`,
    - Auto-rejected if the message does not contain `bazonk`,
- Any other requests will be passed along for manual confirmation.

*Note, to make this example work, please use you own accounts. You can create a new account either via Clef or the traditional account CLI tools. If the latter was chosen, make sure both Clef and Geth use the same keystore by specifying `--keystore path/to/your/keystore` when running Clef.*

Attest the new rule file so that Clef will accept loading it:

```text
$ sha256sum rules.js
f163a1738b649259bb9b369c593fdc4c6b6f86cc87e343c3ba58faee03c2a178  rules.js

$ clef attest f163a1738b649259bb9b369c593fdc4c6b6f86cc87e343c3ba58faee03c2a178
Decrypt master seed of clef
Password:
INFO [07-01|14:11:28.509] Ruleset attestation updated              sha256=f163a1738b649259bb9b369c593fdc4c6b6f86cc87e343c3ba58faee03c2a178
```

Restart Clef with the new rules in place:

```
$ clef --keystore ~/.ethereum/rinkeby/keystore --chainid 4 --rules rules.js

INFO [07-01|14:12:41.636] Rule engine configured                   file=rules.js
INFO [07-01|14:12:41.636] Starting signer                          chainid=4 keystore=$HOME/.ethereum/rinkeby/keystore light-kdf=false advanced=false
DEBUG[07-01|14:12:41.636] FS scan times                            list=46.722µs set=4.47µs diff=2.157µs
DEBUG[07-01|14:12:41.637] Ledger support enabled
DEBUG[07-01|14:12:41.637] Trezor support enabled via HID
DEBUG[07-01|14:12:41.638] Trezor support enabled via WebUSB
INFO [07-01|14:12:41.638] Audit logs configured                    file=audit.log
DEBUG[07-01|14:12:41.638] IPC registered                           namespace=account
INFO [07-01|14:12:41.638] IPC endpoint opened                      url=$HOME/.clef/clef.ipc
------- Signer info -------
* intapi_version : 7.0.0
* extapi_version : 6.0.0
* extapi_http : n/a
* extapi_ipc : $HOME/.clef/clef.ipc
```

Then test signing, once with `bazonk` and once without:

```
$ echo '{"id": 1, "jsonrpc":"2.0", "method":"account_signData", "params":["data/plain", "0xd9c9cd5f6779558b6e0ed4e6acf6b1947e7fa1f3", "0x202062617a6f6e6b2062617a2067617a0a"]}' | nc -U ~/.clef/clef.ipc
{"jsonrpc":"2.0","id":1,"result":"0x4f93e3457027f6be99b06b3392d0ebc60615ba448bb7544687ef1248dea4f5317f789002df783979c417d969836b6fda3710f5bffb296b4d51c8aaae6e2ac4831c"}

$ echo '{"id": 1, "jsonrpc":"2.0", "method":"account_signData", "params":["data/plain", "0xd9c9cd5f6779558b6e0ed4e6acf6b1947e7fa1f3", "0x2020626f6e6b2062617a2067617a0a"]}' | nc -U ~/.clef/clef.ipc
{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"Request denied"}}
```

Meanwhile, in the Clef output log you can see:
```text
INFO [02-21|14:42:41] Op approved
INFO [02-21|14:42:56] Op rejected
```

The signer also stores all traffic over the external API in a log file. The last 4 lines shows the two requests and their responses:

```text
$ tail -n 4 audit.log
t=2019-07-01T15:52:14+0300 lvl=info msg=SignData   api=signer type=request  metadata="{\"remote\":\"NA\",\"local\":\"NA\",\"scheme\":\"NA\",\"User-Agent\":\"\",\"Origin\":\"\"}" addr="0xd9c9cd5f6779558b6e0ed4e6acf6b1947e7fa1f3 [chksum INVALID]" data=0x202062617a6f6e6b2062617a2067617a0a content-type=data/plain
t=2019-07-01T15:52:14+0300 lvl=info msg=SignData   api=signer type=response data=4f93e3457027f6be99b06b3392d0ebc60615ba448bb7544687ef1248dea4f5317f789002df783979c417d969836b6fda3710f5bffb296b4d51c8aaae6e2ac4831c error=nil
t=2019-07-01T15:52:23+0300 lvl=info msg=SignData   api=signer type=request  metadata="{\"remote\":\"NA\",\"local\":\"NA\",\"scheme\":\"NA\",\"User-Agent\":\"\",\"Origin\":\"\"}" addr="0xd9c9cd5f6779558b6e0ed4e6acf6b1947e7fa1f3 [chksum INVALID]" data=0x2020626f6e6b2062617a2067617a0a     content-type=data/plain
t=2019-07-01T15:52:23+0300 lvl=info msg=SignData   api=signer type=response data=                                     error="Request denied"
```

For more details on writing automatic rules, please see the [rules spec](https://github.com/ethereum/go-ethereum/blob/master/cmd/clef/rules.md).


## Under the hood

While doing the operations above, these files have been created:

```text
$ ls -laR ~/.clef/

$HOME/.clef/:
total 24
drwxr-x--x   3 user user  4096 Jul  1 13:45 .
drwxr-xr-x 102 user user 12288 Jul  1 13:39 ..
drwx------   2 user user  4096 Jul  1 13:25 02f90c0603f4f2f60188
-r--------   1 user user   868 Jun 28 13:55 masterseed.json

$HOME/.clef/02f90c0603f4f2f60188:
total 12
drwx------ 2 user user 4096 Jul  1 13:25 .
drwxr-x--x 3 user user 4096 Jul  1 13:45 ..
-rw------- 1 user user  159 Jul  1 13:25 config.json

$ cat ~/.clef/02f90c0603f4f2f60188/config.json
{"ruleset_sha256":{"iv":"SWWEtnl+R+I+wfG7","c":"I3fjmwmamxVcfGax7D0MdUOL29/rBWcs73WBILmYK0o1CrX7wSMc3y37KsmtlZUAjp0oItYq01Ow8VGUOzilG91tDHInB5YHNtm/YkufEbo="}}
```

In `$HOME/.clef`, the `masterseed.json` file was created, containing the master seed. This seed was then used to derive a few other things:

- **Vault location**: in this case `02f90c0603f4f2f60188`.
   - If you use a different master seed, a different vault location will be used that does not conflict with each other (e.g. `clef --signersecret /path/to/file`). This allows you to run multiple instances of Clef, each with its own rules (e.g. mainnet + testnet).
- **`config.json`**: the encrypted key/value storage for configuration data, currently only containing the key `ruleset_sha256`, the attested hash of the automatic rules to use.





## Geth integration

Of course, as awesome as Clef is, it's not feasible to interact with it via JSON RPC by hand. Long term, we're hoping to convince the general Ethereum community to support Clef as a general signer (it's only 3-5 methods), thus allowing your favorite DApp, Metamask, MyCrypto, etc to request signatures directly.

Until then however, we're trying to pave the way via Geth. Geth v1.9.0 has built in support via `--signer <API endpoint>` for using a local or remote Clef instance as an account backend!

We can try this by running Clef with our previous rules on Rinkeby (for now it's a good idea to allow auto-listing accounts, since Geth likes to retrieve them once in a while).

```text
$ clef --keystore ~/.ethereum/rinkeby/keystore --chainid 4 --rules rules.js
```

In a different window we can start Geth, list our accounts, even list our wallets to see where the accounts originate from:

```text
$ geth --rinkeby --signer=~/.clef/clef.ipc console

> eth.accounts
["0xd9c9cd5f6779558b6e0ed4e6acf6b1947e7fa1f3", "0x086278a6c067775f71d6b2bb1856db6e28c30418"]

> personal.listWallets
[{
    accounts: [{
        address: "0xd9c9cd5f6779558b6e0ed4e6acf6b1947e7fa1f3",
        url: "extapi://$HOME/.clef/clef.ipc"
    }, {
        address: "0x086278a6c067775f71d6b2bb1856db6e28c30418",
        url: "extapi://$HOME/.clef/clef.ipc"
    }],
    status: "ok [version=6.0.0]",
    url: "extapi://$HOME/.clef/clef.ipc"
}]

> eth.sendTransaction({from: eth.accounts[0], to: eth.accounts[0]})
```

Lastly, when we requested a transaction to be sent, Clef prompted us in the original window to approve it:

```text
--------- Transaction request-------------
to:       0xD9C9Cd5f6779558b6e0eD4e6Acf6b1947E7fA1F3
from:     0xD9C9Cd5f6779558b6e0eD4e6Acf6b1947E7fA1F3 [chksum ok]
value:    0 wei
gas:      0x5208 (21000)
gasprice: 1000000000 wei
nonce:    0x2366 (9062)

Request context:
	NA -> NA -> NA

Additional HTTP header data, provided by the external caller:
	User-Agent:
	Origin:
-------------------------------------------
Approve? [y/N]:
> y
```

:boom:

*Note, if you enable the external signer backend in Geth, all other account management is disabled. This is because long term we want to remove account management from Geth.*
