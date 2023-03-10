---
title: Batch requests
description: How to make batch requests using JSON-RPC API
---

The JSON-RPC [specification](https://www.jsonrpc.org/specification#batch) outlines how clients can send multiple requests at the same time by filling the request objects in an array. This feature is implemented by Geth's API and can be used to cut network delays. Batching offers visible speed-ups specially when used for fetching larger amounts of mostly independent data objects.

Below is an example for fetching a list of blocks in JS:

```js
import fetch from 'node-fetch';

async function main() {
  const endpoint = 'http://127.0.0.1:8545';
  const from = parseInt(process.argv[2]);
  const to = parseInt(process.argv[3]);

  const reqs = [];
  for (let i = from; i < to; i++) {
    reqs.push({
      method: 'eth_getBlockByNumber',
      params: [`0x${i.toString(16)}`, false],
      id: i - from,
      jsonrpc: '2.0'
    });
  }

  const res = await fetch(endpoint, {
    method: 'POST',
    body: JSON.stringify(reqs),
    headers: { 'Content-Type': 'application/json' }
  });
  const data = await res.json();
}

main()
  .then()
  .catch(err => console.log(err));
```

In this case there's no dependency between the requests. Often the retrieved data from one request is needed to issue a second one. Let's take the example of fetching all the receipts for a range of blocks. The JSON-RPC API provides `eth_getTransactionReceipt` which takes in a transaction hash and returns the corresponding receipt object, but no method to fetch receipt objects for a whole block. We need to get the list of transactions in a block and then call `eth_getTransactionReceipt` for each of them.

We can break this into 2 batch requests:

- First to download the list of transaction hashes for all of the blocks in our desired range
- And then to download the list of receipts objects for all of the transaction hashes

For use-cases which depend on several JSON-RPC endpoints the batching approach can get easily complicated. In that case Geth offers a [GraphQL API](/docs/interacting-with-geth/rpc/graphql) which is more suitable.
