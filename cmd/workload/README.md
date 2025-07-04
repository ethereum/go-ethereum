## Workload Testing Tool

This tool performs RPC calls against a live node. It has tests for the Sepolia testnet and
Mainnet. Note the tests require a fully synced node.

To run the tests against a Sepolia node, use:

```shell
> ./workload test --sepolia http://host:8545
```

To run a specific test, use the `--run` flag to filter the test cases. Filtering works
similar to the `go test` command. For example, to run only tests for `eth_getBlockByHash`
and `eth_getBlockByNumber`, use this command:

```
> ./workload test --sepolia --run History/getBlockBy http://host:8545
```

Notably, trace tests require archive which keeps all the historical states for tracing.
The additional flag is required to activate the trace tests.

```
> ./workload test --sepolia --archive --run Trace/Block http://host:8545
```

### Regenerating tests

There is a facility for updating the tests from the chain. This can also be used to
generate the tests for a new network. As an example, to recreate tests for mainnet, run
the following commands (in this directory) against a synced mainnet node:

```shell
> go run . filtergen --queries queries/filter_queries_mainnet.json http://host:8545
> go run . historygen --history-tests queries/history_mainnet.json http://host:8545
> go run . tracegen --trace-tests queries/trace_mainnet.json --trace-start 4000000 --trace-end 4000100 http://host:8545
```
