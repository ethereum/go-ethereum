---
title: Go Account Management
---

To provide Ethereum integration for your native applications, the very first thing you
should be interested in doing is account management.

Although all current leading Ethereum implementations provide account management built in,
it is ill advised to keep accounts in any location that is shared between multiple
applications and/or multiple people. The same way you do not entrust your ISP (who is
after all your gateway into the internet) with your login credentials; you should not
entrust an Ethereum node (who is your gateway into the Ethereum network) with your
credentials either.

The proper way to handle user accounts in your native applications is to do client side
account management, everything self-contained within your own application. This way you
can ensure as fine grained (or as coarse) access permissions to the sensitive data as
deemed necessary, without relying on any third party application's functionality and/or
vulnerabilities.

To support this, `go-ethereum` provides a simple, yet thorough accounts package that gives
you all the tools to do properly secured account management via encrypted keystores and
passphrase protected accounts. You can leverage all the security of the `go-ethereum`
crypto implementation while at the same time running everything in your own application.

## Encrypted keystores

Although handling accounts locally to an application does provide certain security
guarantees, access keys to Ethereum accounts should never lay around in clear-text form.
As such, we provide an encrypted keystore that provides the proper security guarantees for
you without requiring a thorough understanding from your part of the associated
cryptographic primitives.

The important thing to know when using the encrypted keystore is that the cryptographic
primitives used within can operate either in *standard* or *light* mode. The former
provides a higher level of security at the cost of increased computational burden and
resource consumption:

 * *standard* needs 256MB memory and 1 second processing on a modern CPU to access a key
 * *light* needs 4MB memory and 100 millisecond processing on a modern CPU to access a key

As such, *standard* is more suitable for native applications, but you should be aware of
the trade-offs nonetheless in case you you're targeting more resource constrained
environments.

*For those interested in the cryptographic and/or implementation details, the key-store
uses the `secp256k1` elliptic curve as defined in the [Standards for Efficient
Cryptography](sec2), implemented by the [`libsecp256k`](secp256k1) library and wrapped by
[`github.com/ethereum/go-ethereum/accounts`](accounts-go). Accounts are stored on disk in
the [Web3 Secret Storage](secstore) format.*

[sec2]: http://www.secg.org/sec2-v2.pdf
[accounts-go]: https://godoc.org/github.com/ethereum/go-ethereum/accounts
[secp256k1]: https://github.com/bitcoin-core/secp256k1
[secstore]: https://github.com/ethereum/wiki/wiki/Web3-Secret-Storage-Definition

### Keystores from Go

The encrypted keystore is implemented by the
[`accounts.Manager`](https://godoc.org/github.com/ethereum/go-ethereum/accounts#Manager)
struct from the
[`github.com/ethereum/go-ethereum/accounts`](https://godoc.org/github.com/ethereum/go-ethereum/accounts)
package, which also contains the configuration constants for the *standard* or *light*
security modes described above. Hence to do client side account management from Go, you'll
need to import only the `accounts` package into your code:

```go
import "github.com/ethereum/go-ethereum/accounts"
```

Afterwards you can create a new encrypted account manager via:

```go
am := accounts.NewManager("/path/to/keystore", accounts.StandardScryptN, accounts.StandardScryptP);
```

The path to the keystore folder needs to be a location that is writable by the local user
but non-readable for other system users (for security reasons obviously), so we'd
recommend placing it either inside your user's home directory or even more locked down for
backend applications.

The last two arguments of
[`accounts.NewManager`](https://godoc.org/github.com/ethereum/go-ethereum/accounts#NewManager)
are the crypto parameters defining how resource-intensive the keystore encryption should
be. You can choose between [`accounts.StandardScryptN, accounts.StandardScryptP`,
`accounts.LightScryptN,
accounts.LightScryptP`](https://godoc.org/github.com/ethereum/go-ethereum/accounts#pkg-constants)
or specify your own numbers (please make sure you understand the underlying cryptography
for this). We recommend using the *standard* version.

## Account lifecycle

Having created an encrypted keystore for your Ethereum accounts, you can use this account
manager for the entire account lifecycle requirements of your native application. This
includes the basic functionality of creating new accounts and deleting existing ones; as
well as the more advanced functionality of updating access credentials, exporting existing
accounts, and importing them on another device.

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
   machines and applications without sharing original credentials.
 * When importing a new account, the caller must supply both the encryption passphrase of
   the key-file being imported, as well as a new passhprase with which to store the
   account. This is required to allow storing account with different credentials than used
   for moving them around.

*Please note, there is no recovery mechanisms for losing the passphrases. The
cryptographic properties of the encrypted keystore (if using the provided parameters)
guarantee that account credentials cannot be brute forced in any meaningful time.*

### Accounts from Go

An Ethereum account is implemented by the
[`accounts.Account`](https://godoc.org/github.com/ethereum/go-ethereum/accounts#Account)
struct from the
[`github.com/ethereum/go-ethereum/accounts`](https://godoc.org/github.com/ethereum/go-ethereum/accounts)
package. Assuming we already have an instance of an
[`accounts.Manager`](https://godoc.org/github.com/ethereum/go-ethereum/accounts#Manager)
called `am` from the previous section, we can easily execute all of the described
lifecycle operations with a handful of function calls (error handling omitted).

```go
// Create a new account with the specified encryption passphrase.
newAcc, _ := am.NewAccount("Creation password");

// Export the newly created account with a different passphrase. The returned
// data from this method invocation is a JSON encoded, encrypted key-file.
jsonAcc, _ := am.Export(newAcc, "Creation password", "Export password");

// Update the passphrase on the account created above inside the local keystore.
am.Update(newAcc, "Creation password", "Update password");

// Delete the account updated above from the local keystore.
am.Delete(newAcc, "Update password");

// Import back the account we've exported (and then deleted) above with yet
// again a fresh passphrase.
impAcc, _ := am.Import(jsonAcc, "Export password", "Import password");
```

*Although instances of
[`accounts.Account`](https://godoc.org/github.com/ethereum/go-ethereum/accounts#Account)
can be used to access various information about specific Ethereum accounts, they do not
contain any sensitive data (such as passphrases or private keys), rather act solely as
identifiers for client code and the keystore.*

## Signing authorization

As mentioned above, account objects do not hold the sensitive private keys of the
associated Ethereum accounts, but are merely placeholders to identify the cryptographic
keys with. All operations that require authorization (e.g. transaction signing) are
performed by the account manager after granting it access to the private keys.

There are a few different ways one can authorize the account manager to execute signing
operations, each having its advantages and drawbacks. Since the different methods have
wildly different security guarantees, it is essential to be clear on how each works:

 * **Single authorization**: The simplest way to sign a transaction via the account
   manager is to provide the passphrase of the account every time something needs to be
   signed, which will ephemerally decrypt the private key, execute the signing operation
   and immediately throw away the decrypted key. The drawbacks are that the passphrase
   needs to be queried from the user every time, which can become annoying if done
   frequently; or the application needs to keep the passphrase in memory, which can have
   security consequences if not done properly; and depending on the keystore's configured
   strength, constantly decrypting keys can result in non-negligible resource
   requirements.
 * **Multiple authorizations**: A more complex way of signing transactions via the account
   manager is to unlock the account via its passphrase once, and allow the account manager
   to cache the decrypted private key, enabling all subsequent signing requests to
   complete without the passphrase. The lifetime of the cached private key may be managed
   manually (by explicitly locking the account back up) or automatically (by providing a
   timeout during unlock). This mechanism is useful for scenarios where the user may need
   to sign many transactions or the application would need to do so without requiring user
   input. The crucial aspect to remember is that **anyone with access to the account
   manager can sign transactions while a particular account is unlocked** (e.g.
   application running untrusted code).

*Note, creating transactions is out of scope here, so the remainder of this section will
assume we already have a transaction hash to sign, and will focus only on creating a
cryptographic signature authorizing it. Creating an actual transaction and injecting the
authorization signature into it will be covered later.*

### Signing from Go

Assuming we already have an instance of an
[`accounts.Manager`](https://godoc.org/github.com/ethereum/go-ethereum/accounts#Manager)
called `am` from the previous sections, we can create a new account to sign transactions
with via it's already demonstrated
[`NewAccount`](https://godoc.org/github.com/ethereum/go-ethereum/accounts#Manager.NewAccount)
method; and to avoid going into transaction creation for now, we can hard-code a random
[`common.Hash`](https://godoc.org/github.com/ethereum/go-ethereum/common#Hash) to sign
instead.

```go
// Create a new account to sign transactions with
signer, _ := am.NewAccount("Signer password");
txHash := common.HexToHash("0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef");
```

With the boilerplate out of the way, we can now sign transaction using the authorization
mechanisms described above:

```go
// Sign a transaction with a single authorization
signature, _ := am.SignWithPassphrase(signer, "Signer password", txHash.Bytes());

// Sign a transaction with multiple manually cancelled authorizations
am.Unlock(signer, "Signer password");
signature, _ = am.Sign(signer.Address, txHash.Bytes());
am.Lock(signer.Address);

// Sign a transaction with multiple automatically cancelled authorizations
am.TimedUnlock(signer, "Signer password", time.Second);
signature, _ = am.Sign(signer.Address, txHash.Bytes());
```

You may wonder why
[`SignWithPassphrase`](https://godoc.org/github.com/ethereum/go-ethereum/accounts#Manager.SignWithPassphrase)
takes an
[`accounts.Account`](https://godoc.org/github.com/ethereum/go-ethereum/accounts#Account)
as the signer, whereas
[`Sign`](https://godoc.org/github.com/ethereum/go-ethereum/accounts#Manager.Sign) takes
only a
[`common.Address`](https://godoc.org/github.com/ethereum/go-ethereum/common#Address). The
reason is that an
[`accounts.Account`](https://godoc.org/github.com/ethereum/go-ethereum/accounts#Account)
object may also contain a custom key-path, allowing
[`SignWithPassphrase`](https://godoc.org/github.com/ethereum/go-ethereum/accounts#Manager.SignWithPassphrase)
to sign using accounts outside of the keystore; however
[`Sign`](https://godoc.org/github.com/ethereum/go-ethereum/accounts#Manager.Sign) relies
on accounts already unlocked within the keystore, so it cannot specify custom paths.

