Hierarchal Deterministic Utilities (BIP32)
==========================================

This sub-module is part of the [ethers project](https://github.com/ethers-io/ethers.js).

It is responsible computing, deriving, encoding and decoding Hierarchal-Deterministic
private keys.

For more information, see the [documentation](https://docs.ethers.io/v5/api/utils/hdnode/).

Importing
---------

Most users will prefer to use the [umbrella package](https://www.npmjs.com/package/ethers),
but for those with more specific needs, individual components can be imported.

```javascript
const {

    HDNode,

    defaultPath,

    mnemonicToSeed,
    mnemonicToEntropy,
    entropyToMnemonic,
    isValidMnemonic,

    // Types

    Mnemonic

} = require("@ethersproject/hdnode");
```


License
-------

MIT License
