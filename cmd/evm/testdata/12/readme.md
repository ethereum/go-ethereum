## Test 1559 balance + gasCap

This test contains an EIP-1559 consensus issue which happened on Ropsten, where
`geth` did not properly account for the value transfer while doing the check on `max_fee_per_gas * gas_limit`.

Before the issue was fixed, this invocation allowed the transaction to pass into a block:
```
dir=./testdata/12 && ./evm t8n --state.fork=London --input.alloc=$dir/alloc.json --input.txs=$dir/txs.json --input.env=$dir/env.json --output.alloc=stdout --output.result=stdout
```

With the fix applied, the result is: 
```
dir=./testdata/12 && ./evm t8n --state.fork=London --input.alloc=$dir/alloc.json --input.txs=$dir/txs.json --input.env=$dir/env.json --output.alloc=stdout --output.result=stdout
INFO [07-21|19:03:50.276] rejected tx                              index=0 hash=ccc996..d83435 from=0xa94f5374Fce5edBC8E2a8697C15331677e6EbF0B error="insufficient funds for gas * price + value: address 0xa94f5374Fce5edBC8E2a8697C15331677e6EbF0B have 84000000 want 84000032"
INFO [07-21|19:03:50.276] Trie dumping started                     root=e05f81..6597a5
INFO [07-21|19:03:50.276] Trie dumping complete                    accounts=1 elapsed="39.549µs"
{
 "alloc": {
  "0xa94f5374fce5edbc8e2a8697c15331677e6ebf0b": {
   "balance": "0x501bd00"
  }
 },
 "result": {
  "stateRoot": "0xe05f81f8244a76503ceec6f88abfcd03047a612a1001217f37d30984536597a5",
  "txRoot": "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
  "receiptRoot": "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
  "logsHash": "0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347",
  "logsBloom": "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
  "receipts": [],
  "rejected": [
   {
    "index": 0,
    "error": "insufficient funds for gas * price + value: address 0xa94f5374Fce5edBC8E2a8697C15331677e6EbF0B have 84000000 want 84000032"
   }
  ]
 }
}
```

The transaction is rejected. 