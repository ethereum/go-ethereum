`npm install`
run `node deploy.js` to deploy the contract.

this will log out a contract in the geth logs.

set the ADDR environment variable to this contract address.  
ie `export ADDR=<contract_addr>`

then run `node calltx.js` 

(TODO: trigger the `calltx.js` function in the `deploy.js` file immediately)

## TEST GREETERS / contract to contract txes

First run `node deploy_greeter_contracts.js`

this will log the address of the greeter and proxygreeter contracts and run various transactions on the contracts.

To re-run transactions on these contracts you'll need to set the env variables, using the addresses logged during the `node deploy_greeter_contracts.js` process:

```
export GREETER=<greeter_address>
export PROXYGREETER=<proxy_greeter_address>
```

Then we can run:

`node call_greeter_fns.js`

This will run several write transactions, the hash will be logged to the geth logs.

To run trace transaction on these txes, run `./build/bin/geth  attach http://127.0.0.1:8545`, which will open an admin console (similar to a node console). Then run `debug.traceTransaction("<tx_hash>", {tracer: "callTracer"})`, or `debug.traceTransaction("<tx_hash>")`
