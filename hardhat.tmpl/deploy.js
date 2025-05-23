const Web3 = require('web3');
const fs = require('fs');

// Connect to local Geth devnet
const web3 = new Web3.default('http://localhost:8545');

// Load compiled contract
const abi = JSON.parse(fs.readFileSync('Lock.abi.json'));
const bytecode = fs.readFileSync('Lock.bytecode', 'utf8');

async function main() {
  const accounts = await web3.eth.getAccounts();
  const deployer = accounts[0];

  console.log('Deploying from account:', deployer);

  const unlockTime = Math.floor(Date.now() / 1000) + 60;

  const contract = new web3.eth.Contract(abi);
  const tx = contract.deploy({
    data: '0x' + bytecode,
    arguments: [unlockTime]
  });

  const deployed = await tx.send({
    from: deployer,
    value: web3.utils.toWei('1', 'ether'),
    gas: 3000000
  });

  console.log('Contract deployed at:', deployed.options.address);
}

main().catch(console.error);

