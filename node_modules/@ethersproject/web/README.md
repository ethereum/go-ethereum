Web Utilities
=============

This sub-module is part of the [ethers project](https://github.com/ethers-io/ethers.js).

It contains functions to abstract safely and responsibly connecting to the web,
including exponential back-off.

For more information, see the [documentation](https://docs.ethers.io/v5/api/utils/web/).

Importing
---------

Most users will prefer to use the [umbrella package](https://www.npmjs.com/package/ethers),
but for those with more specific needs, individual components can be imported.

```javascript
const {

    fetchJson,

    poll,

    // Types
    ConnectionInfo,
    FetchJsonResponse,

    PollOptions,
    OncePollable,
    OnceBlockable

} = require("@ethersproject/web");
```


License
-------

MIT License
