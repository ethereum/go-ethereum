---
title: GraphQL Server
description: Documentation for Geth's GraphQL API
---

In addition to the [JSON-RPC APIs](/docs/interacting-with-geth/rpc/), Geth supports the GraphQL API as specified by [EIP-1767](https://eips.ethereum.org/EIPS/eip-1767). GraphQL lets you specify which fields of an objects you need as part of the query, eliminating the extra load on the client for filling in fields which are not needed. It also allows for combining several traditional JSON-RPC requests into one query which translates into less overhead and more performance.

The GraphQL endpoint piggybacks on the HTTP transport used by JSON-RPC. Hence the relevant `--http` flags and the `--graphql` flag should be passed to Geth:

```sh
geth --http --graphql
```

Now queries can be raised against `http://localhost:8545/graphql`. To change the port, provide a custom port number to `--http.port`, e.g.:

```sh
geth --http --http.port 9545 --graphql
```

## GraphiQL {#graphiql}

An easy way to try out queries is the GraphiQL interface shipped with Geth. To open it visit `http://localhost:8545/graphql/ui`. To see how this works let's read the sender, recipient and value of all transactions in block number 6000000. In GraphiQL:

```graphql
query txInfo {
  block(number: 6000000) {
    transactions {
      hash
      from {
        address
      }
      to {
        address
      }
      value
    }
  }
}
```

GraphiQL also provides a way to explore the schema Geth provides to help you formulate your queries, which you can see on the right sidebar. Under the title `Root Types` click on `Query` to see the high-level types and their fields.

## Query {#query}

Reading out data from Geth is the biggest use-case for GraphQL. In addition to using the UI queries can also be sent programmatically. The official GraphQL[docs](https://graphql.org/code/) explain how to find bindings for many languages, or send http requests from the terminal using tools such as Curl.

For example, the code snippet below shows how to obtain the latest block number using Curl. Note the use of a JSON object for the data section:

```sh
❯ curl -X POST http://localhost:8545/graphql -H "Content-Type: application/json" --data '{ "query": "query { block { number } }" }'
{"data":{"block":{"number":6004069}}}
```

Alternatively store the JSON-ified query in a file (let's call it `block-num.query`) and do:

```sh
❯ curl -X POST http://localhost:8545/graphql -H "Content-Type: application/json" --data '@block-num.query'
```

Executing a simple query in JS looks as follows. Here the lightweight library `graphql-request` is used to perform the request. Note the use of variables instead of hardcoding the block number in the query:

```js
const { request, gql } = require('graphql-request');

const query = gql`
  query blockInfo($number: Long) {
    block(number: $number) {
      hash
      stateRoot
    }
  }
`;
request('http://localhost:8545/graphql', query, { number: '6004067' })
  .then(res => {
    console.log(res);
  })
  .catch(err => {
    console.log(err);
  });
```
