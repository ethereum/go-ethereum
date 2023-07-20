# EVM Pre Execution
Interacting with contracts is not as straightforward as EOA (Externally Owned Account) transfers. Users need to know how their assets change after submitting transactions to avoid potential risks to the greatest extent. Pre-execution executes related transactions on the latest block and collects state changes, event logs, and the call and create call stack during EVM execution.

## Why 
In JSON-RPC, there already exist `eth_call` and `eth_estimateGas` for simulating the result and gas usage of user transactions. This is sufficient for ordinary transfer requests, but for contract calls, we also need to know which data has been modified halfway, which external contracts have been called, and the Event Log during contract execution.

The `pre_traceMany` enhances the tracing capabilities of our API, allowing users to trace multiple transactions in a single call. This can be particularly useful for debugging complex interactions between multiple transactions.

The defining feature of `pre_traceMany` is its sequential execution of transactions, where each transaction is executed based on the final state of the preceding one, thereby creating a cumulative effect on the state.

The primary use case for this feature is pre-execution checks in wallets. When a user wants to understand the potential asset changes from interacting with an unauthorized contract, debug_traceCall requires the user to approve the contract before it can estimate the user's asset changes, which introduces potential risks. However, with debug_traceCallMany, there's no need for the user to approve the contract to make an estimate (the user's contract approval just needs to be passed in as a preceding transaction). In fact, the pre-transaction balance change feature in Rabby Wallet is based on this method.

## Design
### Parameters

- calls - List[TransactionArgs] Transaction call object list

Returns
- List[PreResult]

PreResult
- trace - List[TraceResult], trace result list
- Logs - List[Log], contract log event, 
- StateDiff - state change
- Error - error message
- GasUsed - gas used of the transaction

### Example 
Request
```json
{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "pre_traceMany",
    "params": [
        [
            {
                "chainId": 1,
                "data": "0x5ae401dc0000000000000000000000000000000000000000000000000000000063735a2b00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000001a00000000000000000000000000000000000000000000000000000000000000124b858183f00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000494010000000000000000000000000000000000000000000000000000ab48ad714c000000000000000000000000000000000000000000000000000000000000000042dac17f958d2ee523a2206206994597c13d831ec70001f46b175474e89094c44da98b954eedeac495271d0f0001f4c02aaa39b223fe8d0a0e5c4f27ead9083c756cc200000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000004449404b7c0000000000000000000000000000000000000000000000000000ab48ad714c000000000000000000000000005853ed4f26a3fcea565b3fbc698bb19cdf6deb8500000000000000000000000000000000000000000000000000000000",
                "from": "0x5853ed4f26a3fcea565b3fbc698bb19cdf6deb85",
                "to": "0x68b3465833fb72a70ecdf485e0e4c7bd8665fc45",
                "gas_price": "0x0",
                "nonce": "0xD36",
                "value": "0x0"
            }
        ]
    ]
}
```
Response 
```json
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": [
        {
            "trace": [
                ...
                {
                    "action": {
                        "callType": "call",
                        "from": "0x68b3465833fb72a70ecdf485e0e4c7bd8665fc45",
                        "to": "0x5853ed4f26a3fcea565b3fbc698bb19cdf6deb85",
                        "value": "0xd65434b5d543",
                        "gas": "0x7c07fffffffc7756",
                        "input": "0x"
                    },
                    "blockHash": "0x5fd7ab63f50e1847a61a29dc0e0fa2f04b92854a3bc517ca8e6aac6a2ab8ff16",
                    "blockNumber": 15974376,
                    "result": {
                        "gasUsed": "0x0",
                        "output": "0x"
                    },
                    "subtraces": 0,
                    "traceAddress": [
                        1,
                        2
                    ],
                    "transactionHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
                    "transactionPosition": 0,
                    "type": "call"
                }
            ],
            "logs": [
                ...
                {
                    "address": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
                    "topics": [
                        "0x7fcf532c15f0a6db0bd6d0e038bea71d30d808c7d98cb3bf7268a95bf5081b65",
                        "0x00000000000000000000000068b3465833fb72a70ecdf485e0e4c7bd8665fc45"
                    ],
                    "data": "0x0000000000000000000000000000000000000000000000000000d65434b5d543",
                    "blockNumber": "0xf3bfe8",
                    "transactionHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
                    "transactionIndex": "0x0",
                    "blockHash": "0x5fd7ab63f50e1847a61a29dc0e0fa2f04b92854a3bc517ca8e6aac6a2ab8ff16",
                    "logIndex": "0x6",
                    "removed": false
                }
            ],
            "stateDiff": {
                ...
                "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": {
                    "0xdc51a0a44317b550a6b5ddb34687ce2bd40e4750eec27343e32b23e686739208": {
                        "before": "0x0000000000000000000000000000000000000000000000000000000000000000",
                        "after": "0x0000000000000000000000000000000000000000000000000000d65434b5d543"
                    },
                    "0xf762dfe765e313d39f5dd6e34e29a9ef0af51578e67f7f482bb4f8efd984976b": {
                        "before": "0x0000000000000000000000000000000000000000000000774b2e49fbce11d66a",
                        "after": "0x0000000000000000000000000000000000000000000000774b2d73a7995c0127"
                    }
                },
                "0xdac17f958d2ee523a2206206994597c13d831ec7": {
                    "0x7319dbd3df87dc649aecf2c43364f3729a895a26487ee485b7813b28cddd22fe": {
                        "before": "0x000000000000000000000000000000000000000000000000000000dc6b650f79",
                        "after": "0x000000000000000000000000000000000000000000000000000000dc6b69a37a"
                    },
                    "0xaa92b417b47961809296951f3d875b19705bd9dea1aa082bb9fed9fdc898f2af": {
                        "before": "0x0000000000000000000000000000000000000000000000000000000000049401",
                        "after": "0x0000000000000000000000000000000000000000000000000000000000000000"
                    }
                }
            },
            "error": {
                "code": 0,
                "msg": ""
            },
            "gasUsed": 195350
        }
    ]
}
```