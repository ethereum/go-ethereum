## Changelog for external API

The API uses [semantic versioning](https://semver.org/).

TL;DR: Given a version number MAJOR.MINOR.PATCH, increment the:

* MAJOR version when you make incompatible API changes,
* MINOR version when you add functionality in a backwards-compatible manner, and
* PATCH version when you make backwards-compatible bug fixes.

Additional labels for pre-release and build metadata are available as extensions to the MAJOR.MINOR.PATCH format.


### 6.0.0

* `New` was changed to deliver only an address, not the full `Account` data
* `Export` was moved from External API to the UI Server API

#### 5.0.0

* The external `account_EcRecover`-method was reimplemented.
* The external method `account_sign(address, data)` was replaced with `account_signData(contentType, address, data)`.
The addition of `contentType` makes it possible to use the method for different types of objects, such as:
  * signing data with an intended validator (not yet implemented)
  * signing clique headers,
  * signing plain personal messages,
* The external method `account_signTypedData` implements [EIP-712](https://github.com/ethereum/EIPs/blob/master/EIPS/eip-712.md) and makes it possible to sign typed data.

#### 4.0.0

* The external `account_Ecrecover`-method was removed.
* The external `account_Import`-method was removed.

#### 3.0.0

* The external `account_List`-method was changed to not expose `url`, which contained info about the local filesystem. It now returns only a list of addresses.

#### 2.0.0

* Commit `73abaf04b1372fa4c43201fb1b8019fe6b0a6f8d`, move `from` into `transaction` object in `signTransaction`. This
makes the `accounts_signTransaction` identical to the old `eth_signTransaction`.


#### 1.0.0

Initial release.
