# EVM Transaction Tracing

Ethereum has two different types of transactions: regular transfers and contract executions. A regular transfer simply moves Eth from one account to another. However, if the recipient of the transaction is a contract account with associated EVM (Ethereum Virtual Machine) bytecode, this bytecode will be executed as part of the transaction, in addition to transferring any Eth.

We can check whether this execution was successful through the status code of the transaction receipt, but there's no way to see which data was modified halfway, nor which external contracts were called. To understand what a transaction has done, we need to trace the transaction.

`trace_transaction` is an RPC method exposed by the Openethereum and Erigon Ethereum clients. You can get the EVM traces of a previously executed transaction using this method. This can be useful for debugging purposes, or for understanding how a transaction works.

## Why

The Tracer implemented by Geth/Erigon has relatively poor performance and cannot meet our business needs. The `debug_tracexx/trace_xx` functions of Geth/Erigon perform well for tracing recent transactions (within 128 blocks). However, for tracing historical transactions, the RPC usually takes more than 1 second to return. The fundamental reason is that their trace needs to first backtrack to a specific block and replay all transactions prior to the transaction to be traced in that block, which leads to inefficiency. 

`trace_transaction` implemented in this repository is generated as an add-on during transaction execution and stored in a designated database. The `trace_transaction` only requires a single read to obtain the trace, and the RPC response time is at the millisecond level. 

Moreover, most non-Ethereum public chains are forked from Go-Ethereum, so there is a strong demand for a high-performance tracer in Go version for subsequent business. 

## Design
### Parameters
This method only takes one parameter which is the transaction hash of the transaction whose traces you wish to get:

- Hash - Transaction hash

``` json
params: ["0x17104ac9d3312d8c136b7f44d4b8b47852618065ebfa534bd2d3b5ef218ca1f3"]
```

Returns
- Array - Traces of given transaction

### Example
Request
```bash
curl --data '{"method":"trace_transaction","params":["0x17104ac9d3312d8c136b7f44d4b8b47852618065ebfa534bd2d3b5ef218ca1f3"],"id":1,"jsonrpc":"2.0"}' -H "Content-Type: application/json" -X POST localhost:8545
```
Response
```
{
  "id": 1,
  "jsonrpc": "2.0",
  "result": [
    {
      "action": {
        "callType": "call",
        "from": "0x1c39ba39e4735cb65978d4db400ddd70a72dc750",
        "gas": "0x13e99",
        "input": "0x16c72721",
        "to": "0x2bd2326c993dfaef84f696526064ff22eba5b362",
        "value": "0x0"
      },
      "blockHash": "0x7eb25504e4c202cf3d62fd585d3e238f592c780cca82dacb2ed3cb5b38883add",
      "blockNumber": 3068185,
      "result": {
        "gasUsed": "0x183",
        "output": "0x0000000000000000000000000000000000000000000000000000000000000001"
      },
      "subtraces": 0,
      "traceAddress": [
        0
      ],
      "transactionHash": "0x17104ac9d3312d8c136b7f44d4b8b47852618065ebfa534bd2d3b5ef218ca1f3",
      "transactionPosition": 2,
      "type": "call"
    },
    ...
  ]
}
```

## Reference
- [EVM  Tracin](https://geth.ethereum.org/docs/developers/evm-tracing)
- [Parity Trace Module](https://openethereum.github.io/JSONRPC-trace-module)
