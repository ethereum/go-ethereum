## EIP 4788

This test contains testcases for EIP-4788. The 4788-contract is 
located at address `0xbeac00ddb15f3b6d645c48263dc93862413a222d`, and this test executes a simple transaction. It also
implicitly invokes the system tx, which sets calls the contract and sets the 
storage values
```
$ dir=./testdata/29/ && go run . t8n --state.fork=Cancun  --input.alloc=$dir/alloc.json --input.txs=$dir/txs.json --input.env=$dir/env.json --output.alloc=stdout
INFO [08-15|20:07:56.335] Trie dumping started                     root=ecde45..2af8a7
INFO [08-15|20:07:56.335] Trie dumping complete                    accounts=2 elapsed="225.848µs"
INFO [08-15|20:07:56.335] Wrote file                               file=result.json
{
  "alloc": {
    "0xa94f5374fce5edbc8e2a8697c15331677e6ebf0b": {
      "balance": "0x16345785d871db8",
      "nonce": "0x1"
    },
    "0xbeac00541d49391ed88abf392bfc1f4dea8c4143": {
      "code": "0x3373fffffffffffffffffffffffffffffffffffffffe14604457602036146024575f5ffd5b620180005f350680545f35146037575f5ffd5b6201800001545f5260205ff35b6201800042064281555f359062018000015500",
      "storage": {
        "0x000000000000000000000000000000000000000000000000000000000000079e": "0x000000000000000000000000000000000000000000000000000000000000079e",
        "0x000000000000000000000000000000000000000000000000000000000001879e": "0x0000beac00beac00beac00beac00beac00beac00beac00beac00beac00beac00"
      },
      "balance": "0x
    }
  }
}

```
