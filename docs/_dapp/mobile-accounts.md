---
title: Mobile Account Management
---

To provide Ethereum integration for your mobile applications, the very first thing you
should be interested in doing is account management.

Although all current leading Ethereum implementations provide account management built in,
it is ill advised to keep accounts in any location that is shared between multiple
applications and/or multiple people. The same way you do not entrust your ISP (who is
after all your gateway into the internet) with your login credentials; you should not
entrust an Ethereum node (who is your gateway into the Ethereum network) with your
credentials either.

The proper way to handle user accounts in your mobile applications is to do client side
account management, everything self-contained within your own application. This way you
can ensure as fine grained (or as coarse) access permissions to the sensitive data as
deemed necessary, without relying on any third party application's functionality and/or
vulnerabilities.

To support this, `go-ethereum` provides a simple, yet thorough accounts library that gives
you all the tools to do properly secured account management via encrypted keystores and
passphrase protected accounts. You can leverage all the security of the `go-ethereum`
crypto implementation while at the same time running everything in your own application.

## Encrypted keystores

Although handling your users' accounts locally on their own mobile device does provide
certain security guarantees, access keys to Ethereum accounts should never lay around in
clear-text form. As such, we provide an encrypted keystore that provides the proper
security guarantees for you without requiring a thorough understanding from your part of
the associated cryptographic primitives.

The important thing to know when using the encrypted keystore is that the cryptographic
primitives used within can operate either in *standard* or *light* mode. The former
provides a higher level of security at the cost of increased computational burden and
resource consumption:

 * *standard* needs 256MB memory and 1 second processing on a modern CPU to access a key
 * *light* needs 4MB memory and 100 millisecond processing on a modern CPU to access a key

As such, *light* is more suitable for mobile applications, but you should be aware of the
trade-offs nonetheless.

*For those interested in the cryptographic and/or implementation details, the key-store
uses the `secp256k1` elliptic curve as defined in the [Standards for Efficient
Cryptography](sec2), implemented by the [`libsecp256k`](secp256k1) library and wrapped by
[`github.com/ethereum/go-ethereum/accounts`](accounts-go). Accounts are stored on disk in
the [Web3 Secret Storage](secstore) format.*

[sec2]: http://www.secg.org/sec2-v2.pdf
[accounts-go]: https://godoc.org/github.com/ethereum/go-ethereum/accounts
[secp256k1]: https://github.com/bitcoin-core/secp256k1
[secstore]: https://github.com/ethereum/wiki/wiki/Web3-Secret-Storage-Definition

### Keystores on Android (Java)

The encrypted keystore on Android is implemented by the `KeyStore` class from the
`org.ethereum.geth` package. The configuration constants (for the *standard* or *light*
security modes described above) are located in the `Geth` abstract class, similarly from
the `org.ethereum.geth` package. Hence to do client side account management on Android,
you'll need to import two classes into your Java code:

```java
import org.ethereum.geth.Geth;
import org.ethereum.geth.KeyStore;
```

Afterwards you can create a new encrypted keystore via:

```java
KeyStore ks = new KeyStore("/path/to/keystore", Geth.LightScryptN, Geth.LightScryptP);
```

The path to the keystore folder needs to be a location that is writable by the local
mobile application but non-readable for other installed applications (for security reasons
obviously), so we'd recommend placing it inside your app's data directory. If you are
creating the `KeyStore` from within a class extending an Android object, you will most
probably have access to the `Context.getFilesDir()` method via `this.getFilesDir()`, so
you could set the keystore path to `this.getFilesDir() + "/keystore"`.

The last two arguments of the `KeyStore` constructor are the crypto parameters defining
how resource-intensive the keystore encryption should be. You can choose between
`Geth.StandardScryptN, Geth.StandardScryptP`, `Geth.LightScryptN, Geth.LightScryptP` or
specify your own numbers (please make sure you understand the underlying cryptography for
this). We recommend using the *light* version.

### Keystores on iOS (Swift 3)

The encrypted keystore on iOS is implemented by the `GethKeyStore` class from the `Geth`
framework. The configuration constants (for the *standard* or *light* security modes
described above) are located in the same namespace as global variables. Hence to do client
side account management on iOS, you'll need to import the framework into your Swift code:

```swift
import Geth
```

Afterwards you can create a new encrypted account manager via:

```swift
let ks = GethNewKeyStore("/path/to/keystore", GethLightScryptN, GethLightScryptP);
```

The path to the keystore folder needs to be a location that is writable by the local
mobile application but non-readable for other installed applications (for security reasons
obviously), so we'd recommend placing it inside your app's document directory. You should
be able to retrieve the document directory via `let datadir =
NSSearchPathForDirectoriesInDomains(.documentDirectory, .userDomainMask, true)[0]`, so you
could set the keystore path to `datadir + "/keystore"`.

The last two arguments of the `GethNewKeyStore` factory method are the crypto parameters
defining how resource-intensive the keystore encryption should be. You can choose between
`GethStandardScryptN, GethStandardScryptP`, `GethLightScryptN, GethLightScryptP` or
specify your own numbers (please make sure you understand the underlying cryptography for
this). We recommend using the *light* version.

## Account lifecycle

Having created an encrypted keystore for your Ethereum accounts, you can use this for the
entire account lifecycle requirements of your mobile application. This includes the basic
functionality of creating new accounts and deleting existing ones; as well as the more
advanced functionality of updating access credentials, exporting existing accounts, and
importing them on another device.

Although the keystore defines the encryption strength it uses to store your accounts,
there is no global master password that can grant access to all of them. Rather each
account is maintained individually, and stored on disk in its [encrypted
format](https://github.com/ethereum/wiki/wiki/Web3-Secret-Storage-Definition)
individually, ensuring a much cleaner and stricter separation of credentials.

This individuality however means that any operation requiring access to an account will
need to provide the necessary authentication credentials for that particular account in
the form of a passphrase:

 * When creating a new account, the caller must supply a passphrase to encrypt the account
   with. This passphrase will be required for any subsequent access, the lack of which
   will forever forfeit using the newly created account.
 * When deleting an existing account, the caller must supply a passphrase to verify
   ownership of the account. This isn't cryptographically necessary, rather a protective
   measure against accidental loss of accounts.
 * When updating an existing account, the caller must supply both current and new
   passphrases. After completing the operation, the account will not be accessible via the
   old passphrase any more.
 * When exporting an existing account, the caller must supply both the current passphrase
   to decrypt the account, as well as an export passphrase to re-encrypt it with before
   returning the key-file to the user. This is required to allow moving accounts between
   devices without sharing original credentials.
 * When importing a new account, the caller must supply both the encryption passphrase of
   the key-file being imported, as well as a new passhprase with which to store the
   account. This is required to allow storing account with different credentials than used
   for moving them around.

*Please note, there is no recovery mechanisms for losing the passphrases. The
cryptographic properties of the encrypted keystore (if using the provided parameters)
guarantee that account credentials cannot be brute forced in any meaningful time.*

### Accounts on Android (Java)

An Ethereum account on Android is implemented by the `Account` class from the
`org.ethereum.geth` package. Assuming we already have an instance of a `KeyStore` called
`ks` from the previous section, we can easily execute all of the described lifecycle
operations with a handful of function calls.

```java
// Create a new account with the specified encryption passphrase.
Account newAcc = ksm.newAccount("Creation password");

// Export the newly created account with a different passphrase. The returned
// data from this method invocation is a JSON encoded, encrypted key-file.
byte[] jsonAcc = ks.exportKey(newAcc, "Creation password", "Export password");

// Update the passphrase on the account created above inside the local keystore.
ks.updateAccount(newAcc, "Creation password", "Update password");

// Delete the account updated above from the local keystore.
ks.deleteAccount(newAcc, "Update password");

// Import back the account we've exported (and then deleted) above with yet
// again a fresh passphrase.
Account impAcc = ks.importKey(jsonAcc, "Export password", "Import password");
```

*Although instances of `Account` can be used to access various information about specific
Ethereum accounts, they do not contain any sensitive data (such as passphrases or private
keys), rather act solely as identifiers for client code and the keystore.*

### Accounts on iOS (Swift 3)

An Ethereum account on iOS is implemented by the `GethAccount` class from the `Geth`
framework. Assuming we already have an instance of a `GethKeyStore` called `ks` from the
previous section, we can easily execute all of the described lifecycle operations with a
handful of function calls.

```swift
// Create a new account with the specified encryption passphrase.
let newAcc = try! ks?.newAccount("Creation password")

// Export the newly created account with a different passphrase. The returned
// data from this method invocation is a JSON encoded, encrypted key-file.
let jsonKey = try! ks?.exportKey(newAcc!, passphrase: "Creation password", newPassphrase: "Export password")

// Update the passphrase on the account created above inside the local keystore.
try! ks?.update(newAcc, passphrase: "Creation password", newPassphrase: "Update password")

// Delete the account updated above from the local keystore.
try! ks?.delete(newAcc, passphrase: "Update password")

// Import back the account we've exported (and then deleted) above with yet
// again a fresh passphrase.
let impAcc  = try! ks?.importKey(jsonKey, passphrase: "Export password", newPassphrase: "Import password")
```

*Although instances of `GethAccount` can be used to access various information about
specific Ethereum accounts, they do not contain any sensitive data (such as passphrases or
private keys), rather act solely as identifiers for client code and the keystore.*

## Signing authorization

As mentioned above, account objects do not hold the sensitive private keys of the
associated Ethereum accounts, but are merely placeholders to identify the cryptographic
keys with. All operations that require authorization (e.g. transaction signing) are
performed by the account manager after granting it access to the private keys.

There are a few different ways one can authorize the account manager to execute signing
operations, each having its advantages and drawbacks. Since the different methods have
wildly different security guarantees, it is essential to be clear on how each works:

 * **Single authorization**: The simplest way to sign a transaction via the keystore is to
   provide the passphrase of the account every time something needs to be signed, which
   will ephemerally decrypt the private key, execute the signing operation and immediately
   throw away the decrypted key. The drawbacks are that the passphrase needs to be queried
   from the user every time, which can become annoying if done frequently; or the
   application needs to keep the passphrase in memory, which can have security
   consequences if not done properly; and depending on the keystore's configured strength,
   constantly decrypting keys can result in non-negligible resource requirements.
 * **Multiple authorizations**: A more complex way of signing transactions via the
   keystore is to unlock the account via its passphrase once, and allow the account
   manager to cache the decrypted private key, enabling all subsequent signing requests to
   complete without the passphrase. The lifetime of the cached private key may be managed
   manually (by explicitly locking the account back up) or automatically (by providing a
   timeout during unlock). This mechanism is useful for scenarios where the user may need
   to sign many transactions or the application would need to do so without requiring user
   input. The crucial aspect to remember is that **anyone with access to the account
   manager can sign transactions while a particular account is unlocked** (e.g. device
   left unattended; application running untrusted code).

*Note, creating transactions is out of scope here, so the remainder of this section will
assume we already have a transaction to sign, and will focus only on creating an
authorized version of it. Creating an actually meaningful transaction will be covered
later.*

### Signing on Android (Java)

Assuming we already have an instance of a `KeyStore` called `ks` from the previous
sections, we can create a new account to sign transactions with via it's already
demonstrated `newAccount` method; and to avoid going into transaction creation for now, we
can hard-code a random transaction to sign instead.

```java
// Create a new account to sign transactions with
Account signer = ks.newAccount("Signer password");
Transaction tx = new Transaction(
	1, new Address("0x0000000000000000000000000000000000000000"),
	new BigInt(0), new BigInt(0), new BigInt(1), null); // Random empty transaction
BigInt chain = new BigInt(1); // Chain identifier of the main net
```

With the boilerplate out of the way, we can now sign transaction using the authorization
mechanisms described above:

```java
// Sign a transaction with a single authorization
Transaction signed = ks.signTxPassphrase(signer, "Signer password", tx, chain);

// Sign a transaction with multiple manually cancelled authorizations
ks.unlock(signer, "Signer password");
signed = ks.signTx(signer, tx, chain);
ks.lock(signer.getAddress());

// Sign a transaction with multiple automatically cancelled authorizations
ks.timedUnlock(signer, "Signer password", 1000000000);
signed = ks.signTx(signer, tx, chain);
```

### Signing on iOS (Swift 3)

Assuming we already have an instance of a `GethKeyStore` called `ks` from the previous
sections, we can create a new account to sign transactions with via it's already
demonstrated `newAccount` method; and to avoid going into transaction creation for now, we
can hard-code a random transaction to sign instead.

```swift
// Create a new account to sign transactions with
var error: NSError?
let signer = try! ks?.newAccount("Signer password")
        
let to    = GethNewAddressFromHex("0x0000000000000000000000000000000000000000", &error)
let tx    = GethNewTransaction(1, to, GethNewBigInt(0), GethNewBigInt(0), GethNewBigInt(0), nil) // Random empty transaction
let chain = GethNewBigInt(1) // Chain identifier of the main net
```

*Note, although Swift usually rewrites `NSError` returns to throws, this particular
instance seems to have been missed for some reason (possibly due to it being a
constructor). It will be fixed in a later version of the iOS bindings when the appropriate
fixed are implemented upstream in the `gomobile` project.*

With the boilerplate out of the way, we can now sign transaction using the authorization
methods described above:

```swift
// Sign a transaction with a single authorization
var signed = try! ks?.signTxPassphrase(signer, passphrase: "Signer password", tx: tx, chainID: chain)

// Sign a transaction with multiple manually cancelled authorizations
try! ks?.unlock(signer, passphrase: "Signer password")
signed = try! ks?.signTx(signer, tx: tx, chainID: chain)
try! ks?.lock(signer?.getAddress())

// Sign a transaction with multiple automatically cancelled authorizations
try! ks?.timedUnlock(signer, passphrase: "Signer password", timeout: 1000000000)
signed = try! ks?.signTx(signer, tx: tx, chainID: chain)
```

