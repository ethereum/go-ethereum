---
title: Go Account Management
description: Introduction to account management in Go native applications.
---

Geth provides a simple, yet thorough accounts package that includes all the tools developers need to leverage all the security of Geth's crypto implementation in a Go native application. The account management is done client side with all sensitive data held inside the application. This gives the user control over access permissions without relying on any third party.

**Note: Geth's built-in account management is convenient and straightforward to use, but best practice is to use the external tool _Clef_ for key management.**

## Encrypted keystores {#encrypted-keystores}

Access keys to Ethereum accounts should never be stored in plain-text. Instead, they should be stored encrypted so that even if the mobile device is accessed by a malicious third party the keys are still hidden under an additional layer of security. Geth provides a keystore that enables developers to store keys securely. The Geth keystore uses [Scrypt](https://pkg.go.dev/golang.org/x/crypto/scrypt) to store keys that are encoded using the [`secp256k1`](https://www.secg.org/sec2-v2.pdf) elliptic curve. Accounts are stored on disk in the [Web3 Secret Storage](https://github.com/ethereum/wiki/wiki/Web3-Secret-Storage-Definition) format. Developers should be aware of these implementation details
but are not required to deeply understand the cryptographic primitives in order to use the keystore.

One thing that should be understood, though, is that the cryptographic primitives underpinning the keystore can operate in light or standard mode.Light mode is computationally cheaper, while standard mode has extra security. Light mode is appropriate for mobile devices, but developers should be aware that there is a security trade-off.

- standard needs 256MB memory and 1 second processing on a modern CPU to access a key
- light needs 4MB memory and 100 millisecond processing on a modern CPU to access a key

The encrypted keystore is implemented by the [`accounts.Manager`](https://godoc.org/github.com/ethereum/go-ethereum/accounts#Manager) struct from the [`accounts`](https://godoc.org/github.com/ethereum/go-ethereum/accounts) package, which also contains the configuration constants for the _standard_ or _light_ security modes described above. Hence client side account management
simply requires importing the `accounts` package into the application code.

```go
import "github.com/ethereum/go-ethereum/accounts"
import "github.com/ethereum/go-ethereum/accounts/keystore"
import "github.com/ethereum/go-ethereum/common"
```

Afterwards a new encrypted account manager can be created via:

```go
ks := keystore.NewKeyStore("/path/to/keystore", keystore.StandardScryptN, keystore.StandardScryptP)
am := accounts.NewManager(&accounts.Config{InsecureUnlockAllowed: false}, ks)
```

The path to the keystore folder needs to be a location that is writable by the local user but non-readable for other system users, such as inside the user's home directory.

The last two arguments of [`keystore.NewKeyStore`](https://godoc.org/github.com/ethereum/go-ethereum/accounts/keystore#NewKeyStore) are the crypto parameters defining how resource-intensive the keystore encryption should be. The options are [`accounts.StandardScryptN, accounts.StandardScryptP`, `accounts.LightScryptN, accounts.LightScryptP`](https://godoc.org/github.com/ethereum/go-ethereum/accounts#pkg-constants) or custom values (requiring understanding of the underlying cryptography). The _standard_ version is recommended.

## Account lifecycle {#account-lifecycle}

Once an encrypted keystore for Ethereum accounts exists it, it can be used to manage accounts for the entire account lifecycle requirements of a Go native application. This includes the basic functionality of creating new accounts and deleting existing ones as well as updating access credentials, exporting existing accounts, and importing them on other devices.

Although the keystore defines the encryption strength it uses to store accounts, there is no global master password that can grant access to all of them. Rather each account is maintained individually, and stored on disk in its [encrypted format](https://github.com/ethereum/wiki/wiki/Web3-Secret-Storage-Definition) individually, ensuring a much cleaner and stricter separation of credentials.

This individuality means that any operation requiring access to an account will need to provide the necessary authentication credentials for that particular account in the form of a passphrase:

- When creating a new account, the caller must supply a passphrase to encrypt the account with. This passphrase will be required for any subsequent access, the lack of which will forever forfeit using the newly created account.

- When deleting an existing account, the caller must supply a passphrase to verify ownership of the account. This isn't cryptographically necessary, rather a protective measure against accidental loss of accounts.

- When updating an existing account, the caller must supply both current and new passphrases. After completing the operation, the account will not be accessible via the old passphrase anymore.

- When exporting an existing account, the caller must supply both the current passphrase to decrypt the account, as well as an export passphrase to re-encrypt it with before returning the key-file to the user. This is required to allow moving accounts between machines and applications without sharing original credentials.

- When importing a new account, the caller must supply both the encryption passphrase of the key-file being imported, as well as a new passphrase with which to store the account. This is required to allow storing account with different credentials than used for moving them around.

**_Please note, there are no recovery mechanisms for lost passphrases. The cryptographic properties of the encrypted keystore (using the provided parameters) guarantee that account credentials cannot be brute forced in any meaningful time._**

An Ethereum account is implemented by the [`accounts.Account`](https://godoc.org/github.com/ethereum/go-ethereum/accounts#Account) struct from the Geth [accounts](https://godoc.org/github.com/ethereum/go-ethereum/accounts) package. Assuming an instance of an [`accounts.Manager`](https://godoc.org/github.com/ethereum/go-ethereum/accounts#Manager) called `am` exists, all of the described lifecycle operations can be executed with a handful of function calls (error handling omitted).

```go
// Create a new account with the specified encryption passphrase.
newAcc, _ := ks.NewAccount("Creation password")
fmt.Println(newAcc)

// Export the newly created account with a different passphrase. The returned
// data from this method invocation is a JSON encoded, encrypted key-file.
jsonAcc, _ := ks.Export(newAcc, "Creation password", "Export password")

// Update the passphrase on the account created above inside the local keystore.
_ = ks.Update(newAcc, "Creation password", "Update password")

// Delete the account updated above from the local keystore.
_ = ks.Delete(newAcc, "Update password")

// Import back the account we've exported (and then deleted) above with yet
// again a fresh passphrase.
impAcc, _ := ks.Import(jsonAcc, "Export password", "Import password")
```

_Although instances of [`accounts.Account`](https://godoc.org/github.com/ethereum/go-ethereum/accounts#Account) can be used to access various information about specific Ethereum accounts, they do not contain any sensitive data (such as passphrases or private keys), rather they act solely as identifiers for client code and the keystore._

## Signing authorization {#signing-authorization}

Account objects do not hold the sensitive private keys of the associated Ethereum accounts. Account objects are placeholders that identify the cryptographic keys. All operations that require authorization (e.g. transaction signing) are performed by the account manager after granting it access to the private keys.

There are a few different ways to authorize the account manager to execute signing operations, each having its advantages and drawbacks. Since the different methods have wildly different security guarantees, it is essential to be clear on how each works:

- **Single authorization**: The simplest way to sign a transaction via the account manager is to provide the passphrase of the account every time something needs to be signed, which will ephemerally decrypt the private key, execute the signing operation and immediately throw away the decrypted key. The drawbacks are that the passphrase needs to be queried from the user every time, which can become annoying if done frequently or the application needs to keep the passphrase in memory, which can have security consequences if not done properly. Depending on the keystore's configured strength, constantly decrypting keys can result in non-negligible resource requirements.

- **Multiple authorizations**: A more complex way of signing transactions via the account manager is to unlock the account via its passphrase once, and allow the account manager to cache the decrypted private key, enabling all subsequent signing requests to complete without the passphrase. The lifetime of the cached private key may be managed manually (by explicitly locking the account back up) or automatically (by providing a timeout during unlock). This mechanism is useful for scenarios where the user may need to sign many transactions or the application would need to do so without requiring user input. The crucial aspect to remember is that **anyone with access to the account manager can sign transactions while a particular account is unlocked** (e.g. application running untrusted code).

Assuming an instance of an [`accounts.Manager`](https://godoc.org/github.com/ethereum/go-ethereum/accounts#Manager) called `am` exists, a new account can be created to sign transactions using [`NewAccount`](https://godoc.org/github.com/ethereum/go-ethereum/accounts#Manager.NewAccount). Creating transactions is out of scope for this page so instead a random [`common.Hash`](https://godoc.org/github.com/ethereum/go-ethereum/common#Hash) will be signed instead.

For information on creating transactions in Go native applications see the [Go API page](/docs/developers/dapp-developer/native).

```go
// Create a new account to sign transactions with
signer, _ := ks.NewAccount("Signer password")
txHash := common.HexToHash("0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
```

With the boilerplate out of the way, the transaction can be signed using the authorization mechanisms described above:

```go
// Sign a transaction with a single authorization
signature, _ := ks.SignHashWithPassphrase(signer, "Signer password", txHash.Bytes())

// Sign a transaction with multiple manually cancelled authorizations
_ = ks.Unlock(signer, "Signer password")
signature, _ = ks.SignHash(signer, txHash.Bytes())
_ = ks.Lock(signer.Address)

// Sign a transaction with multiple automatically cancelled authorizations
_ = ks.TimedUnlock(signer, "Signer password", time.Second)
signature, _ = ks.SignHash(signer, txHash.Bytes())
```

Note that [`SignWithPassphrase`](https://godoc.org/github.com/ethereum/go-ethereum/accounts#Manager.SignWithPassphrase) takes an [`accounts.Account`](https://godoc.org/github.com/ethereum/go-ethereum/accounts#Account) as the signer, whereas [`Sign`](https://godoc.org/github.com/ethereum/go-ethereum/accounts#Manager.Sign) takes only a [`common.Address`](https://godoc.org/github.com/ethereum/go-ethereum/common#Address). The reason for this is that an [`accounts.Account`](https://godoc.org/github.com/ethereum/go-ethereum/accounts#Account) object may also contain a custom key-path, allowing [`SignWithPassphrase`](https://godoc.org/github.com/ethereum/go-ethereum/accounts#Manager.SignWithPassphrase) to sign using accounts outside of the keystore; however [`Sign`](https://godoc.org/github.com/ethereum/go-ethereum/accounts#Manager.Sign) relies on accounts already unlocked within the keystore, so it cannot specify custom paths.

## Summary {#summary}

Account management is a fundamental pillar of Ethereum development. Geth's Go API provides the tools required to integrate best-practice account security into Go native applications using a simple set of Go functions.
