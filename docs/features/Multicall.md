# Multicall

Similar to the multicall provided by [multicall contract](https://github.com/mds1/multicall). The `eth_multiCall` implemented in this repository has two main use cases:
* Aggregate results from multiple contract reads into a single JSON-RPC request.
* Execute multiple state-changing calls in a single transaction.

## Why 
The newly added `eth_multiCall` will solve the following problems existing in the multicall contract:
* Multicall contract is not applicable to old blocks before the deployment of the multicall contract.
* Multicall contract aggregates multiple parameters together and executes them as a whole in the EVM, which is not conducive to implementing computational layer caching.
For example, if this time the calls are [A, B, C, D], and the next call is [B, A, C, D] or [A, E, C, D], then the results of A, C, D cached the first time cannot make the next request hit.
* Multicall contract is executed in series in the EVM. Having too many calls in the multicall contract can easily lead to cumulative time consumption.

## Design
### Parameters

- calls - List[TransactionArgs] Transaction call object list
- blockNrOrHash - Block number or the string latest or pending
- pfastFail - bool, if true, Any single call fails the entire multicall return directly when a failure occurs.
- puseParallel - bool, if true, Calls are executed concurrently, not sequentially one after another.
- pdisableCache - bool, if true, Disable caching of call results.

Returns
- result - List[Result], Call result list
- stats - Stats, The status information of this request
- err 

### Example
Request
``` json
{
  "jsonrpc":"2.0",
  "method":"eth_multiCall",
  "params":[
    [
      {
        "to":"0xf824717e5e7cefd7aa78112fb7e62dd9422488d3",
        "data":"0x3fc1cc26000000000000000000000000ac7a698a85102f7b1dc7345e7f17ebca74e5a9e7000000000000000000000000000000000000000000000000000000000000041f000000000000000000000000000000000000000000000000000000000000000100000000000000000000000004068da6c83afcfa0e13ba15a6696662335d5b75000000000000000000000000000000000000000000000000000000001dcd650000000000000000000000000000000000000000000000000000000000636b1b8d"
      },
      {
        "to":"0x8166994d9ebBe5829EC86Bd81258149B87faCFd3",
        "data":"0x93f1a40b00000000000000000000000000000000000000000000000000000000000000370000000000000000000000004f2769e87c7d96ed9ca72084845ee05e7de5dda2"
      },
      {
        "to":"0xf824717e5e7cefd7aa78112fb7e62dd9422488d3",
        "data":"0x3fc1cc26000000000000000000000000ac7a698a85102f7b1dc7345e7f17ebca74e5a9e7000000000000000000000000000000000000000000000000000000000000041f000000000000000000000000000000000000000000000000000000000000000100000000000000000000000004068da6c83afcfa0e13ba15a6696662335d5b75000000000000000000000000000000000000000000000000000000001dcd650000000000000000000000000000000000000000000000000000000000636b1b8d"
      },
      {
        "to":"0x49894fCC07233957c35462cfC3418Ef0CC26129f",
        "data":"0x70a08231000000000000000000000000205b993bb19930c80fb10ddf4f4c423e49c4caac"
      },
      {
        "to":"0xffb02c56bb2843b794016ddc08ab11a8be7d73ca",
        "data":"0xa799234e0000000000000000000000007e75ce11a7ea2969ea1c0e5b3a9ed4c45ec8363b0000000000000000000000000000000000000000000000a2a15d09519be00000000000000000000000000000000000000000000000000000000000002490181a000000000000000000000000d34ec5fda2e2f1098cd8ba4b883993cc0b9ec8c3000000000000000000000000f6f67f5639caf9bf36e7e32992cb7fa2d7d9df3500000000000000000000000000000000000000000000000000000000000026de00000000000000000000000000000000000000000000000000000000636b1c99"
      },
      {
        "to":"0x3776b8c349bc9af202e4d98af163d59ca56d2fc5",
        "data":"0x8343d129"
      }
    ],
    "0x87e31644c43654708bfc0de50ba8ae3a3d6ecf99eacda6041072ea526c7aec16",
    false,
    true,
    true
  ],
  "id":2
}
``` 
Response
```json
{
  "jsonrpc":"2.0",
  "id":2,
  "result":{
    "results":[
      {
        "err":"execution reverted",
        "fromCache":false,
        "result":"0x",
        "gasUsed":0,
        "timeCost":0.003416628
      },
      {
        "err":"",
        "fromCache":false,
        "result":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
        "gasUsed":23794,
        "timeCost":0.002786535
      },
      {
        "err":"execution reverted",
        "fromCache":false,
        "result":"0x",
        "gasUsed":0,
        "timeCost":0.003377504
      },
      {
        "err":"",
        "fromCache":false,
        "result":"0x0000000000000000000000000000000000000000000000000000000000000000",
        "gasUsed":22718,
        "timeCost":0.002273802
      },
      {
        "err":"",
        "fromCache":false,
        "result":"0x",
        "gasUsed":22884,
        "timeCost":0.001404123
      },
      {
        "err":"",
        "fromCache":false,
        "result":"0x",
        "gasUsed":21064,
        "timeCost":0.001325271
      }
    ],
    "stats":{
      "blockNum": "0x87e31644c43654708bfc0de50ba8ae3a3d6ecf99eacda6041072ea526c7aec16",
      "blockTime":1648512415,
      "success":true,
      "timeCost":0.003840274,
      "gasUsed":90460,
      "usedParallel":true
    }
  }
}
```