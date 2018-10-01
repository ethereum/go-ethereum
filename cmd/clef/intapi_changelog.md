### Changelog for internal API (ui-api)

### 2.1.0

* Add `OnInputRequired(info UserInputRequest)` to internal API. This method is used when Clef needs user input, e.g. passwords.

The following structures are used:
```golang
       UserInputRequest struct {
               Prompt     string `json:"prompt"`
               Title      string `json:"title"`
               IsPassword bool   `json:"isPassword"`
       }
       UserInputResponse struct {
               Text string `json:"text"`
       }
```

### 2.0.0

* Modify how `call_info` on a transaction is conveyed. New format:

```
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "ApproveTx",
  "params": [
    {
      "transaction": {
        "from": "0x82A2A876D39022B3019932D30Cd9c97ad5616813",
        "to": "0x07a565b7ed7d7a678680a4c162885bedbb695fe0",
        "gas": "0x333",
        "gasPrice": "0x123",
        "value": "0x10",
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
          "type": "WARNING",
          "message": "Tx contains data, but provided ABI signature could not be matched: Did not match: test (0 matches)"
        }
      ],
      "meta": {
        "remote": "127.0.0.1:54286",
        "local": "localhost:8550",
        "scheme": "HTTP/1.1"
      }
    }
  ]
}
```

#### 1.2.0

* Add `OnStartup` method, to provide the UI with information about what API version
the signer uses (both internal and external) aswell as build-info and external api.

Example call:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "OnSignerStartup",
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

#### 1.1.0

* Add `OnApproved` method

#### 1.0.0

Initial release.

### Versioning

The API uses [semantic versioning](https://semver.org/).

TLDR; Given a version number MAJOR.MINOR.PATCH, increment the:

* MAJOR version when you make incompatible API changes,
* MINOR version when you add functionality in a backwards-compatible manner, and
* PATCH version when you make backwards-compatible bug fixes.

Additional labels for pre-release and build metadata are available as extensions to the MAJOR.MINOR.PATCH format.
