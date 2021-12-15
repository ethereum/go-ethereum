## Difficulty calculation

This test shows how the `evm t8n` can be used to calculate the (ethash) difficulty, if none is provided by the caller, 
this time on `ArrowGlacier` (Eip 4345).

Calculating it (with an empty set of txs) using `ArrowGlacier` rules (and no provided unclehash for the parent block):
```
[user@work evm]$ ./evm t8n --input.alloc=./testdata/14/alloc.json --input.txs=./testdata/14/txs.json --input.env=./testdata/14/env.json --output.result=stdout --state.fork=ArrowGlacier
```