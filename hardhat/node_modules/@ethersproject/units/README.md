Ethereum Unit Conversion Utilities
==================================

This sub-module is part of the [ethers project](https://github.com/ethers-io/ethers.js).

It contains functions to convert between string representations and numeric
representations of numbers, including those out of the range of JavaScript.

For more information, see the [documentation](https://docs.ethers.io/v5/api/utils/display-logic/).


Importing
---------

Most users will prefer to use the [umbrella package](https://www.npmjs.com/package/ethers),
but for those with more specific needs, individual components can be imported.

```javascript
const {

    formatUnits,
    parseUnits,

    formatEther,
    parseEther,

    commify

} = require("@ethersproject/units");
```


License
-------

MIT License
