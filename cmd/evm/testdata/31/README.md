This test does some EVM execution, and can be used to test the tracers and trace-outputs.
This test should yield three output-traces, in separate files

For example:
```
[user@work evm]$ go  run . t8n --input.alloc ./testdata/31/alloc.json --input.txs ./testdata/31/txs.json --input.env ./testdata/31/env.json --state.fork Cancun --output.basedir /tmp --trace
INFO [12-06|09:53:32.123] Created tracing-file                     path=/tmp/trace-0-0x88f5fbd1524731a81e49f637aa847543268a5aaf2a6b32a69d2c6d978c45dcfb.jsonl
INFO [12-06|09:53:32.124] Created tracing-file                     path=/tmp/trace-1-0x03a7b0a91e61a170d64ea94b8263641ef5a8bbdb10ac69f466083a6789c77fb8.jsonl
INFO [12-06|09:53:32.125] Created tracing-file                     path=/tmp/trace-2-0xd96e0ce6418ee3360e11d3c7b6886f5a9a08f7ef183da72c23bb3b2374530128.jsonl
```

