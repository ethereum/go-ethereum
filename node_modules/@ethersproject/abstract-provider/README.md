Abstract Provider
=================

This sub-module is part of the [ethers project](https://github.com/ethers-io/ethers.js).

It is responsible for defining the common interface for a Provider, which in
ethers differs quite substantially from Web3.js.

A Provider is an abstraction of non-account-based operations on a blockchain and
is generally not directly involved in signing transaction or data.

For signing, see the [Abstract Signer](https://www.npmjs.com/package/@ethersproject/abstract-signer)
or [Wallet](https://www.npmjs.com/package/@ethersproject/wallet) sub-modules.

For more information, see the [documentation](https://docs.ethers.io/v5/api/providers/).

Importing
---------

Most users will prefer to use the [umbrella package](https://www.npmjs.com/package/ethers),
but for those with more specific needs, individual components can be imported.

```javascript
const {

    Provider,

    ForkEvent,
    BlockForkEvent,
    TransactionForkEvent,
    TransactionOrderForkEvent,

    // Types
    BlockTag,

    Block,
    BlockWithTransactions,

    TransactionRequest,
    TransactionResponse,
    TransactionReceipt,

    Log,
    EventFilter,

    Filter,
    FilterByBlockHash,

    EventType,
    Listener

} = require("@ethersproject/abstract-provider");
```

License
-------

MIT License
