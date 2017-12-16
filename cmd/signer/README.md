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
or fetching additional information that can help the user to make a decision to sign a transaction or data. 

## Command line flags
The signer accepts the following command line options:
```
   --chainid value    chain identifier (default: 1)
   --loglevel value   log level to emit to the screen (default: 4)
   --keystore value   Directory for the keystore (default: "/home/martin/.ethereum/keystore")
   --networkid value  Network identifier (integer, 1=Frontier, 2=Morden (disused), 3=Ropsten, 4=Rinkeby) (default: 1)
   --lightkdf         Reduce key-derivation RAM & CPU usage at some expense of KDF strength
   --nousb            Disables monitoring for and managing USB hardware wallets
   --rpcaddr value    HTTP-RPC server listening interface (default: "localhost")
   --rpcport value    HTTP-RPC server listening port (default: 8550)
   --4bytedb value    File containing 4byte-identifiers (default: "./4byte.json")
   --stdio-ui         Use STDIN/STDOUT as a channel for an external UI. This means that an STDIN/STDOUT is used for RPC-communication with a e.g. a graphical user interface, and can be used when the signer is started by an external process.
   --help, -h         show help
```


Example:
```
signer -keystore /my/keystore -chainid 4
```

## Communication

### External API

The signer listens to HTTP requests on `rpcaddr`:`rpcport`. The messages are
expected to be JSON [jsonrpc 2.0 standard](http://www.jsonrpc.org/specification).

Some of these call can require user interaction. Clients must be aware that responses
may be deplayed significanlty or may never be received if a users decideds to ignore the confirmation request.

The External API is **untrusted** : it does not accept credentials over this api, nor does it expect
that requests have any authority.

### UI API

The signer has one native console-based UI, for operation without any standalone tools.
However, there is also an API to communicate with an external UI. To enable that UI,
the signer needs to be executed with the `--stdio-ui` option, which allocates the
`stdin`/`stdout` for the UI-api.

An example (insecure) proof-of-concept of has been implemented in `pythonsigner.py`.

The model is as follows:

* The user starts the UI app (`pythonsigner.py`).
* The UI app starts the `signer` with `--stdio-ui`, and listens to the
process output for confirmation-requests.
* The `signer` opens the external http api.
* When the `signer` receives requests, it sends a `jsonrpc` request via `stdout`.
* The UI app prompts the user accordingly, and responds to the `signer`
* The `signer` signs (or not), and responds to the original request.

## External API

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

None

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

None

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
   Signs a transactions and responds with the signed transaction in RLP encoded form.

#### Arguments
  1. from [address]: account to send the transaction from
  2. transaction object:
     - `to` [address]: receiver account
     - `gas` [number]: maximum amount of gas to burn
     - `gasPrice` [number]: gas price
     - `value` [number:optional]: amount of Wei to send with the transaction
     - `data` [data:optional]:  input data
     - `nonce` [number]: account nonce
  3. method signature [string:optional]
     - The method signature, if present, is to aid decoding the calldata. Should consist of `methodname(paramtype,...)`, e.g. `transfer(uint256,address)`. The signer may use this data to parse the supplied calldata, and show the user. The data, however, is considered totally untrusted, and reliability is not expected.


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



## UI API

These methods needs to be implemented by a UI listener.

still work in progress

### Rules for UI apis

A UI should conform to the following rules.

* A UI MUST NOT load any external resources that were not embedded/part of the UI package.
  * For example, not load icons, stylesheets from the internet
  * Not load files from the filesystem, unless they reside in the same local directory (e.g. config files)
* A UI MUST NOT open any ports or services
  * The signer opens the public port
* A UI SHOULD verify the permissions on the signer binary, and refuse to execute or warn if permissions allow non-user write.
* A UI SHOULD inform the user about the `SHA256` or `MD5` hash of the binary being executed
* A UI SHOULD NOT maintain a secondary storage of data, e.g. list of accounts
  * The signer provides accounts
* A UI SHOULD, to the best extent possible, use static linking / bundling, so that requried libraries are bundled
along with the UI.


## TODOs

Some snags and todos

* Currently, the API does not make it possible for the signer to forward data about the
checksum, since the addresses are common.Address, and not String. This should be changed upstream,
so that they are some more complex form with both common.Address and the original string (?)


* The audit-log perhaps leave some things to be desired. I have not found a perfect way to save an audit log of events.

* Some more fields should be added to calldata, e.g http-header `Origin`.

* The signer should take a startup param "--no-change", for UI:s that do not contain the capability
   to perform changes to things, only approve/deny. Such a UI should be able to start the signer in
   a more secure mode by telling it that it only wants approve/deny capabilities.

* It would be nice if the signer could collect new 4byte-id:s/method selectors, and have a
secondary database for those (`4byte_custom.json`). Users could then (optionally) submit their collections for
inclusion upstream.

* It should be possible to configure the signer to check if an account is indeed known to it, before
passing on to the UI. The reason it currently does not, is that it would make it possible to enumerate
accounts if it immediately returned "unknown account". Similarly, it should be possible to configure
the signer to auto-allow listing (certain) accounts, instead of asking every time.

* Upon startup, the signer should spit out some info to the caller (particularly important when executed in `stdio-ui`-mode),
invoking methods with the following info:
  * Version info about the signer
  * Address of API (http/ipc)
    * This makes it posible for the UI to use the api for creating transactions
  * List of known accounts

* The signer should pass the `Origin` header as call-info to the UI. As of right now, the way that info about the request is
put together is a bit of a hack into the http server. This could probably be greatly improved

* Geth relay
    - Geth should be started in `geth --external_signer localhost:8550`.

* Wallets / accounts. Add API methods for wallets.

* Rules. In the future, it should be possible to specify rules, e.g. "Allow sending up to 1 eth per day to contract Y". Two problems:
   * There needs to be a very good structure around rules. Either a full language (lua/js) or a limited but flexible syntax based on e.g. json/yaml. Kind of like a firewall ruleset.
   * This implies that the signer would remember passwords, which is very problematic. However, a good UI implementation will want these things,
   and it would be better to implement it once in the signer, than having UI:s develop their own remember-password logic.
  
