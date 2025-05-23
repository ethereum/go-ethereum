Etheruem Hash Utilities
=======================

This sub-module is part of the [ethers project](https://github.com/ethers-io/ethers.js).

It contains several common hashing utilities (but not the actual hash functions).

For more information, see the [documentation](https://docs.ethers.io/v5/api/utils/hashing/).

Importing
---------

Most users will prefer to use the [umbrella package](https://www.npmjs.com/package/ethers),
but for those with more specific needs, individual components can be imported.

```javascript
const {

    isValidName,
    namehash,

    id,

    messagePrefix,
    hashMessage

} = require("@ethersproject/hash");
```


License
-------

MIT License
