## Transaction Pool Feeds

This fork of Geth includes two new types of subscriptions, available through the
eth_subscribe method on Websockets.

### Rejected Transactions

Using Websockets, you can subscribe to a feed of rejected transactions with:

```
{"id": 0, "method": "eth_subscribe", "params":["rejectedTransactions"]}
```

This will immediately return a payload of the form:

```
{"jsonrpc":"2.0","id":0,"result":"$SUBSCRIPTION_ID"}
```

And as messages are rejected by the transaction pool, it will send additional
messages of the form:

```
{
  "jsonrpc": "2.0",
  "method": "eth_subscription",
  "params": {
    "subscription": "$SUBSCRIPTION_ID",
    "result": {
      "tx": "$ETHEREUM_TRANSACTION",
      "reason": "$REJECT_REASON"
    }
  }
}
```

One message will be emitted on this feed for every transaction rejected by the
transaction pool, excluding those rejected because they were already known by
the transaction pool.

It is important that consuming applications process messages quickly enough to
keep up with the process. Geth will buffer up to 20,000 messages, but if that
threshold is reached the subscription will be discarded by the server.

The reject reason corresponds to the error messages returned by Geth within the
txpool. At the time of this writing, these include:

* invalid sender
* nonce too low
* transaction underpriced
* replacement transaction underpriced
* insufficient funds for gas * price + value
* intrinsic gas too low
* exceeds block gas limit
* negative value
* oversized data

However it is possible that in the future Geth may add new error types that
could be included by this response without modification to the rejection feed
itself.

## Dropped Transactions

Using Websockets, you can subscribe to a feed of dropped transaction hashes with:

```
{"id": 0, "method": "eth_subscribe", "params":["droppedTransactions"]}
```

This will immediately return a payload of the form:

```
{"jsonrpc":"2.0","id":0,"result":"$SUBSCRIPTION_ID"}
```

And as messages are dropped from the transaction pool, it will send additional
messages of the form:

```
{
  "jsonrpc": "2.0",
  "method": "eth_subscription",
  "params": {
    "subscription": "0xe5fa5d3c8ec05953bd746a784cfeade6",
    "result": {
      "txhash": "$TRANSACTION_HASH",
      "reason": "$REASON"
    }
  }
}
```

One message will be emitted on this feed for every transaction dropped from the
transaction pool.

It is important that consuming applications process messages quickly enough to
keep up with the process. Geth will buffer up to 20,000 messages, but if that
threshold is reached the subscription will be discarded by the server.

The following reasons may be included as reasons transactions were rejected:

* underpriced-txs: Indicates the transaction's gas price is below the node's threshold.
* low-nonce-txs: Indicates that the account nonce for the sender of this transaction has exceeded the nonce on this transction. That may happen when this transaction is included in a block, or when a replacement transaction is included in a block.
* unpayable-txs: Indicates that the sender lacks sufficient funds to pay the intrinsic gas for this transaction
* account-cap-txs: Indicates that this account has sent enough transactions to exceed the per-account limit on the node.
* replaced-txs: Indicates that the transaction was dropped because a replacement transaction with the same nonce and higher gas has replaced it.
* unexecutable-txs: Indicates that a transaction is no longer considered executable. This typically applies to queued transaction, when a dependent pending transaction was removed for a reason such as unpayable-txs.
* truncating-txs: The transaction was dropped because the number of transactions in the mempool exceeds the allowable limit.
* old-txs: The transaction was dropped because it has been in the mempool longer than the allowable period of time without inclusion in a block.
* updated-gas-price: The node's minimum gas price was updated, and transactions below that price were dropped.
