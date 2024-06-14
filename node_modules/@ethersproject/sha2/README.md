SHA2 Hash Functions
===================

This sub-module is part of the [ethers project](https://github.com/ethers-io/ethers.js).

It is responsible for common cryptographic hashes and HMAC.

For more information, see the [documentation](https://docs.ethers.io/v5/api/utils/hashing/).


Importing
---------

Most users will prefer to use the [umbrella package](https://www.npmjs.com/package/ethers),
but for those with more specific needs, individual components can be imported.

```javascript
const {

    ripemd160,

    sha256,
    sha512,

    computeHmac,

    // Enums

    SupportedAlgorithm

} = require("@ethersproject/sha2");
```


License
-------

MIT License
