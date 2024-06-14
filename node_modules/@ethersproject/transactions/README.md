Ethereum Transaction Utilities
==============================

This sub-module is part of the [ethers project](https://github.com/ethers-io/ethers.js).

It contains various functions for encoding and decoding serialized transactios.

For more information, see the [documentation](https://docs.ethers.io/v5/api/utils/transactions/).


Importing
---------

Most users will prefer to use the [umbrella package](https://www.npmjs.com/package/ethers),
but for those with more specific needs, individual components can be imported.

```javascript
const {

    computeAddress,
    recoverAddress,

    serialize,
    parse,

    // Types

    Transaction,
    UnsignedTransaction

} = require("@ethersproject/transactions");
```


License
-------

MIT License
