# prestateTracer Behavior Test

This directory contains a simple test case to demonstrate the behavior of the prestateTracer when accessing account balance.

The test shows that when a contract calls `address.balance`, the prestateTracer will by default include the code of the target address in the trace output, even though only the balance field is accessed. This is the expected behavior of the tracer, but can be modified using the `disableCode` configuration option.

## Files

- `BalanceReader.sol`: A simple contract that reads an external account's balance
- `test_commands.sh`: Shell commands to deploy and test the contract

## Expected Results

1. Without `disableCode`: The trace includes the full account state including code
2. With `disableCode: true`: The trace includes only balance and nonce, but no code
