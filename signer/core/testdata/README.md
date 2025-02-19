### EIP-712 tests

These tests are json files which are converted into [EIP-712](https://github.com/ethereum/EIPs/blob/master/EIPS/eip-712.md) typed data. 
All files are expected to be proper json, and tests will fail if they are not. 
Files that begin with `expfail' are expected to not pass the hashstruct construction. 
