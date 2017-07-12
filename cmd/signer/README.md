**Signer API**
----
The signer utility can be used to sign transactions and data and is meant as a replacement for geth's account management.
This allows DApp's not to depend on geth's account management. When a DApp wants to sign data it can send the data to
the signer, the signer will than provide the user with context and asks the user for permission to sign the data. If
the users grants the signing request the signer will send the signature back to the DApp.
  
This setup allows a DApp to connect to a remote Ethereum node and send transactions that are locally signed. This can
help in situations when a DApp is connected to a remote node because a local Ethereum node is not available, not
synchronised with the chain or a particular Ethereum node that has no build in, or limited account management.
  
In its current form the signer is very limited and designed to work with Mist. It hasn't got a connection to an
Ethereum node. This restriction imposed many limitations such as the lack of ability to keep track of nonces, balances
or fetching additional information that can help the user to make a decision to sign a transaction or data. Currently
the signer only supports password protected accounts. Support for hardware tokens such as Trezor and Legder is planned.

## Command line flags
The signer accepts the following command line options:
- keystore, the directory where the password protected keystore stores keyfiles. The default directory is within geth's datadir. It is OS dependand, use `signer -h` to see where the location is on your system.
- chainid, the chain identifier. Default value is the Ethereum mainnet. See of a list of chain identifiers that are used https://github.com/ethereum/EIPs/blob/master/EIPS/eip-155.md.

Example:
```
signer -keystore /my/keystore -chainid 4
```

## Communicating
The signer listens on stdin for incoming requests and sends responses on stdout. Messages are expected to follow the
[jsonrpc 2.0 standard](http://www.jsonrpc.org/specification).

Some of these call can require user interaction. Clients must be aware that responses may be deplayed significanlty or
may never be received if a users decideds to ignore the confirmation request.

## API

### Encoding
- number: positive integers that are hex encoded
- data: hex encoded data
- string: ASCII string

All hex encoded values must be prefixed with `0x`.

## Methods

### account_new

#### Create new password protected account
The signer will generate a new private key, encrypts it according to [web3 keystore spec](https://github.com/ethereum/wiki/wiki/Web3-Secret-Storage-Definition) and stores it in the keystore directory.
The client is responsible for creating a backup of the keystore. If the keystore is lost there is no method of retrieving
lost accounts.

#### Arguments
  - passphrase [string]: passphrase that is used to protect the private key that is stored within the keystore

#### Result
  - address [string]: account address that is derived from the generated key
  - url [string]: location of the keyfile
  
#### Sample call
```
{
  "id": 0,
  "jsonrpc": "2.0",
  "method": "account_new",
  "params": [
    "my password"
  ]
}

{
  "id": 0,
  "jsonrpc": "2.0",
  "result": {
    "address": "0xbea9183f8f4f03d427f6bcea17388bdff1cab133",
    "url": "keystore:///my/keystore/UTC--2017-08-24T08-40-15.419655028Z--bea9183f8f4f03d427f6bcea17388bdff1cab133"
  }
}
```

### account_list

#### List available accounts
   List all accounts that this signer currently manages

#### Arguments
  none

#### Result
  - array with account records:
     - account.address [string]: account address that is derived from the generated key
     - account.type [string]: type of the 
     - account.url [string]: location of the account
  
#### Sample call
```
{
  "id": 1,
  "jsonrpc": "2.0",
  "method": "account_list"
}

{
  "id": 1,
  "jsonrpc": "2.0",
  "result": [
    {
      "address": "0xafb2f771f58513609765698f65d3f2f0224a956f",
      "type": "account",
      "url": "keystore:///tmp/keystore/UTC--2017-08-24T07-26-47.162109726Z--afb2f771f58513609765698f65d3f2f0224a956f"
    },
    {
      "address": "0xbea9183f8f4f03d427f6bcea17388bdff1cab133",
      "type": "account",
      "url": "keystore:///tmp/keystore/UTC--2017-08-24T08-40-15.419655028Z--bea9183f8f4f03d427f6bcea17388bdff1cab133"
    }
  ]
}
```

### account_signTransaction

#### Sign transactions
   Signs a transactions and respons with the signed transaction in RLP encoded form.

#### Arguments
  - from [address]: account to send the transaction from
  - passphrase [string]: passphrase to unlock the from account
  - Transaction object:
     - transaction.to [address]: receiver account
     - gas [number]: maximum amount of gas to burn
     - gasPrice [number]: gas price
     - value [number:optional]: amount of Wei to send with the transaction
     - data [data:optional]:  input data
     - transaction.nonce [number]: account nonce

#### Result
  - signed transaction in RLP encoded form [data]
  
#### Sample call
```
{
  "id": 2,
  "jsonrpc": "2.0",
  "method": "account_signTransaction",
  "params": [
    "0x1923f626bb8dc025849e00f99c25fe2b2f7fb0db",
    "my password",
    {
      "gas": "0x55555",
      "gasPrice": "0x1234",
      "input": "0xabcd",
      "nonce": "0x0",
      "to": "0x07a565b7ed7d7a678680a4c162885bedbb695fe0",
      "value": "0x1234"
    }
  ]
}

{
  "id": 2,
  "jsonrpc": "2.0",
  "result": "0xf86480821234830555559407a565b7ed7d7a678680a4c162885bedbb695fe0821234802ea028f9ebeff90732eae45692a11c4ca2ef7f631a0a25bf8763d093e770c4ec464aa01fae77b24617913e718b989be78bc1aabb2fed3f2d4e3b93bd36759f1b5b4904"
}
```

### account_sign

#### Sign data
   Signs a chunk of data and returns the calculated signature.

#### Arguments
  - account [address]: account to sign with
  - passphrase [string]: passphrase to unlock the account
  - data [data]: data to sign

#### Result
  - calculated signature [data]
  
#### Sample call
```
{
  "id": 3,
  "jsonrpc": "2.0",
  "method": "account_sign",
  "params": [
    "0x1923f626bb8dc025849e00f99c25fe2b2f7fb0db",
    "my password",
    "0xaabbccdd"
  ]
}

{
  "id": 3,
  "jsonrpc": "2.0",
  "result": "0x5b6693f153b48ec1c706ba4169960386dbaa6903e249cc79a8e6ddc434451d417e1e57327872c7f538beeb323c300afa9999a3d4a5de6caf3be0d5ef832b67ef1c"
}
```

### account_ecRecover

#### Recover address
   Derive the address from the account that was used to sign data from the data and signature.
   
#### Arguments
  - data [data]: data that was signed
  - signature [data]: the signature to verify

#### Result
  - derived account [address]
  
#### Sample call
```
{
  "id": 4,
  "jsonrpc": "2.0",
  "method": "account_ecRecover",
  "params": [
    "0xaabbccdd",
    "0x5b6693f153b48ec1c706ba4169960386dbaa6903e249cc79a8e6ddc434451d417e1e57327872c7f538beeb323c300afa9999a3d4a5de6caf3be0d5ef832b67ef1c"
  ]
}

{
  "id": 4,
  "jsonrpc": "2.0",
  "result": "0x1923f626bb8dc025849e00f99c25fe2b2f7fb0db"
}

```

### account_import

#### Import account
   Import a private key into the keystore. The imported key is expected to be encrypted according to the web3 keystore
   format.
   
#### Arguments
  - account [object]: key in [web3 keystore format](https://github.com/ethereum/wiki/wiki/Web3-Secret-Storage-Definition) (retrieved with account_export) 
  - passphrase [string]: password to decrypt the given account
  - newPassphrase [string]: password to encrypt the imported key in the keystore with 

#### Result
  - imported key [object]:
     - key.address [address]: address of the imported key
     - key.type [string]: type of the account
     - key.url [string]: key URL
  
#### Sample call
```
{
  "id": 6,
  "jsonrpc": "2.0",
  "method": "account_import",
  "params": [
    {
      "address": "c7412fc59930fd90099c917a50e5f11d0934b2f5",
      "crypto": {
        "cipher": "aes-128-ctr",
        "cipherparams": {
          "iv": "401c39a7c7af0388491c3d3ecb39f532"
        },
        "ciphertext": "eb045260b18dd35cd0e6d99ead52f8fa1e63a6b0af2d52a8de198e59ad783204",
        "kdf": "scrypt",
        "kdfparams": {
          "dklen": 32,
          "n": 262144,
          "p": 1,
          "r": 8,
          "salt": "9a657e3618527c9b5580ded60c12092e5038922667b7b76b906496f021bb841a"
        },
        "mac": "880dc10bc06e9cec78eb9830aeb1e7a4a26b4c2c19615c94acb632992b952806"
      },
      "id": "09bccb61-b8d3-4e93-bf4f-205a8194f0b9",
      "version": 3
    },
    "my password",
    "my password"
  ]
}
{
  "id": 6,
  "jsonrpc": "2.0",
  "result": {
    "address": "0xc7412fc59930fd90099c917a50e5f11d0934b2f5",
    "type": "account",
    "url": "keystore:///tmp/keystore/UTC--2017-08-24T11-00-42.032024108Z--c7412fc59930fd90099c917a50e5f11d0934b2f5"
  }
}
```

### account_export

#### Export account from keystore
   Export a private key from the keystore. The exported private key is encrypted with the original passphrase. When the
   key is imported later this passphrase is required.
   
#### Arguments
  - account [address]: export private key that is associated with this account

#### Result
  - exported key, see [web3 keystore format](https://github.com/ethereum/wiki/wiki/Web3-Secret-Storage-Definition) for
  more information
  
#### Sample call
```
{
  "id": 5,
  "jsonrpc": "2.0",
  "method": "account_export",
  "params": [
    "0xc7412fc59930fd90099c917a50e5f11d0934b2f5"
  ]
}
{
  "id": 5,
  "jsonrpc": "2.0",
  "result": {
    "address": "c7412fc59930fd90099c917a50e5f11d0934b2f5",
    "crypto": {
      "cipher": "aes-128-ctr",
      "cipherparams": {
        "iv": "401c39a7c7af0388491c3d3ecb39f532"
      },
      "ciphertext": "eb045260b18dd35cd0e6d99ead52f8fa1e63a6b0af2d52a8de198e59ad783204",
      "kdf": "scrypt",
      "kdfparams": {
        "dklen": 32,
        "n": 262144,
        "p": 1,
        "r": 8,
        "salt": "9a657e3618527c9b5580ded60c12092e5038922667b7b76b906496f021bb841a"
      },
      "mac": "880dc10bc06e9cec78eb9830aeb1e7a4a26b4c2c19615c94acb632992b952806"
    },
    "id": "09bccb61-b8d3-4e93-bf4f-205a8194f0b9",
    "version": 3
  }
}
```



