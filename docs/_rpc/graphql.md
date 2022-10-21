---
title: GraphQL Server
sort_key: C
---

In addition to the [JSON-RPC APIs](../rpc/server), Geth supports the GraphQL API as specified by [EIP-1767](https://eips.ethereum.org/EIPS/eip-1767). GraphQL lets you specify which fields of an objects you need as part of the query, eliminating the extra load on the client for filling in fields which are not needed. It also allows for combining several traditional JSON-RPC requests into one query which translates into less overhead and more performance.

The GraphQL endpoint piggybacks on the HTTP transport used by JSON-RPC. Hence you'll have to enable and configure the relevant `--http` flags, and the `--graphql` flag itself:

```bash
geth --http --graphql
```

Now you can start querying against `http://localhost:8545/graphql`. To change the port, you'll need to provide `--http.port`, e.g.:

```bash
geth --http --http.port 9545 --graphql
```

### GraphiQL

An easy way to get started right away and try out queries is the GraphiQL interface shipped with Geth. To open it visit `http://localhost:8545/graphql/ui`. To see how this works let's read the sender, recipient and value of all transactions in block number 6000000. Try this out in GraphiQL:

```graphql
query txInfo {
    block (number: 6000000) { transactions { hash from { address } to { address } value } }
}
```

GraphiQL also provides a way to explore the schema Geth provides to help you formulate your queries, which you can see on the right sidebar. Under the title `Root Types` click on `Query` to see the high-level types and their fields.

### Query

Reading out data from Geth is the biggest use-case for GraphQL. However after trying out queries in the UI you may want to do it programmatically. You can consult the official [docs](graphql-code) to find bindings for your language. Or use your favorite tool for sending HTTP requests. For sake of completeness we briefly touch on two approaches here. First via cURL, and second via a JS script.

Here's how you'd get the latest block's number via cURL. Note the use of a JSON object for the data section:

```bash
❯ curl -X POST http://localhost:8545/graphql -H "Content-Type: application/json" --data '{ "query": "query { block { number } }" }'
{"data":{"block":{"number":6004069}}}
```

Alternatively store the JSON-ified query in a file (let's call it `block-num.query`) and do:

```bash
❯ curl -X POST http://localhost:8545/graphql -H "Content-Type: application/json" --data '@block-num.query'
```

Executing a simple query in JS looks like the following. Here we're using the lightweight library `graphql-request` to perform the request. Note the use of variables instead of hardcoding the block number in the query:

```javascript
const { request, gql } = require('graphql-request')

const query = gql`
    query blockInfo($number: Long) {
        block (number: $number) { hash stateRoot }
    }
`
request('http://localhost:8545/graphql', query, { number: '6004067' })
    .then((res) => { console.log(res) })
    .catch((err) => { console.log(err) })
```

[eip-1767]: https://eips.ethereum.org/EIPS/eip-1767
[graphql-code]: https://graphql.org/code/
