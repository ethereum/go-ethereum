---
title: Clique signing
description: Instructions for setting up Clef to seal blocks on a Clique network
---

Clique is a proof-of-authority system where new blocks can be created by authorized ‘signers’ only. The initial set of authorized signers is configured in the genesis block. Signers can be authorized and de-authorized using a voting mechanism, thus allowing the set of signers to change while the blockchain operates. Signing blocks in Clique networks classically uses the "unlock" feature of Geth so that each node is always ready to sign without requiring a user to manually provide authorization.

However, using the `--unlock` flag is generally a highly dangerous thing to do because it is indiscriminate, i.e. if an account is unlocked and an attacker obtains access to the RPC api, the attacker can sign anything without supplying a password.

Clef provides a way to safely circumvent `--unlock` while maintaining a enough automation for the network to be useable.

## Prerequisites {#prerequisites}

It is useful to have basic knowledge of private networks and Clef. These topics are covered on our [private networks](/docs/fundamentals/private-network) and [Introduction to Clef](/docs/tools/clef/introduction) pages.

## Prepping a Clique network {#prepping-clique-network}

First of all, set up a rudimentary testnet to have something to sign. Create a new keystore (password `testtesttest`)

```terminal
$ geth account new --datadir ./ddir
INFO [06-16|11:10:39.600] Maximum peer count                       ETH=50 LES=0 total=50
Your new account is locked with a password. Please give a password. Do not forget this password.
Password:
Repeat password:

Your new key was generated

Public address of the key:   0x9CD932F670F7eDe5dE86F756A6D02548e5899f47
Path of the secret key file: ddir/keystore/UTC--2022-06-16T09-10-48.578523828Z--9cd932f670f7ede5de86f756a6d02548e5899f47

- You can share your public address with anyone. Others need it to interact with you.
- You must NEVER share the secret key with anyone! The key controls access to your funds!
- You must BACKUP your key file! Without the key, it's impossible to access account funds!
- You must REMEMBER your password! Without the password, it's impossible to decrypt the key!
```

Create a genesis with that account as a sealer:

```json
{
  "config": {
    "chainId": 15,
    "homesteadBlock": 0,
    "eip150Block": 0,
    "eip155Block": 0,
    "eip158Block": 0,
    "byzantiumBlock": 0,
    "constantinopleBlock": 0,
    "petersburgBlock": 0,
    "clique": {
      "period": 30,
      "epoch": 30000
    }
  },
  "difficulty": "1",
  "gasLimit": "8000000",
  "extradata": "0x00000000000000000000000000000000000000000000000000000000000000009CD932F670F7eDe5dE86F756A6D02548e5899f470000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
  "alloc": {
    "0x9CD932F670F7eDe5dE86F756A6D02548e5899f47": {
      "balance": "300000000000000000000000000000000"
    }
  }
}
```

Initiate Geth:

```sh
$ geth  --datadir ./ddir init genesis.json
```

```terminal
...
INFO [06-16|11:14:54.123] Writing custom genesis block
INFO [06-16|11:14:54.125] Persisted trie from memory database      nodes=1 size=153.00B time="64.715µs"  gcnodes=0 gcsize=0.00B gctime=0s livenodes=1 livesize=0.00B
INFO [06-16|11:14:54.125] Successfully wrote genesis state         database=lightchaindata hash=187412..4deb98
```

At this point a Geth has been initiated with a genesis configuration.

## Prepping Clef {#prepping-clef}

In order to make use of `clef` for signing:

1. Ensure `clef` knows the password for the keystore.
2. Ensure `clef` auto-approves clique signing requests.

These two things are independent of each other. First of all, however, `clef` must be initiated (for this example the password is `clefclefclef`)

```sh
$ clef --keystore ./ddir/keystore --configdir ./clef --chainid 15 --suppress-bootwarn init
```

```terminal
The master seed of clef will be locked with a password.
Please specify a password. Do not forget this password!
Password:
Repeat password:

A master seed has been generated into clef/masterseed.json

This is required to be able to store credentials, such as:
* Passwords for keystores (used by rule engine)
* Storage for JavaScript auto-signing rules
* Hash of JavaScript rule-file

You should treat 'masterseed.json' with utmost secrecy and make a backup of it!
* The password is necessary but not enough, you need to back up the master seed too!
* The master seed does not contain your accounts, those need to be backed up separately!
```

After this operation, `clef` has it's own vault where it can store secrets and attestations.

## Storing passwords in `clef` {#storing-passwords}

With that done, `clef` can be made aware of the password. To do this `setpw <address>` is invoked to store a password for a given address. `clef` asks for the password, and it also asks for the master-password, in order to update and store the new secrets inside the vault.

```sh
$ clef --keystore ./ddir/keystore --configdir ./clef --chainid 15 --suppress-bootwarn setpw 0x9CD932F670F7eDe5dE86F756A6D02548e5899f47
```

```terminal
Please enter a password to store for this address:
Password:
Repeat password:

Decrypt master seed of clef
Password:
INFO [06-16|11:27:09.153] Credential store updated                 set=0x9CD932F670F7eDe5dE86F756A6D02548e5899f47
```

At this point, if Clef is used as a sealer, each block would require manual approval, but without needing to provide the password.

### Testing stored password {#testing-stored-password}

To test that the stored password is correct and being properly handled by Clef, first start `clef`:

```sh
$ clef --keystore ./ddir/keystore --configdir ./clef --chainid 15 --suppress-bootwarn
```

then start Geth:

```sh
$ geth  --datadir ./ddir --signer ./clef/clef.ipc --mine
```

Geth will ask what accounts are present - enter `y` to approve:

```terminal
-------- List Account request--------------
A request has been made to list all accounts.
You can select which accounts the caller can see
  [x] 0x9CD932F670F7eDe5dE86F756A6D02548e5899f47
    URL: keystore:///home/user/tmp/clique_clef/ddir/keystore/UTC--2022-06-16T09-10-48.578523828Z--9cd932f670f7ede5de86f756a6d02548e5899f47
-------------------------------------------
Request context:
	NA - ipc - NA

Additional HTTP header data, provided by the external caller:
	User-Agent: ""
	Origin: ""
Approve? [y/N]:
> y
DEBUG[06-16|11:36:42.499] Served account_list                      reqid=2 duration=3.213768195s
```

After this, Geth will start asking `clef` to sign things:

```terminal
-------- Sign data request--------------
Account:  0x9CD932F670F7eDe5dE86F756A6D02548e5899f47 [chksum ok]
messages:
  Clique header [clique]: "clique header 1 [0x9b08fa3705e8b6e1b327d84f7936c21a3cb11810d9344dc4473f78f8da71e571]"
raw data:
	"\xf9\x02\x14\xa0\x18t\x12:\x91f\xa2\x90U\b\xf9\xac\xc02i\xffs\x9f\xf4\xc9⮷!\x0f\x16\xaa?#M똠\x1d\xccM\xe8\xde\xc7]z\xab\x85\xb5g\xb6\xcc\xd4\x1a\xd3\x12E\x1b\x94\x8at\x13\xf0\xa1B\xfd@ԓG\x94\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\xa0]1%\n\xfc\xee'\xd0e\xce\xc7t\xcc\\?\t4v\x8f\x06\xcb\xf8\xa0P5\xfeN\xea\x0ff\xfe\x9c\xa0V\xe8\x1f\x17\x1b\xccU\xa6\xff\x83E\xe6\x92\xc0\xf8n[H\xe0\x1b\x99l\xad\xc0\x01b/\xb5\xe3c\xb4!\xa0V\xe8\x1f\x17\x1b\xccU\xa6\xff\x83E\xe6\x92\xc0\xf8n[H\xe0\x1b\x99l\xad\xc0\x01b/\xb5\xe3c\xb4!\xb9\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x02\x01\x83z0\x83\x80\x84b\xaa\xf9\xaa\xa0\u0603\x01\n\x14\x84geth\x88go1.18.1\x85linux\x00\x00\x00\x00\x00\x00\x00\xa0\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x88\x00\x00\x00\x00\x00\x00\x00\x00"
data hash:  0x9589ed81e959db6330b3d70e5f8e426fb683d03512f203009f7e41fc70662d03
-------------------------------------------
Request context:
	NA -> ipc -> NA

Additional HTTP header data, provided by the external caller:
	User-Agent: ""
	Origin: ""
Approve? [y/N]:
> y
```

And indeed, after approving with `y`, the password is not required - the signed block is returned to Geth:

```terminal
INFO [06-16|11:36:46.714] Successfully sealed new block            number=1 sealhash=9589ed..662d03 hash=bd20b9..af8b87 elapsed=4.214s
```

This mode of operation offers quite a poor UX because each block to be sealed requires manual approval. That is fixed in the following section.

## Using rules to approve blocks {#using-rules}

Clef rules allow a piece of Javascript take over the Approve/Deny decision. The Javascript snippet has access to the same information as the manual operator.

The first approach, which approves listing, and returns the request data for `ApproveListing`, is demonstrated below:

```js
function ApproveListing() {
  return 'Approve';
}

function ApproveSignData(r) {
  console.log('In Approve Sign data');
  console.log(JSON.stringify(r));
}
```

In order to use a certain ruleset, it must first be 'attested'. This is to prevent someone from modifying a ruleset-file on disk after creation.

```sh
$ clef --keystore ./ddir/keystore --configdir ./clef --chainid 15 --suppress-bootwarn  attest  `sha256sum rules.js | cut -f1`
```

which returns:

```terminal
Decrypt master seed of clef
Password:
INFO [06-16|13:49:00.298] Ruleset attestation updated              sha256=54aae496c3f0eda063a62c73ee284ca9fae3f43b401da847ef30ea30e85e35d1
```

And `clef` can be started, pointing out the `rules.js` file.

```sh
$ clef --keystore ./ddir/keystore --configdir ./clef --chainid 15  --suppress-bootwarn  --rules ./rules.js
```

Once Geth starts asking `clef` to seal blocks, the data will be displayed. From that data, rules can be defined that allow signing clique headers but nothing else.

The actual data that gets passed to the js environment (and which the ruleset display in the terminal) looks as follows:

```json
{
  "content_type": "application/x-clique-header",
  "address": "0x9CD932F670F7eDe5dE86F756A6D02548e5899f47",
  "raw_data": "+QIUoL0guY+66jZpzZh1wDX4Si/ycX4zD8FQqF/1Apy/r4uHoB3MTejex116q4W1Z7bM1BrTEkUblIp0E/ChQv1A1JNHlAAAAAAAAAAAAAAAAAAAAAAAAAAAoF0xJQr87ifQZc7HdMxcPwk0do8Gy/igUDX+TuoPZv6coFboHxcbzFWm/4NF5pLA+G5bSOAbmWytwAFiL7XjY7QhoFboHxcbzFWm/4NF5pLA+G5bSOAbmWytwAFiL7XjY7QhuQEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAICg3pPDoCEYqsY1qDYgwEKFIRnZXRoiGdvMS4xOC4xhWxpbnV4AAAAAAAAAKAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAIgAAAAAAAAAAA==",
  "messages": [
    {
      "name": "Clique header",
      "value": "clique header 2 [0xae525b65bc7f711bc136f502650039cd6959c3abc28fdf0ebfe2a5f85c92f3b6]",
      "type": "clique"
    }
  ],
  "call_info": null,
  "hash": "0x8ca6c78af7d5ae67ceb4a1e465a8b639b9fbdec4b78e4d19cd9b1232046fbbf4",
  "meta": {
    "remote": "NA",
    "local": "NA",
    "scheme": "ipc",
    "User-Agent": "",
    "Origin": ""
  }
}
```

To create an extremely trustless ruleset, the `raw_data` could be verified to ensure it has the right rlp structure for a Clique header:

```sh
 echo "+QIUoL0guY+66jZpzZh1wDX4Si/ycX4zD8FQqF/1Apy/r4uHoB3MTejex116q4W1Z7bM1BrTEkUblIp0E/ChQv1A1JNHlAAAAAAAAAAAAAAAAAAAAAAAAAAAoF0xJQr87ifQZc7HdMxcPwk0do8Gy/igUDX+TuoPZv6coFboHxcbzFWm/4NF5pLA+G5bSOAbmWytwAFiL7XjY7QhoFboHxcbzFWm/4NF5pLA+G5bSOAbmWytwAFiL7XjY7QhuQEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAICg3pPDoCEYqsY1qDYgwEKFIRnZXRoiGdvMS4xOC4xhWxpbnV4AAAAAAAAAKAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAIgAAAAAAAAAAA==" | base64 -d | rlpdump
[
  bd20b98fbaea3669cd9875c035f84a2ff2717e330fc150a85ff5029cbfaf8b87,
  1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347,
  0000000000000000000000000000000000000000,
  5d31250afcee27d065cec774cc5c3f0934768f06cbf8a05035fe4eea0f66fe9c,
  56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421,
  56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421,
  00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000,
  02,
  02,
  7a4f0e,
  "",
  62ab18d6,
  d883010a14846765746888676f312e31382e31856c696e757800000000000000,
  0000000000000000000000000000000000000000000000000000000000000000,
  0000000000000000,
]
```

However, `messages` could also be used. They do not come from the external caller, but are generated internally: `clef` parsed the incoming request and verified the Clique wellformedness of the content. The following simply checks for such a message:

```js
function OnSignerStartup(info) {}

function ApproveListing() {
  return 'Approve';
}

function ApproveSignData(r) {
  if (r.content_type == 'application/x-clique-header') {
    for (var i = 0; i < r.messages.length; i++) {
      var msg = r.messages[i];
      if (msg.name == 'Clique header' && msg.type == 'clique') {
        return 'Approve';
      }
    }
  }
  return 'Reject';
}
```

Attest the ruleset:

```sh
$ clef --keystore ./ddir/keystore --configdir ./clef --chainid 15 --suppress-bootwarn  attest  `sha256sum rules.js | cut -f1`
```

returning

```terminal
Decrypt master seed of clef
Password:
INFO [06-16|14:18:53.476] Ruleset attestation updated              sha256=7d5036d22d1cc66599e7050fb1877f4e48b89453678c38eea06e3525996c2379
```

Run `clef`:

```sh
$ clef --keystore ./ddir/keystore --configdir ./clef --chainid 15  --suppress-bootwarn  --rules ./rules.js
```

Run Geth:

```sh
$ geth  --datadir ./ddir --signer ./clef/clef.ipc --mine
```

And `clef` should now happily sign blocks:

```terminal
DEBUG[06-16|14:20:02.136] Served account_version                   reqid=1 duration="131.38µs"
INFO [06-16|14:20:02.289] Op approved
DEBUG[06-16|14:20:02.289] Served account_list                      reqid=2 duration=4.672441ms
INFO [06-16|14:20:02.303] Op approved
DEBUG[06-16|14:20:03.450] Served account_signData                  reqid=3 duration=1.152074109s
INFO [06-16|14:20:03.456] Op approved
DEBUG[06-16|14:20:04.267] Served account_signData                  reqid=4 duration=815.874746ms
INFO [06-16|14:20:32.823] Op approved
DEBUG[06-16|14:20:33.584] Served account_signData                  reqid=5 duration=766.840681ms

```

## Refinements {#refinements}

If an attacker find the Clef "external" interface (which would only happen if you start it with `http` enabled), they

- cannot make it sign arbitrary transactions,
- cannot sign arbitrary data message,

However, they could still make it sign e.g. 1000 versions of a certain block height, making the chain very unstable.

It is possible for rule execution to be stateful (i.e. storing data). In this case, one could, for example, store what block heights have been sealed and reject sealing a particular block height twice. In other words, these rules could be used to build a miniature version of an execution layer slashing-db.

The `clique header 2 [0xae525b65bc7f711bc136f502650039cd6959c3abc28fdf0ebfe2a5f85c92f3b6]` line is split, and the number stored using `storage.get` and `storage.put`:

```js
function OnSignerStartup(info) {}

function ApproveListing() {
  return 'Approve';
}

function ApproveSignData(r) {
  if (r.content_type != 'application/x-clique-header') {
    return 'Reject';
  }
  for (var i = 0; i < r.messages.length; i++) {
    var msg = r.messages[i];
    if (msg.name == 'Clique header' && msg.type == 'clique') {
      var number = parseInt(msg.value.split(' ')[2]);
      var latest = storage.get('lastblock') || 0;
      console.log('number', number, 'latest', latest);
      if (number > latest) {
        storage.put('lastblock', number);
        return 'Approve';
      }
    }
  }
  return 'Reject';
}
```

Running with this ruleset:

```terminal
JS:>  number 45 latest 44
INFO [06-16|22:26:43.023] Op approved
DEBUG[06-16|22:26:44.305] Served account_signData                  reqid=3 duration=1.287465394s
JS:>  number 46 latest 45
INFO [06-16|22:26:44.313] Op approved
DEBUG[06-16|22:26:45.317] Served account_signData                  reqid=4 duration=1.010612774s
```

This might be a bit over-the-top, security-wise, and may cause problems if, for some reason, a clique-deadlock needs to be resolved by rolling back and continuing on a side-chain. It is mainly meant as a demonstration that rules can use Javascript and statefulness to construct very intricate signing logic.

## TLDR quick-version {#tldr-version}

Creation and attestation is a one-off event:

```sh
## Create the rules-file
cat << END > rules.js
function OnSignerStartup(info){}

function ApproveListing(){
  return "Approve"
}

function ApproveSignData(r){
  if (r.content_type == "application/x-clique-header"){
    for(var i = 0; i < r.messages.length; i++){
      var msg = r.messages[i]
      if (msg.name=="Clique header" && msg.type == "clique"){
        return "Approve"
      }
    }
  }
  return "Reject"
}
END
## Attest it, assumes clef master password is in `./clefpw`
clef --keystore ./ddir/keystore \
  --configdir ./clef --chainid 15 \
  --suppress-bootwarn --signersecret ./clefpw \
    attest  `sha256sum rules.js | cut -f1`
```

The normal startup command for `clef`:

```sh
clef --keystore ./ddir/keystore \
    --configdir ./clef --chainid 15  \
    --suppress-bootwarn --signersecret ./clefpw --rules ./rules.js
```

For Geth, the only change is to provide `--signer <path to clef ipc>`.

## Summary {#summary}

Clef can be used as a signer that automatically seals Clique blocks. This is a much more secure option than unlocking accounts using Geth's built-in account manager.
