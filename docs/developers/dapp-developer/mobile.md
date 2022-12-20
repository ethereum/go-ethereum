---
title: Geth for Mobile
description: Introduction to mobile development with Geth
---

Embedding clients into mobile devices is an important part of Ethereum's decentralization vision. This is because being able to verify data, follow the chain and submit transactions without relying on centralized intermediaries is critical for censorship resistant access to the network. Doing so on a mobile device is the most convenient route for many users. This relies on Geth running a [light client](/docs/fundamentals/les) on the mobile
device and exposing an API that developers can use to build mobile apps on top of Geth. This page outlines how to download Geth for mobile and how to get started with managing Ethereum accounts in mobile applications. Ethereum mobile development is relatively nascent, but there is an active developer community. For further information on Geth mobile development visit the #mobile channel in the [Geth discord](https://discord.gg/wQdpS5aA).

## Download and install {#download-and-install}

### Android {#android}

#### Android Studio {#android-studio}

Geth for Mobile bundles can be downloaded directly from [the download page](/downloads) and inserted into a project in Android Studio via `File -> New -> New module... -> Import .JAR/.AAR Package`.

It is also necessary to configure `gradle` to link the mobile library bundle to the application. This can be done by adding a new entry to the `dependencies` section of the `build.gradle` script, pointing it to the module that was just added (named `geth` by default).

```gradle
dependencies {
    // All previous dependencies
    compile project(':geth')
}
```

#### Manual build {#manual-build}

Geth can also be built it locally using a `make` command. This will create an Android archive called `geth.aar` in the `build/bin` folder that can be imported into Android Studio as described above.

```shell
$ make android
[...]
Done building.
Import "build/bin/geth.aar" to use the library.
```

### iOS {#ios}

Geth must be downloaded and built locally for IoS. Building locally is achieved using the `make` command. This will create an iOS XCode framework called `Geth.framework` in the `build/bin` folder that can be imported into XCode as described above.

```bash
$ make ios
[...]
Done building.
Import "build/bin/Geth.framework" to use the library.
```

## Mobile API {#mobile-api}

Similarly to the reusable [Go libraries](/docs/developers/dapp-developer/native-accounts), the mobile wrappers focus on three main usage areas:

- Simplified client side account management
- Remote node interfacing via different transports
- Contract interactions through auto-generated bindings

The Geth mobile API is broadly equivalent to the [Go API](/docs/developers/dapp-developer/native-accounts). The source code can be found in the `mobile` section of Geth's
[GitHub](https://github.com/ethereum/go-ethereum/tree/master/mobile).

## Mobile Account Management {#mobile-account-management}

Best practise for account management is to do it client-side, with all sensitive information self-contained inside the local application. This ensures the developer/user retains fine-grained control over the access permissions for user-data instead of outsourcing security
to a third party.

To support this, Geth provides an accounts library that includes the tools required for secure account management via encrypted keystores and passphrase protected accounts, similarly to running a full Geth node.

### Encrypted keystores {#encrypted-keystores}

Access keys to Ethereum accounts should never be stored in plain-text. Instead, they should be stored encrypted so that even if the mobile device is accessed by a malicious third party the keys are still hidden under an additional layer of security. Geth provides a keystore that enables developers to store keys securely using the [`secp256k1` elliptic curve](https://www.secg.org/sec2-v2.pdf), implemented using [`libsecp256k`](https://github.com/bitcoin-core/secp256k1) and wrapped by [Geth accounts](https://godoc.org/github.com/ethereum/go-ethereum/accounts).

Accounts are stored on disk in the [Web3 Secret Storage](https://github.com/ethereum/wiki/wiki/Web3-Secret-Storage-Definition) format. Developers should be aware of these implementation details but are not required to deeply understand the cryptographic primitives in order to use the keystore.

One thing that should be understood, though, is that the cryptographic primitives underpinning the keystore can operate in _light_ or _standard_ mode. Light mode is computationally cheaper, while standard mode has extra security. Light mode is appropriate for mobile devices, but developers
should be aware that there is a security trade-off.

- _standard_ needs 256MB memory and 1 second processing on a modern CPU to access a key
- _light_ needs 4MB memory and 100 millisecond processing on a modern CPU to access a key

### Keystores on Android (Java) {#keystores-on-android}

The encrypted keystore on Android is implemented by the `KeyStore` class from the `org.ethereum.geth` package. The configuration constants are located in the `Geth` abstract class, similarly from the `org.ethereum.geth` package. Hence to do client side account management on Android, two classes should be imported into the Java code:

```java
import org.ethereum.geth.Geth;
import org.ethereum.geth.KeyStore;
```

Then new encrypted keystore can be created via:

```java
KeyStore ks = new KeyStore("/path/to/keystore", Geth.LightScryptN, Geth.LightScryptP);
```

The keystore should be in a location writable by the local mobile application but on-readable for other installed applications such as inside the app's data directory. If the `KeyStore` is created from within a class extending an Android object, access to the `Context.getFilesDir()` method is probably provided via `this.getFilesDir()`, so the keystore path could be set to `this.getFilesDir() + "/keystore"`.

The last two arguments of the `KeyStore` constructor are the crypto parameters defining how resource-intensive the keystore encryption should be. The choices are `Geth.StandardScryptN, Geth.StandardScryptP`, `Geth.LightScryptN, Geth.LightScryptP` or custom numbers. The _light_ version is recommended.

### Keystores on iOS (Swift 3) {#keystores-on-ios}

The encrypted keystore on iOS is implemented by the `GethKeyStore` class from the `Geth` framework. The configuration constants are located in the same namespace as global variables. Hence to do client side account management on iOS, `Geth` framework should be
imported into the Swift code:

```swift
import Geth
```

Then a new encrypted account manager can be created using:

```swift
let ks = GethNewKeyStore("/path/to/keystore", GethLightScryptN, GethLightScryptP);
```

The keystore folder needs to be in a location writable by the local mobile application but non-readable for other installed applications such as inside the app's document directory. The document directory shopuld be retrievable using
`let datadir = NSSearchPathForDirectoriesInDomains(.documentDirectory, .userDomainMask, true)[0]`, so the keystore path could be `datadir + "/keystore"`.

The last two arguments of the `GethNewKeyStore` factory method are the crypto parameters defining how resource-intensive the keystore encryption should be. The choices are `GethStandardScryptN, GethStandardScryptP`, `GethLightScryptN, GethLightScryptP` or custom numbers. The _light_ version is recommended.

### Account lifecycle {#account-lifecycle}

The encyrpted keystore can be used for the entire account lifecycle requirements of a mobile application. This includes the basic functionality of creating new accounts and deleting existing ones as well as more advanced functions like updating access credentials and account
import/export.

Although the keystore defines the encryption strength it uses to store accounts, there is no global master password that can grant access to all of them. Rather each account is maintained individually, and stored on disk in its [encrypted format][secstore] individually, ensuring a much cleaner and stricter separation of credentials.

This individuality means that any operation requiring access to an account will need to provide the necessary authentication credentials for that particular account in the form of a passphrase:

- When creating a new account, the caller must supply a passphrase to encrypt the account with. This passphrase will be required for any subsequent access.
- When deleting an existing account, the caller must supply a passphrase to verify ownership of the account. This isn't cryptographically necessary, rather a protective measure against accidental loss of accounts.
- When updating an existing account, the caller must supply both current and new passphrases. After completing the operation, the account will not be accessible via the old passphrase.
- When exporting an existing account, the caller must supply both the current passphrase to decrypt the account, as well as an export passphrase to re-encrypt it with before returning the key-file to the user. This is required to allow moving accounts between devices without sharing original credentials.
- When importing a new account, the caller must supply both the encryption passphrase of the key-file being imported, as well as a new passphrase with which to store the account. This is required to allow storing account with different credentials than used for moving them around.

_Please note, there is no recovery mechanisms for losing the passphrases. The cryptographic properties of the encrypted keystore (if using the provided parameters) guarantee that account credentials cannot be brute forced in any meaningful time._

### Accounts on Android (Java) {#accounts-on-android}

An Ethereum account on Android is implemented by the `Account` class from the `org.ethereum.geth` package. Assuming an instance of a `KeyStore` called `ks` exists, all of the described lifecycle operations can be executed with a handful of function calls:

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

Although instances of `Account` can be used to access various information about specific Ethereum accounts, they do not contain any sensitive data (such as passphrases or private keys), rather they act solely as identifiers for client code and the keystore.

### Accounts on iOS (Swift 3) {#accounts-on-ios}

An Ethereum account on iOS is implemented by the `GethAccount` class from the `Geth` framework. Assuming an instance of a `GethKeyStore` called `ks` exists, all of the described lifecycle operations can be executed with a handful of function calls:

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

Although instances of `GethAccount` can be used to access various information about specific Ethereum accounts, they do not contain any sensitive data (such as passphrases or private keys), rather they act solely as identifiers for client code and the keystore.

## Signing authorization {#signing-authorization}

As mentioned above, account objects do not hold the sensitive private keys of the associated Ethereum accounts - they are merely placeholders to identify the cryptographic keys with. All operations that require authorization (e.g. transaction signing) are performed by the account manager after granting it access to the private keys.

There are a few different ways one can authorize the account manager to execute signing operations. Since the different methods have very different security guarantees, it is essential to be clear on how each works:

- **Single authorization**: The simplest way to sign a transaction via the keystore is to provide the passphrase of the account every time something needs to be signed, which will ephemerally decrypt the private key, execute the signing operation and immediately throw away the decrypted key. The drawbacks are that the passphrase needs to be queried from the user every time, which can become annoying if done frequently; or the application needs to keep the passphrase in memory, which can have security consequences if not done properly; and depending on the keystore's configured strength, constantly decrypting keys can result in non-negligible resource requirements.

- **Multiple authorizations**: A more complex way of signing transactions via the keystore is to unlock the account via its passphrase once, and allow the account manager to cache the decrypted private key, enabling all subsequent signing requests tocomplete without the passphrase. The lifetime of the cached private key may be managed manually (by explicitly locking the account back up) or automatically (by providing a timeout during unlock). This mechanism is useful for scenarios where the user may need to sign many transactions or the application would need to do so without requiring user input. The crucial aspect to remember is that **anyone with access to the account manager can sign transactions while a particular account is unlocked** (e.g. device left unattended; application running untrusted code).

### Signing on Android (Java) {#signing-on-android}

Assuming an instance of a `KeyStore` called `ks` exists, a new account to sign transactions can be created using its `newAccount` method. For this demonstation a hard-coded example transaction is created to sign:

```java
// Create a new account to sign transactions with
Account signer = ks.newAccount("Signer password");
Transaction tx = new Transaction(
    1, new Address("0x0000000000000000000000000000000000000000"),
    new BigInt(0), new BigInt(0), new BigInt(1), null); // Random empty transaction
BigInt chain = new BigInt(1); // Chain identifier of the main net
```

The transaction `tx` can be signed using the authorization mechanisms described above:

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

### Signing on iOS (Swift 3) {#signing-on-ios}

Assuming an instance of a `GethKeyStore` called `ks` exists, a new account can be created to sign transactions with its `newAccount` method. For
this demonstation a hard-coded example transaction is created to sign:

```swift
// Create a new account to sign transactions with
var error: NSError?
let signer = try! ks?.newAccount("Signer password")

let to    = GethNewAddressFromHex("0x0000000000000000000000000000000000000000", &error)
let tx    = GethNewTransaction(1, to, GethNewBigInt(0), GethNewBigInt(0), GethNewBigInt(0), nil) // Random empty transaction
let chain = GethNewBigInt(1) // Chain identifier of the main net
```

The transaction `tx` can now be signed using the authorization methods described above:

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

## Summary {#summary}

This page introduced Geth for mobile. In addition to download and installation instructions, basic account management was demonstrated for mobile applications on iOS and Android.
