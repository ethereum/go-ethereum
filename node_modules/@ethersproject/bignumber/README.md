Big Numbers
===========

This sub-module is part of the [ethers project](https://github.com/ethers-io/ethers.js).

It is responsible for handling arbitrarily large numbers and mathematic operations.

For more information, see the documentation for [Big Numbers](https://docs.ethers.io/v5/api/utils/bignumber/)
and [Fixed-Point Numbers](https://docs.ethers.io/v5/api/utils/fixednumber/).


Importing
---------

Most users will prefer to use the [umbrella package](https://www.npmjs.com/package/ethers),
but for those with more specific needs, individual components can be imported.

```javascript
const {

    BigNumber,

    FixedFormat,
    FixedNumber,

    formatFixed,

    parseFixed

    // Types

    BigNumberish

} = require("@ethersproject/bignumber");
```


License
-------

MIT License
