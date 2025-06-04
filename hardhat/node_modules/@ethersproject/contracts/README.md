Ethereum Contract Meta-Class
============================

This sub-module is part of the [ethers project](https://github.com/ethers-io/ethers.js).

It is creating (at run-time) an object which interacts with an on-chain
contract as a native JavaScript object.

If you are familiar with ORM for Databases, this is similar, but for smart contracts.

For more information, see the [documentation](https://docs.ethers.io/v5/api/contract/).

Importing
---------

Most users will prefer to use the [umbrella package](https://www.npmjs.com/package/ethers),
but for those with more specific needs, individual components can be imported.

```javascript
const {

    Contract,
    ContractFactory,

    RunningEvent,

    // Types

    ContractInterface,

    Overrides,
    PayableOverrides,
    CallOverrides,

    PopulatedTransaction,

    EventFilter,

    ContractFunction,

    Event,
    ContractReceipt,
    ContractTransaction

} = require("@ethersproject/contracts");
```


License
-------

MIT License
