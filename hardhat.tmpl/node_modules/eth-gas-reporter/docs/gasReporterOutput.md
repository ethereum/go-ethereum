### Gas reporter JSON output

A sample of the data written to file `gasReporterOutput.json` in the project directory root
when `eth-gas-reporter` is run with the environment variable `CI` set to true. You
can use this as an input to more complex or long running gas analyses, develop
CI integrations with it, make a nicer table, etc.

```json
{
 "namespace": "ethGasReporter",
 "config": {
  "blockLimit": 6718946,
  "currency": "eur",
  "ethPrice": "316.615237512",
  "gasPrice": 2,
  "outputFile": null,
  "rst": false,
  "rstTitle": "",
  "showTimeSpent": false,
  "srcPath": "contracts",
  "artifactType": "truffle-v5",
  "proxyResolver": null,
  "metadata": {
   "compiler": {
    "version": "0.5.0+commit.1d4f565a"
   },
   "settings": {
    "evmVersion": "byzantium",
    "optimizer": {
     "enabled": false,
     "runs": 200
    },
   },
  },
  "excludeContracts": [],
  "onlyCalledMethods": true,
  "url": "http://localhost:8545"
 },
 "info": {
  "methods": {
   "EtherRouter_4e543b26": {
    "key": "4e543b26",
    "contract": "EtherRouter",
    "method": "setResolver",
    "gasData": [
     43192
    ],
    "numberOfCalls": 1
   },
   "Resolver_1e59c529": {
    "key": "1e59c529",
    "contract": "Resolver",
    "method": "register",
    "gasData": [
     30133,
     45133
    ],
    "numberOfCalls": 2
   },
   ...
  },
  "deployments": [
   {
    "name": "ConvertLib",
    "bytecode": "0x60dd61002...",
    "deployedBytecode": "0x73000...",
    "gasData": [
     111791
    ]
   },
   {
    "name": "EtherRouter",
    "bytecode": "0x608060...",
    "deployedBytecode": "0x60806040...",
    "gasData": [
     278020
    ]
   },
   ...
  ],
 }
}
```
