### Changelog for external API



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
