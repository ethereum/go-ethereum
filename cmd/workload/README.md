## Workload Testing Tool

This tool performs RPC calls against a live node. It has tests for the Sepolia testnet and
Mainnet.

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

### Regenerating tests

There are also commands for generating tests. To create filter tests, run:

```shell
> ./workload filtergen http://host:8545
```
