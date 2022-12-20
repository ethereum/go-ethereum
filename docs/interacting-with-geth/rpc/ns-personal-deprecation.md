---
title: Personal namespace deprecation notes
description: Alternatives to the methods in the deprecated personal namespace
---

The JSON-RPC API's `personal` namespace has historically been used to manage accounts and sign transactions and data over RPC. However, it is being deprecated in favour of using [Clef](/docs/tools/clef/introduction) as an external signer and account manager. One of the major changes is moving away from indiscriminate locking and unlocking of accounts and instead using Clef to explicitly approve or deny specific actions. This page shows the suggested replacement for each method in `personal`.

## Methods without replacements

### personal_unlockAccount

There is no need for a direct replacement for `personal_unlockAccount`. Using Clef to manually approve actions or to attest custom rulesets is a much more secure way to interact with accounts without needing to indiscriminately unlock accounts.

### personal_lockAccount

There is no need for a direct replacement for `personal_lockAccount` because account locking/unlocking is replaced by Clef's approve/deny logic. This is a more secure way to interact with accounts.

### personal.unpair

Unpair deletes a pairing between some specific types of smartcard wallet and Geth. There is not yet an equivalent method in Clef.

### personal_initializeWallet

InitializeWallet is for initializing some specific types of smartcard wallet at a provided URL. There is not yet a corresponding method in Clef.

## Methods with replacements

### personal_listAccounts

`personal_listAccounts` displays the addresses of all accounts in the keystore. It is identical to `eth.accounts`. Calling `eth.accounts` requires manual approval in Clef (unless a rule for it has been attested). There is also Clef's `list-accounts` command that can be called from the terminal.

Examples:

```sh
# eth_accounts using curl
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_accounts","params":[],"id":1}'
```

```js
// eth_accounts in Geth's JS console
eth.accounts;
```

```sh
# clef list-accounts in the terminal
clef list-accounts
```

### personal_deriveAccount

`personal_deriveAccount` requests a hardware wallet to derive a new account, optionally pinning it for later use. This method is identical to `clef_deriveAccount`. The Clef method is not externally exposed so it must be called via a UI.

### personal.ecRecover

`personal_ecRecover` returns the address for the account that was used to create a signature. An equivalent method, `account_ecRecover` is available on the Clef external API.

Example call:

```sh
curl --data '{"id": 4, "jsonrpc": "2.0", "method": "account_ecRecover","params": ["0xaabbccdd",     "0x5b6693f153b48ec1c706ba4169960386dbaa6903e249cc79a8e6ddc434451d417e1e57327872c7f538beeb323c300afa9999a3d4a5de6caf3be0d5ef832b67ef1c"]}' -X POST localhost:8550
```

### personal_importRawKey

`personal.importRawKey` was used to create a new account in the keystore from a raw private key. Clef has an equivalent method that can be invoked in the terminal using:

```sh
clef importraw <private-key-as-hex-string>
```

### personal_listWallets

As opposed to `listAccounts`, this method lists full details, including usb path or keystore-file paths. The equivalent method is `clef_listWallets`. This method can be called from the terminal using:

```sh
clef list-wallets
```

### personal_newAccount

`personal_newAccount` was used to create a new accoutn and save it in the keystore. Clef has an equivalent method, `account_new`. It can be accessed on the terminal using an http request or using a Clef command:

Example call (curl):

```sh
curl --data '{"id": 1, "jsonrpc": "2.0", "method": "account_new", "params": []}' -X POST localhost:8550
```

Example call (Clef command):

```sh
clef newaccount
```

Both require manual approval in Clef unless a custom ruleset is in place.

### personal_openWallet

`personal_OpenWallet` initiates a hardware wallet opening procedure by establishing a USB connection and then attempting to authenticate via the provided passphrase. Note, the method may return an extra challenge requiring a second open (e.g. the Trezor PIN matrix challenge). `personal_openWallet` is identical to `clef_openWallet`. The Clef method is not externally eposed, meaning it must be called via a UI.

### personal_sendTransaction

`personal_sendTransaction` ws used to sign and submit a transaction. This can be done using `eth_sendTransaction`, requiring manual approval in Clef.

Example call (Javascript console):

```js
// this command requires 2x approval in Clef because it loads account data via eth.accounts[0]
// and eth.accounts[1]
var tx = { from: eth.accounts[0], to: eth.accounts[1], value: web3.toWei(0.1, 'ether') };

// then send the transaction
eth.sendTransaction(tx);
```

Example call (terminal)

```sh
curl --data '{"id":1,"jsonrpc":"2.0", "method":"eth_sendTransaction","params":[{"from": "0xE70CAD05D0D54Ae3C9Fe5442f901E0433f9bd14B", "to":"0x4FDc03d09Ffca5Bba3138149E29D85C8A9E2Ac42", "gas":"21000","gasPrice":"20000000000", "nonce":"94"}]}' -H "Content-Type: application/json" -X POST localhost:8545
```

### personal_sign

The sign method calculates an Ethereum specific signature with `sign(keccak256("\x19Ethereum Signed Message:\n" + len(message) + message))`. Adding a prefix to the message makes the calculated signature recognisable as an Ethereum specific signature. This prevents misuse where a malicious DApp can sign arbitrary data (e.g. transaction) and use the signature to impersonate the victim.

`personal.sign` is equivalent to Clef's `account_signData`. It returns the calculated signature.

Example call:

```sh
curl --data {"id": 3, "jsonrpc": "2.0", "method": "account_signData", "params": ["data/plain", "0x1923f626bb8dc025849e00f99c25fe2b2f7fb0db","0xaabbccdd"]} -X POST localhost:8550
```

Clef also has `account_signTypedData` that signs data structured according to [EIP-712](https://github.com/ethereum/EIPs/blob/master/EIPS/eip-712.md) and returns the signature.

Example call (use the following as a template for `<data>` in `curl --data <data> -X POST localhost:8550 -H "Content-Type: application/json"`)

```sh
{
  "id": 68,
  "jsonrpc": "2.0",
  "method": "account_signTypedData",
  "params": [
    "0xcd2a3d9f938e13cd947ec05abc7fe734df8dd826",
    {
      "types": {
        "EIP712Domain": [
          {
            "name": "name",
            "type": "string"
          },
          {
            "name": "version",
            "type": "string"
          },
          {
            "name": "chainId",
            "type": "uint256"
          },
          {
            "name": "verifyingContract",
            "type": "address"
          }
        ],
        "Person": [
          {
            "name": "name",
            "type": "string"
          },
          {
            "name": "wallet",
            "type": "address"
          }
        ],
        "Mail": [
          {
            "name": "from",
            "type": "Person"
          },
          {
            "name": "to",
            "type": "Person"
          },
          {
            "name": "contents",
            "type": "string"
          }
        ]
      },
      "primaryType": "Mail",
      "domain": {
        "name": "Ether Mail",
        "version": "1",
        "chainId": 1,
        "verifyingContract": "0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC"
      },
      "message": {
        "from": {
          "name": "Cow",
          "wallet": "0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826"
        },
        "to": {
          "name": "Bob",
          "wallet": "0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB"
        },
        "contents": "Hello, Bob!"
      }
    }
  ]
}
```

### personal_signTransaction

`personal_signTransaction` was used to create and sign a transaction from the given arguments. The transaction was returned in RLP-form, not broadcast to other nodes. The equivalent method is Clef's `account_signTransaction` from the external API. The arguments are a transaction object (`{"from": , "to": , "gas": , "maxPriorityFeePerGas": , "MaxFeePerGas": , "value": , "data": , "nonce": }`)) and an optional method signature that enables Clef to decode the calldata and show the user the methods, arguments and values being sent.

Example call (terminal):

```sh
curl --data '{"id": 2, "jsonrpc": "2.0", "method": "account_signTransaction", "params": [{"from": "0x1923f626bb8dc025849e00f99c25fe2b2f7fb0db", "gas": "0x55555","gasPrice": "0x1234", "input": "0xabcd", "nonce": "0x0", "to": "0x07a565b7ed7d7a678680a4c162885bedbb695fe0", "value": "0x1234"}]}' -X POST -H "Content-Type: application/json" localhost:8550
```
