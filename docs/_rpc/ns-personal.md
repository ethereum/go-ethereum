---
title: personal Namespace
sort_key: C
---

The personal API manages private keys in the key store.

* TOC
{:toc}

### personal_importRawKey

Imports the given unencrypted private key (hex string) into the key store,
encrypting it with the passphrase.

Returns the address of the new account.
 
| Client   | Method invocation                                                 |
| :--------| ----------------------------------------------------------------- |
| Console  | `personal.importRawKey(keydata, passphrase)`                      |
| RPC      | `{"method": "personal_importRawKey", "params": [string, string]}` |

### personal_listAccounts

Returns all the Ethereum account addresses of all keys
in the key store.

| Client   | Method invocation                                   |
| :--------| --------------------------------------------------- |
| Console  | `personal.listAccounts`                             |
| RPC      | `{"method": "personal_listAccounts", "params": []}` |

#### Example

``` javascript
> personal.listAccounts
["0x5e97870f263700f46aa00d967821199b9bc5a120", "0x3d80b31a78c30fc628f20b2c89d7ddbf6e53cedc"]
```

### personal_lockAccount

Removes the private key with given address from memory.
The account can no longer be used to send transactions.
 
| Client   | Method invocation                                        |
| :--------| -------------------------------------------------------- |
| Console  | `personal.lockAccount(address)`                          |
| RPC      | `{"method": "personal_lockAccount", "params": [string]}` |

### personal_newAccount

Generates a new private key and stores it in the key store directory.
The key file is encrypted with the given passphrase.
Returns the address of the new account.

At the geth console, `newAccount` will prompt for a passphrase when 
it is not supplied as the argument.

| Client   | Method invocation                                       |
| :--------| ---------------------------------------------------     |
| Console  | `personal.newAccount()`                                 |
| RPC      | `{"method": "personal_newAccount", "params": [string]}` |

#### Example
 
``` javascript
> personal.newAccount()
Passphrase: 
Repeat passphrase: 
"0x5e97870f263700f46aa00d967821199b9bc5a120"
```

The passphrase can also be supplied as a string.

``` javascript
> personal.newAccount("h4ck3r")
"0x3d80b31a78c30fc628f20b2c89d7ddbf6e53cedc"
```

### personal_unlockAccount

Decrypts the key with the given address from the key store.

Both passphrase and unlock duration are optional when using the JavaScript console.
If the passphrase is not supplied as an argument, the console will prompt for
the passphrase interactively.

The unencrypted key will be held in memory until the unlock duration expires.
If the unlock duration defaults to 300 seconds. An explicit duration
of zero seconds unlocks the key until geth exits.

The account can be used with `eth_sign` and `eth_sendTransaction` while it is unlocked.
 
| Client   | Method invocation                                                          |
| :--------| -------------------------------------------------------------------------- |
| Console  | `personal.unlockAccount(address, passphrase, duration)`                    |
| RPC      | `{"method": "personal_unlockAccount", "params": [string, string, number]}` |

#### Examples

``` javascript
> personal.unlockAccount("0x5e97870f263700f46aa00d967821199b9bc5a120")
Unlock account 0x5e97870f263700f46aa00d967821199b9bc5a120
Passphrase: 
true
```

Supplying the passphrase and unlock duration as arguments:

``` javascript
> personal.unlockAccount("0x5e97870f263700f46aa00d967821199b9bc5a120", "foo", 30)
true
```

If you want to type in the passphrase and stil override the default unlock duration,
pass `null` as the passphrase.

```
> personal.unlockAccount("0x5e97870f263700f46aa00d967821199b9bc5a120", null, 30)
Unlock account 0x5e97870f263700f46aa00d967821199b9bc5a120
Passphrase: 
true
```

### personal_sendTransaction

Validate the given passphrase and submit transaction.

The transaction is the same argument as for `eth_sendTransaction` and contains the `from` address. If the passphrase can be used to decrypt the private key belogging to `tx.from` the transaction is verified, signed and send onto the network. The account is not unlocked globally in the node and cannot be used in other RPC calls.

| Client   | Method invocation                                                |
| :--------| -----------------------------------------------------------------|
| Console  | `personal.sendTransaction(tx, passphrase)`                       |
| RPC      | `{"method": "personal_sendTransaction", "params": [tx, string]}` |

*Note, prior to Geth 1.5, please use `personal_signAndSendTransaction` as that was the
original introductory name and only later renamed to the current final version.*

#### Examples

``` javascript
> var tx = {from: "0x391694e7e0b0cce554cb130d723a9d27458f9298", to: "0xafa3f8684e54059998bc3a7b0d2b0da075154d66", value: web3.toWei(1.23, "ether")}
undefined
> personal.sendTransaction(tx, "passphrase")
0x8474441674cdd47b35b875fd1a530b800b51a5264b9975fb21129eeb8c18582f
```

### personal_sign

The sign method calculates an Ethereum specific signature with:
`sign(keccack256("\x19Ethereum Signed Message:\n" + len(message) + message)))`.

By adding a prefix to the message makes the calculated signature recognisable as an Ethereum specific signature. This prevents misuse where a malicious DApp can sign arbitrary data (e.g. transaction) and use the signature to impersonate the victim.

See ecRecover to verify the signature.

| Client  | Method invocation                                     |
|:--------|-------------------------------------------------------|   
| Console | `personal.sign(message, account, [password])`                |
| RPC     | `{"method": "personal_sign", "params": [message, account, password]}` |


#### Examples

``` javascript
> personal.sign("0xdeadbeaf", "0x9b2055d370f73ec7d8a03e965129118dc8f5bf83", "")
"0xa3f20717a250c2b0b729b7e5becbff67fdaef7e0699da4de7ca5895b02a170a12d887fd3b17bfdce3481f10bea41f45ba9f709d39ce8325427b57afcfc994cee1b"
```

### personal_ecRecover

`ecRecover` returns the address associated with the private key that was used to calculate the signature in `personal_sign`. 

| Client  | Method invocation                                     |
|:--------|-------------------------------------------------------|   
| Console | `personal.ecRecover(message, signature)`                 |
| RPC     | `{"method": "personal_ecRecover", "params": [message, signature]}` |


#### Examples

``` javascript
> personal.sign("0xdeadbeaf", "0x9b2055d370f73ec7d8a03e965129118dc8f5bf83", "")
"0xa3f20717a250c2b0b729b7e5becbff67fdaef7e0699da4de7ca5895b02a170a12d887fd3b17bfdce3481f10bea41f45ba9f709d39ce8325427b57afcfc994cee1b"
> personal.ecRecover("0xdeadbeaf", "0xa3f20717a250c2b0b729b7e5becbff67fdaef7e0699da4de7ca5895b02a170a12d887fd3b17bfdce3481f10bea41f45ba9f709d39ce8325427b57afcfc994cee1b")
"0x9b2055d370f73ec7d8a03e965129118dc8f5bf83"
```
