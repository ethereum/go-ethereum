## Difficulty calculation

This test shows how the `evm t8n` can be used to calculate the (ethash) difficulty, if none is provided by the caller, 
this time on `GrayGlacier` (Eip 5133).

Calculating it (with an empty set of txs) using `GrayGlacier` rules (and no provided unclehash for the parent block):
```
[user@work evm]$ ./evm t8n --input.alloc=./testdata/19/alloc.json --input.txs=./testdata/19/txs.json --input.env=./testdata/19/env.json --output.result=stdout --state.fork=GrayGlacier
```