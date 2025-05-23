Ethereum ABI Coder
==================

This sub-module is part of the [ethers project](https://github.com/ethers-io/ethers.js).

It is responsible for encoding and decoding the Application Binary Interface (ABI)
used by most smart contracts to interoperate between other smart contracts and clients.

For more information, see the [documentation](https://docs.ethers.io/v5/api/utils/abi/).

Importing
---------

Most users will prefer to use the [umbrella package](https://www.npmjs.com/package/ethers),
but for those with more specific needs, individual components can be imported.

```javascript
const {

    ConstructorFragment,
    EventFragment,
    Fragment,
    FunctionFragment,
    ParamType,
    FormatTypes,

    AbiCoder,
    defaultAbiCoder,

    Interface,
    Indexed,

    /////////////////////////
    // Types

    CoerceFunc,
    JsonFragment,
    JsonFragmentType,

    Result,
    checkResultErrors,

    LogDescription,
    TransactionDescription

} = require("@ethersproject/abi");
```

License
-------

MIT License
