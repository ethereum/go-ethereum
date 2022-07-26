---
title: Introduction to Clef
sort_key: A
---

{:toc}
-   this will be removed by the toc
  
## What is Clef?

Clef is a tool for **signing transactions and data** in a secure local environment. 
It is intended to become a more composable and secure replacement for Geth's built-in 
account management. Clef decouples key management from Geth itself, meaning it can be 
used as an independent, standalone key management and signing application, or it
can be integrated into Geth. This provides a more flexible modular tool compared to 
Geth's account manager. Clef can be used safely in situations where access to Ethereum is 
via a remote and/or untrusted node because signing happens locally, either manually or 
automatically using custom rulesets. The separation of Clef from the node itself enables it 
to run as a daemon on the same machine as the client software, on a secure usb-stick like 
[USB armory](https://inversepath.com/usbarmory), or even a separate VM in a 
[QubesOS](https://www.qubes-os.org/) type setup.

## Installing and starting Clef

Clef comes bundled with Geth and can be built along with Geth and the other bundled tools using:

`make all`

However, Clef is not bound to Geth and can be built on its own using:

`make clef`

Once built, Clef must be initialized. This includes storing some data, some of which is sensitive 
(such as passwords, account data, signing rules etc). Initializing Clef takes that data and 
encrypts it using a user-defined password.

`clef init`

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

## Security model

One of the major benefits of Clef is that it is decoupled from the client software, 
meaning it can be used by users and dapps to sign data and transactions in a secure, 
local environment and send the signed packet to an arbitrary Ethereum entry-point, which 
might include, for example, an untrusted remote node. Alternatively, Clef can simply be
used as a standalone, composable signer that can be a backend component for decentralized 
applications. This requires a secure architecture that separates cryptographic operations 
from user interactions and internal/external communication.

The security model of Clef is as follows:

* A self-contained binary controls all cryptographic operations including encryption, 
  decryption and storage of keystore files, and signing data and transactions.

* A well defined, deliberately minimal "external" API is used to communicate with the 
  Clef binary - Clef considers this external traffic to be UNTRUSTED. This means Clef 
  does not accept any credentials and does not recognize authority of requests received 
  over this channel. Clef listens on `http.addr:http.port` or `ipcpath` - the same as Geth - 
  and expects messages to be formatted using the [JSON-RPC 2.0 standard](https://www.jsonrpc.org/specification). 
  Some of the external API calls require some user interaction (manual approve/deny)- if it is 
  not received responses can be delayed indefinitely.

* Clef communicates with the process that invoked the binary using stin/stout. The process 
  invoking the binary is usually the native console-based user interface (UI) but there is 
  also an API that enables communication with an external UI. This has to be enabled using `--stdio-ui` 
  at startup. This channel is considered TRUSTED and is used to pass approvals and passwords between 
  the user and Clef. 

* Clef does not store keys - the user is responsible for securely storing and backing up keyfiles. 
  Clef does store account passwords in its encrypted vault if they are explicitly provided to 
  Clef by the user to enable automatic account unlocking.

The external API never handles any sensitive data directly, but it can be used to request Clef to
sign some data or a transaction. It is the internal API that controls signing and triggers requests for
manual approval (automatic approves actions that conform to attested rulesets) and passwords.

The general flow for a basic transaction-signing operation using Clef and an Ethereum node such as 
Geth is as follows:

![Clef signing logic](/static/images/clef_sign_flow.png)

In the case illustrated in the schematic above, Geth would be started with `--signer <addr>:<port>` and 
would relay requests to `eth.sendTransaction`. Text in `mono` font positioned along arrows shows the objects
passed between each component.

Most users use Clef by manually approving transactions through the UI as in the schematic above, but it is also
possible to configure Clef to sign transactions without always prompting the user. This requires defining the
precise conditions under which a transaction will be signed. These conditions are known as `Rules` and they are 
small Javascript snippets that are *attested* by the user by injecting the snippet's hash into Clef's secure
whitelist. Clef is then started with the rule file, so that requests that satisfy the conditions in the whitelisted
rule files are automatically signed. This is covered in detail on the [Rules page](/docs/_clef/Rules.md).


## Basic usage

Clef is started on the command line using the `clef` command. Clef can be configured by providing flags and 
commands to `clef` on startup. The full list of command line options is available [below](#command-line-options).
Frequently used options include `--keystore` and `--chainid` which configure the path to an existing keystore
and a network to connect to. These options default to `$HOME/.ethereum/keystore` and `1` (corresponding to
Ethereum Mainnet) respectively. The following code snippet starts Clef, providing a custom path to an existing 
keystore and connecting to the Goerli testnet:

```sh
clef --keystore /my/keystore --chainid 5
```

On starting Clef, the following welcome messgae is displayed in the terminal:

```terminal
WARNING!

Clef is an account management tool. It may, like any software, contain bugs.

Please take care to
- backup your keystore files,
- verify that the keystore(s) can be opened with your password.

Clef is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY.
without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR
PURPOSE. See the GNU General Public License for more details.

Enter 'ok' to proceed:
>
```

Requests requiring account access or signing now require explicit consent in this terminal.
Activities such as sending transactions via a local Geth node's attached Javascript console or
RPC will now hang indefinitely, awaiting approval in this terminal.

A much more detailed Clef tutorial is available on the [Tutorial page](/docs/clef/tutorial).


## Command line options

```sh
COMMANDS:
   init         Initialize the signer, generate secret storage
   attest       Attest that a js-file is to be used
   setpw        Store a credential for a keystore file
   delpw        Remove a credential for a keystore file
   newaccount   Create a new account
   gendoc       Generate documentation about json-rpc format
   help, h      Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --loglevel value        log level to emit to the screen (default: 4)
   --keystore value        Directory for the keystore (default: "$HOME/.ethereum/keystore")
   --configdir value       Directory for Clef configuration (default: "$HOME/.clef")
   --chainid value         Chain id to use for signing (1=mainnet, 3=Ropsten, 4=Rinkeby, 5=Goerli) (default: 1)
   --lightkdf              Reduce key-derivation RAM & CPU usage at some expense of KDF strength
   --nousb                 Disables monitoring for and managing USB hardware wallets
   --pcscdpath value       Path to the smartcard daemon (pcscd) socket file (default: "/run/pcscd/pcscd.comm")
   --http.addr value       HTTP-RPC server listening interface (default: "localhost")
   --http.vhosts value     Comma separated list of virtual hostnames from which to accept requests (server enforced). Accepts '*' wildcard. (default: "localhost")
   --ipcdisable            Disable the IPC-RPC server
   --ipcpath value         Filename for IPC socket/pipe within the datadir (explicit paths escape it)
   --http                  Enable the HTTP-RPC server
   --http.port value       HTTP-RPC server listening port (default: 8550)
   --signersecret value    A file containing the (encrypted) master seed to encrypt Clef data, e.g. keystore credentials and ruleset hash
   --4bytedb-custom value  File used for writing new 4byte-identifiers submitted via API (default: "./4byte-custom.json")
   --auditlog value        File used to emit audit logs. Set to "" to disable (default: "audit.log")
   --rules value           Path to the rule file to auto-authorize requests with
   --stdio-ui              Use STDIN/STDOUT as a channel for an external UI. This means that an STDIN/STDOUT is used for RPC-communication with a e.g. a graphical user interface, and can be used when Clef is started by an external process.
   --stdio-ui-test         Mechanism to test interface between Clef and UI. Requires 'stdio-ui'.
   --advanced              If enabled, issues warnings instead of rejections for suspicious requests. Default off
   --suppress-bootwarn     If set, does not show the warning during boot
```

## Summary

Clef is an external key management and signer tool that comes bundled with Geth but can either be used 
as a backend account manager and signer for Geth or as a completely separate standalone application. Being 
modular and composable it can be used as a component in decentralized applications or to sign data and
transactions in untrusted environments. Clef is intended to eventually replace Geth's built-in account
management tools.
