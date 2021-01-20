## EIP-1559 testing

This test contains testcases for EIP-1559, which uses an new transaction type and has a new block parameter. 

### Prestate

The alloc portion contains one contract (`0x000000000000000000000000000000000000aaaa`), containing the 
following code: `0x58585454`: `PC; PC; SLOAD; SLOAD`.

Essentialy, this contract does `SLOAD(0)` and `SLOAD(1)`.

The alloc also contains some funds on `0xa94f5374fce5edbc8e2a8697c15331677e6ebf0b`. 

## Transactions

There are two transactions, each invokes the contract above. 

1. EIP-1559 ACL-transaction, which contains the `0x0` slot for `0xaaaa`
2. Legacy transaction

## Execution 

Running it yields: 
```
$ dir=./testdata/9 && ./evm t8n --state.fork=Aleut --input.alloc=$dir/alloc.json --input.txs=$dir/txs.json --input.env=$dir/env.json --trace && cat trace-* | grep SLOAD
{"pc":2,"op":84,"gas":"0x48c28","gasCost":"0x834","memory":"0x","memSize":0,"stack":["0x0","0x1"],"returnStack":null,"returnD
ata":"0x","depth":1,"refund":0,"opName":"SLOAD","error":""}
{"pc":3,"op":84,"gas":"0x483f4","gasCost":"0x64","memory":"0x","memSize":0,"stack":["0x0","0x0"],"returnStack":null,"returnDa
ta":"0x","depth":1,"refund":0,"opName":"SLOAD","error":""}
{"pc":2,"op":84,"gas":"0x49cf4","gasCost":"0x834","memory":"0x","memSize":0,"stack":["0x0","0x1"],"returnStack":null,"returnD
ata":"0x","depth":1,"refund":0,"opName":"SLOAD","error":""}
{"pc":3,"op":84,"gas":"0x494c0","gasCost":"0x834","memory":"0x","memSize":0,"stack":["0x0","0x0"],"returnStack":null,"returnD
ata":"0x","depth":1,"refund":0,"opName":"SLOAD","error":""}
```

We can also get the post-alloc:
```
$ dir=./testdata/9 && ./evm t8n --state.fork=Aleut --input.alloc=$dir/alloc.json --input.txs=$dir/txs.json --input.env=$dir/env.json --output.alloc=stdout
{
 "alloc": {
  "0x000000000000000000000000000000000000aaaa": {
   "code": "0x58585454",
   "balance": "0x3",
   "nonce": "0x1"
  },
  "0x2adc25665018aa1fe0e6bc666dac8fc2697ff9ba": {
   "balance": "0xbfc02677a000"
  },
  "0xa94f5374fce5edbc8e2a8697c15331677e6ebf0b": {
   "balance": "0xff104fcfea7800",
   "nonce": "0x2"
  }
 }
}
```

If we try to execute it on older rules: 
```
dir=./testdata/9 && ./evm t8n --state.fork=Berlin --input.alloc=$dir/alloc.json --input.txs=$dir/txs.json --input.env=$dir/env.json 
INFO [04-28|22:05:19.339] rejected tx                              index=0 hash=9a5bbc..d8cc42 from=0xa94f5374Fce5edBC8E2a8697C15331677e6EbF0B error="baseFee before fork block"
INFO [04-28|22:05:19.339] rejected tx                              index=1 hash=a9c6c6..fa4036 from=0xa94f5374Fce5edBC8E2a8697C15331677e6EbF0B error="nonce too high: address 0xa94f5374Fce5edBC8E2a8697C15331677e6EbF0B, tx: 1 state: 0"
```
Number `0` is not applicable, and therefore number `1` has wrong nonce. 
