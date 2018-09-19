### Changelog for external API

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

### Versioning

The API uses [semantic versioning](https://semver.org/).

TLDR; Given a version number MAJOR.MINOR.PATCH, increment the:

* MAJOR version when you make incompatible API changes,
* MINOR version when you add functionality in a backwards-compatible manner, and
* PATCH version when you make backwards-compatible bug fixes.

Additional labels for pre-release and build metadata are available as extensions to the MAJOR.MINOR.PATCH format.
