const Web3 = require('web3');
const fs = require('fs');

// Load the keystore file
const keyfile = fs.readFileSync('Full path to JSON key file', 'utf-8');

// Your password
const password = 'password';

const web3 = new Web3();
const account = web3.eth.accounts.decrypt(JSON.parse(keyfile), password);

// Print the private key
console.log('Private Key:', account.privateKey);
