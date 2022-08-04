---
title: Clef APIs
sort_key: E
---

Clef uses two separate APIs. The **external API** is an untrusted set of JSON-RPC methods that can be called by a user. The **internal API** is a set of JSON-RPC methods that can be called by a UI. The UI could be Clef's native command line interface or a custom UI.

{:toc}
-   this will be removed by the toc

## External API

Clef listens to HTTP requests on `http.addr`:`http.port` (or to IPC on `ipcpath`), with the same JSON-RPC standard as Geth. The messages are expected to be [JSON-RPC 2.0 standard](https://www.jsonrpc.org/specification).

Some of these JSON-RPC calls require user interaction in the Clef terminal. Responses may be delayed significantly or may never be received if a user fails to respond to a confirmation request.

The External API is **untrusted**: it does not accept credentials, nor does it expect that requests have any authority.

See the [external API changelog](https://github.com/ethereum/go-ethereum/blob/master/cmd/clef/extapi_changelog.md) for up to date information about changes to this API.

The External API encoding is as follows:

- number: positive integers that are hex encoded
- data: hex encoded data
- string: ASCII string

All hex encoded values must be prefixed with `0x`.

### Methods

#### account_new

##### Create new password protected account

The signer will generate a new private key, encrypt it according to [web3 keystore spec](https://github.com/ethereum/wiki/wiki/Web3-Secret-Storage-Definition) and store it in the keystore directory.  
The client is responsible for creating a backup of the keystore. If the keystore is lost there is no method of retrieving lost accounts.

##### Arguments

None

##### Result
  - address [string]: account address that is derived from the generated key

##### Sample call
```json
{
  "id": 0,
  "jsonrpc": "2.0",
  "method": "account_new",
  "params": []
}
```
Response
```json
{
  "id": 0,
  "jsonrpc": "2.0",
  "result": "0xbea9183f8f4f03d427f6bcea17388bdff1cab133"
}
```

#### account_list

##### List available accounts
   List all accounts that this signer currently manages

##### Arguments

None

##### Result
  - array with account records:
     - account.address [string]: account address that is derived from the generated key

##### Sample call
```json
{
  "id": 1,
  "jsonrpc": "2.0",
  "method": "account_list"
}
```
Response
```json
{
  "id": 1,
  "jsonrpc": "2.0",
  "result": [
    "0xafb2f771f58513609765698f65d3f2f0224a956f",
    "0xbea9183f8f4f03d427f6bcea17388bdff1cab133"
  ]
}
```

#### account_signTransaction

##### Sign transactions
   Signs a transaction and responds with the signed transaction in RLP-encoded and JSON forms. Supports both legacy and EIP-1559-style transactions. 

##### Arguments
  1. transaction object (legacy):
     - `from` [address]: account to send the transaction from
     - `to` [address]: receiver account. If omitted or `0x`, will cause contract creation.
     - `gas` [number]: maximum amount of gas to burn
     - `gasPrice` [number]: gas price
     - `value` [number:optional]: amount of Wei to send with the transaction
     - `data` [data:optional]:  input data
     - `nonce` [number]: account nonce
  1. transaction object (1559):
     - `from` [address]: account to send the transaction from
     - `to` [address]: receiver account. If omitted or `0x`, will cause contract creation.
     - `gas` [number]: maximum amount of gas to burn
     - `maxPriorityFeePerGas` [number]: maximum priority fee per unit of gas for the transaction
     - `maxFeePerGas` [number]: maximum fee per unit of gas for the transaction
     - `value` [number:optional]: amount of Wei to send with the transaction
     - `data` [data:optional]:  input data
     - `nonce` [number]: account nonce
  3. method signature [string:optional]
     - The method signature, if present, is to aid decoding the calldata. Should consist of `methodname(paramtype,...)`, e.g. `transfer(uint256,address)`. The signer may use this data to parse the supplied calldata, and show the user. The data, however, is considered totally untrusted, and reliability is not expected.


##### Result
  - raw [data]: signed transaction in RLP encoded form
  - tx [json]: signed transaction in JSON form

##### Sample call (legacy)
```json
{
  "id": 2,
  "jsonrpc": "2.0",
  "method": "account_signTransaction",
  "params": [
    {
      "from": "0x1923f626bb8dc025849e00f99c25fe2b2f7fb0db",
      "gas": "0x55555",
      "gasPrice": "0x1234",
      "input": "0xabcd",
      "nonce": "0x0",
      "to": "0x07a565b7ed7d7a678680a4c162885bedbb695fe0",
      "value": "0x1234"
    }
  ]
}
```
Response

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "raw": "0xf88380018203339407a565b7ed7d7a678680a4c162885bedbb695fe080a44401a6e4000000000000000000000000000000000000000000000000000000000000001226a0223a7c9bcf5531c99be5ea7082183816eb20cfe0bbc322e97cc5c7f71ab8b20ea02aadee6b34b45bb15bc42d9c09de4a6754e7000908da72d48cc7704971491663",
    "tx": {
      "nonce": "0x0",
      "gasPrice": "0x1234",
      "gas": "0x55555",
      "to": "0x07a565b7ed7d7a678680a4c162885bedbb695fe0",
      "value": "0x1234",
      "input": "0xabcd",
      "v": "0x26",
      "r": "0x223a7c9bcf5531c99be5ea7082183816eb20cfe0bbc322e97cc5c7f71ab8b20e",
      "s": "0x2aadee6b34b45bb15bc42d9c09de4a6754e7000908da72d48cc7704971491663",
      "hash": "0xeba2df809e7a612a0a0d444ccfa5c839624bdc00dd29e3340d46df3870f8a30e"
    }
  }
}
```

##### Sample call (1559)
```json
{
  "id": 2,
  "jsonrpc": "2.0",
  "method": "account_signTransaction",
  "params": [
    {
      "from": "0xd1a9C60791e8440AEd92019a2C3f6c336ffefA27",
      "to": "0x8A8eAFb1cf62BfBeb1741769DAE1a9dd47996192",
      "gas": "0x33333",
      "maxPriorityFeePerGas": "0x174876E800",
      "maxFeePerGas": "0x174876E800",
      "nonce": "0x0",
      "value": "0x10",
      "data": "0x4401a6e40000000000000000000000000000000000000000000000000000000000000012"
    }
  ]
}
```
Response

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "raw": "0x02f891018085174876e80085174876e80083033333948a8eafb1cf62bfbeb1741769dae1a9dd4799619210a44401a6e40000000000000000000000000000000000000000000000000000000000000012c080a0c8b59180c6e0c154284402b52d772f1afcf8ec2d245cf75bfb3212ebe676135ba02c660aaebf92d5e314fc2ba4c70f018915d174c3c1fc6e4e38d00ebf1a5bb69f",
    "tx": { 
      "type": "0x2", 
      "nonce": "0x0", 
      "gasPrice": null,
      "maxPriorityFeePerGas": "0x174876e800",
      "maxFeePerGas": "0x174876e800",
      "gas": "0x33333",
      "value": "0x10",
      "input": "0x4401a6e40000000000000000000000000000000000000000000000000000000000000012",
      "v": "0x0",
      "r": "0xc8b59180c6e0c154284402b52d772f1afcf8ec2d245cf75bfb3212ebe676135b",
      "s": "0x2c660aaebf92d5e314fc2ba4c70f018915d174c3c1fc6e4e38d00ebf1a5bb69f",
      "to": "0x8a8eafb1cf62bfbeb1741769dae1a9dd47996192",
      "chainId": "0x1",
      "accessList": [],
      "hash": "0x8e096eb11ea89aa83900e6816fb182ff0adb2c85d270998ca2dd2394ec6c5a73"
    }
  }
}
```

##### Sample call with ABI-data


```json
{
  "id": 67,
  "jsonrpc": "2.0",
  "method": "account_signTransaction",
  "params": [
    {
      "from": "0x694267f14675d7e1b9494fd8d72fefe1755710fa",
      "gas": "0x333",
      "gasPrice": "0x1",
      "nonce": "0x0",
      "to": "0x07a565b7ed7d7a678680a4c162885bedbb695fe0",
      "value": "0x0",
      "data": "0x4401a6e40000000000000000000000000000000000000000000000000000000000000012"
    },
    "safeSend(address)"
  ]
}
```
Response

```json
{
  "jsonrpc": "2.0",
  "id": 67,
  "result": {
    "raw": "0xf88380018203339407a565b7ed7d7a678680a4c162885bedbb695fe080a44401a6e4000000000000000000000000000000000000000000000000000000000000001226a0223a7c9bcf5531c99be5ea7082183816eb20cfe0bbc322e97cc5c7f71ab8b20ea02aadee6b34b45bb15bc42d9c09de4a6754e7000908da72d48cc7704971491663",
    "tx": {
      "nonce": "0x0",
      "gasPrice": "0x1",
      "gas": "0x333",
      "to": "0x07a565b7ed7d7a678680a4c162885bedbb695fe0",
      "value": "0x0",
      "input": "0x4401a6e40000000000000000000000000000000000000000000000000000000000000012",
      "v": "0x26",
      "r": "0x223a7c9bcf5531c99be5ea7082183816eb20cfe0bbc322e97cc5c7f71ab8b20e",
      "s": "0x2aadee6b34b45bb15bc42d9c09de4a6754e7000908da72d48cc7704971491663",
      "hash": "0xeba2df809e7a612a0a0d444ccfa5c839624bdc00dd29e3340d46df3870f8a30e"
    }
  }
}
```

Bash example:
```bash
> curl -H "Content-Type: application/json" -X POST --data '{"jsonrpc":"2.0","method":"account_signTransaction","params":[{"from":"0x694267f14675d7e1b9494fd8d72fefe1755710fa","gas":"0x333","gasPrice":"0x1","nonce":"0x0","to":"0x07a565b7ed7d7a678680a4c162885bedbb695fe0", "value":"0x0", "data":"0x4401a6e40000000000000000000000000000000000000000000000000000000000000012"},"safeSend(address)"],"id":67}' http://localhost:8550/

{"jsonrpc":"2.0","id":67,"result":{"raw":"0xf88380018203339407a565b7ed7d7a678680a4c162885bedbb695fe080a44401a6e4000000000000000000000000000000000000000000000000000000000000001226a0223a7c9bcf5531c99be5ea7082183816eb20cfe0bbc322e97cc5c7f71ab8b20ea02aadee6b34b45bb15bc42d9c09de4a6754e7000908da72d48cc7704971491663","tx":{"nonce":"0x0","gasPrice":"0x1","gas":"0x333","to":"0x07a565b7ed7d7a678680a4c162885bedbb695fe0","value":"0x0","input":"0x4401a6e40000000000000000000000000000000000000000000000000000000000000012","v":"0x26","r":"0x223a7c9bcf5531c99be5ea7082183816eb20cfe0bbc322e97cc5c7f71ab8b20e","s":"0x2aadee6b34b45bb15bc42d9c09de4a6754e7000908da72d48cc7704971491663","hash":"0xeba2df809e7a612a0a0d444ccfa5c839624bdc00dd29e3340d46df3870f8a30e"}}}
```

#### account_signData

##### Sign data
   Signs a chunk of data and returns the calculated signature.

##### Arguments
  - content type [string]: type of signed data
     - `text/validator`: hex data with custom validator defined in a contract
     - `application/clique`: [clique](https://github.com/ethereum/EIPs/issues/225) headers
     - `text/plain`: simple hex data validated by `account_ecRecover`
  - account [address]: account to sign with
  - data [object]: data to sign

##### Result
  - calculated signature [data]

##### Sample call
```json
{
  "id": 3,
  "jsonrpc": "2.0",
  "method": "account_signData",
  "params": [
    "data/plain",
    "0x1923f626bb8dc025849e00f99c25fe2b2f7fb0db",
    "0xaabbccdd"
  ]
}
```
Response

```json
{
  "id": 3,
  "jsonrpc": "2.0",
  "result": "0x5b6693f153b48ec1c706ba4169960386dbaa6903e249cc79a8e6ddc434451d417e1e57327872c7f538beeb323c300afa9999a3d4a5de6caf3be0d5ef832b67ef1c"
}
```

#### account_signTypedData

##### Sign data
   Signs a chunk of structured data conformant to [EIP-712](https://github.com/ethereum/EIPs/blob/master/EIPS/eip-712.md) and returns the calculated signature.

##### Arguments
  - account [address]: account to sign with
  - data [object]: data to sign

##### Result
  - calculated signature [data]

##### Sample call
```json
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
Response

```json
{
    "id": 1,
    "jsonrpc": "2.0",
    "result": "0x4355c47d63924e8a72e509b65029052eb6c299d53a04e167c5775fd466751c9d07299936d304c153f6443dfa05f40ff007d72911b6f72307f996231605b915621c"
}
```

#### account_ecRecover

##### Recover the signing address

Derive the address from the account that was used to sign data with content type `text/plain` and the signature.

##### Arguments
  - data [data]: data that was signed
  - signature [data]: the signature to verify

##### Result
  - derived account [address]

##### Sample call
```json
{
  "id": 4,
  "jsonrpc": "2.0",
  "method": "account_ecRecover",
  "params": [
    "0xaabbccdd",
    "0x5b6693f153b48ec1c706ba4169960386dbaa6903e249cc79a8e6ddc434451d417e1e57327872c7f538beeb323c300afa9999a3d4a5de6caf3be0d5ef832b67ef1c"
  ]
}
```
Response

```json
{
  "id": 4,
  "jsonrpc": "2.0",
  "result": "0x1923f626bb8dc025849e00f99c25fe2b2f7fb0db"
}
```

#### account_version

##### Get external API version

Get the version of the external API used by Clef.

##### Arguments

None

##### Result

* external API version [string]

##### Sample call
```json
{
  "id": 0,
  "jsonrpc": "2.0",
  "method": "account_version",
  "params": []
}
```

Response
```json
{
    "id": 0,
    "jsonrpc": "2.0",
    "result": "6.0.0"
}
```


## Internal (UI) API

Clef has one native console-based UI, for operation without any standalone tools. However, there is also an API to communicate with an external UI. To enable that UI, the signer needs to be started with the `--stdio-ui` option, which allocates `stdin` / `stdout` for the UI API.

The internal API methods need to be implemented by a UI listener. By starting the signer with the switch `--stdio-ui-test`, the signer will invoke all known methods, and expect the UI to respond with denials. This can be used during development to ensure that the API is (at least somewhat) correctly implemented.

All methods in this API use object-based parameters, so that there can be no mixup of parameters: each piece of data is accessed by key.

An example (insecure) proof-of-concept external UI has been implemented in [`pythonsigner.py`](https://github.com/ethereum/go-ethereum/blob/master/cmd/clef/pythonsigner.py).

The model is as follows:

* The user starts the UI app (`pythonsigner.py`).
* The UI app starts `clef` with `--stdio-ui`, and listens to the
process output for confirmation-requests.
* `clef` opens the external HTTP API.
* When the `signer` receives requests, it sends a JSON-RPC request via `stdout`.
* The UI app prompts the user accordingly, and responds to `clef`.
* `clef` signs (or not), and responds to the original request.

See the [ui API changelog](https://github.com/ethereum/go-ethereum/blob/master/cmd/clef/intapi_changelog.md) for information about changes to this API.

**NOTE** A slight deviation from `json` standard is in place: every request and response should be confined to a single line.
Whereas the `json` specification allows for linebreaks, linebreaks __should not__ be used in this communication channel, to make
things simpler for both parties.

### Methods

#### ApproveTx / `ui_approveTx`

Invoked when there's a transaction for approval.


##### Sample call

Here's a method invocation:
```bash

curl -i -H "Content-Type: application/json" -X POST --data '{"jsonrpc":"2.0","method":"account_signTransaction","params":[{"from":"0x694267f14675d7e1b9494fd8d72fefe1755710fa","gas":"0x333","gasPrice":"0x1","nonce":"0x0","to":"0x07a565b7ed7d7a678680a4c162885bedbb695fe0", "value":"0x0", "data":"0x4401a6e40000000000000000000000000000000000000000000000000000000000000012"},"safeSend(address)"],"id":67}' http://localhost:8550/
```
Results in the following invocation on the UI:
```json

{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "ui_approveTx",
  "params": [
    {
      "transaction": {
        "from": "0x0x694267f14675d7e1b9494fd8d72fefe1755710fa",
        "to": "0x0x07a565b7ed7d7a678680a4c162885bedbb695fe0",
        "gas": "0x333",
        "gasPrice": "0x1",
        "value": "0x0",
        "nonce": "0x0",
        "data": "0x4401a6e40000000000000000000000000000000000000000000000000000000000000012",
        "input": null
      },
      "call_info": [
          {
            "type": "WARNING",
            "message": "Invalid checksum on to-address"
          },
          {
            "type": "Info",
            "message": "safeSend(address: 0x0000000000000000000000000000000000000012)"
          }
        ],
      "meta": {
        "remote": "127.0.0.1:48486",
        "local": "localhost:8550",
        "scheme": "HTTP/1.1"
      }
    }
  ]
}

```

The same method invocation, but with invalid data:
```bash

curl -i -H "Content-Type: application/json" -X POST --data '{"jsonrpc":"2.0","method":"account_signTransaction","params":[{"from":"0x694267f14675d7e1b9494fd8d72fefe1755710fa","gas":"0x333","gasPrice":"0x1","nonce":"0x0","to":"0x07a565b7ed7d7a678680a4c162885bedbb695fe0", "value":"0x0", "data":"0x4401a6e40000000000000002000000000000000000000000000000000000000000000012"},"safeSend(address)"],"id":67}' http://localhost:8550/
```

```json

{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "ui_approveTx",
  "params": [
    {
      "transaction": {
        "from": "0x0x694267f14675d7e1b9494fd8d72fefe1755710fa",
        "to": "0x0x07a565b7ed7d7a678680a4c162885bedbb695fe0",
        "gas": "0x333",
        "gasPrice": "0x1",
        "value": "0x0",
        "nonce": "0x0",
        "data": "0x4401a6e40000000000000002000000000000000000000000000000000000000000000012",
        "input": null
      },
      "call_info": [
          {
            "type": "WARNING",
            "message": "Invalid checksum on to-address"
          },
          {
            "type": "WARNING",
            "message": "Transaction data did not match ABI-interface: WARNING: Supplied data is stuffed with extra data. \nWant 0000000000000002000000000000000000000000000000000000000000000012\nHave 0000000000000000000000000000000000000000000000000000000000000012\nfor method safeSend(address)"
          }
        ],
      "meta": {
        "remote": "127.0.0.1:48492",
        "local": "localhost:8550",
        "scheme": "HTTP/1.1"
      }
    }
  ]
}


```

One which has missing `to`, but with no `data`:


```json

{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "ui_approveTx",
  "params": [
    {
      "transaction": {
        "from": "",
        "to": null,
        "gas": "0x0",
        "gasPrice": "0x0",
        "value": "0x0",
        "nonce": "0x0",
        "data": null,
        "input": null
      },
      "call_info": [
          {
            "type": "CRITICAL",
            "message": "Tx will create contract with empty code!"
          }
        ],
      "meta": {
        "remote": "signer binary",
        "local": "main",
        "scheme": "in-proc"
      }
    }
  ]
}
```

#### ApproveListing / `ui_approveListing`

Invoked when a request for account listing has been made.

##### Sample call

```json

{
  "jsonrpc": "2.0",
  "id": 5,
  "method": "ui_approveListing",
  "params": [
    {
      "accounts": [
        {
          "url": "keystore:///home/bazonk/.ethereum/keystore/UTC--2017-11-20T14-44-54.089682944Z--123409812340981234098123409812deadbeef42",
          "address": "0x123409812340981234098123409812deadbeef42"
        },
        {
          "url": "keystore:///home/bazonk/.ethereum/keystore/UTC--2017-11-23T21-59-03.199240693Z--cafebabedeadbeef34098123409812deadbeef42",
          "address": "0xcafebabedeadbeef34098123409812deadbeef42"
        }
      ],
      "meta": {
        "remote": "signer binary",
        "local": "main",
        "scheme": "in-proc"
      }
    }
  ]
}

```


#### ApproveSignData / `ui_approveSignData`

##### Sample call

```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "ui_approveSignData",
  "params": [
    {
      "address": "0x123409812340981234098123409812deadbeef42",
      "raw_data": "0x01020304",
      "messages": [
        {
          "name": "message",
          "value": "\u0019Ethereum Signed Message:\n4\u0001\u0002\u0003\u0004",
          "type": "text/plain"
        }
      ],
      "hash": "0x7e3a4e7a9d1744bc5c675c25e1234ca8ed9162bd17f78b9085e48047c15ac310",
      "meta": {
        "remote": "signer binary",
        "local": "main",
        "scheme": "in-proc"
      }
    }
  ]
}
```

#### ApproveNewAccount / `ui_approveNewAccount`

Invoked when a request for creating a new account has been made.

##### Sample call

```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "ui_approveNewAccount",
  "params": [
    {
      "meta": {
        "remote": "signer binary",
        "local": "main",
        "scheme": "in-proc"
      }
    }
  ]
}
```

#### ShowInfo / `ui_showInfo`

The UI should show the info (a single message) to the user. Does not expect response.

##### Sample call

```json
{
  "jsonrpc": "2.0",
  "id": 9,
  "method": "ui_showInfo",
  "params": [
    "Tests completed"
  ]
}

```

#### ShowError / `ui_showError`

The UI should show the error (a single message) to the user. Does not expect response.

```json

{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "ui_showError",
  "params": [
    "Something bad happened!"
  ]
}

```

#### OnApprovedTx / `ui_onApprovedTx`

`OnApprovedTx` is called when a transaction has been approved and signed. The call contains the return value that will be sent to the external caller.  The return value from this method is ignored - the reason for having this callback is to allow the ruleset to keep track of approved transactions.

When implementing rate-limited rules, this callback should be used.

TLDR; Use this method to keep track of signed transactions, instead of using the data in `ApproveTx`.

Example call:
```json

{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "ui_onApprovedTx",
  "params": [
    {
      "raw": "0xf88380018203339407a565b7ed7d7a678680a4c162885bedbb695fe080a44401a6e4000000000000000000000000000000000000000000000000000000000000001226a0223a7c9bcf5531c99be5ea7082183816eb20cfe0bbc322e97cc5c7f71ab8b20ea02aadee6b34b45bb15bc42d9c09de4a6754e7000908da72d48cc7704971491663",
      "tx": {
        "nonce": "0x0",
        "gasPrice": "0x1",
        "gas": "0x333",
        "to": "0x07a565b7ed7d7a678680a4c162885bedbb695fe0",
        "value": "0x0",
        "input": "0x4401a6e40000000000000000000000000000000000000000000000000000000000000012",
        "v": "0x26",
        "r": "0x223a7c9bcf5531c99be5ea7082183816eb20cfe0bbc322e97cc5c7f71ab8b20e",
        "s": "0x2aadee6b34b45bb15bc42d9c09de4a6754e7000908da72d48cc7704971491663",
        "hash": "0xeba2df809e7a612a0a0d444ccfa5c839624bdc00dd29e3340d46df3870f8a30e"
      }
    }
  ]
}
```

#### OnSignerStartup / `ui_onSignerStartup`

This method provides the UI with information about what API version the signer uses (both internal and external) as well as build-info and external API,
in k/v-form.

Example call:
```json

{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "ui_onSignerStartup",
  "params": [
    {
      "info": {
        "extapi_http": "http://localhost:8550",
        "extapi_ipc": null,
        "extapi_version": "2.0.0",
        "intapi_version": "1.2.0"
      }
    }
  ]
}

```

#### OnInputRequired / `ui_onInputRequired`

Invoked when Clef requires user input (e.g. a password).

Example call:
```json

{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "ui_onInputRequired",
  "params": [
    {
      "title": "Account password",
      "prompt": "Please enter the password for account 0x694267f14675d7e1b9494fd8d72fefe1755710fa",
      "isPassword": true
    }
  ]
}
```
