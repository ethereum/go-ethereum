const solc = require('solc');
const fs = require('fs');
const path = require('path');

// ✅ Path to the Solidity contract
const contractPath = path.resolve(__dirname, 'contracts', 'Lock.sol');

// ✅ Read contract source
const source = fs.readFileSync(contractPath, 'utf8');

// Solidity compiler input format
const input = {
  language: 'Solidity',
  sources: {
    'Lock.sol': {
      content: source
    }
  },
  settings: {
    outputSelection: {
      '*': {
        '*': ['abi', 'evm.bytecode.object']
      }
    }
  }
};

const output = JSON.parse(solc.compile(JSON.stringify(input)));
const contract = output.contracts['Lock.sol']['Lock'];

// Save ABI and bytecode
fs.writeFileSync('Lock.abi.json', JSON.stringify(contract.abi, null, 2));
fs.writeFileSync('Lock.bytecode', contract.evm.bytecode.object);

console.log('✅ Contract compiled. ABI and bytecode saved.');

