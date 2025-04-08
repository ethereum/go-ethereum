const { ethers } = require("ethers");

// Create a random wallet
const wallet = ethers.Wallet.createRandom();

console.log("Private Key:", wallet.privateKey);
console.log("Address:", wallet.address);