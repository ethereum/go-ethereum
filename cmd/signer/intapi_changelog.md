### Changelog for internal API (ui-api)

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
