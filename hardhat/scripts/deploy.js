const { ethers } = require("hardhat");

async function main() {
  // Connect to local devnet RPC explicitly
  const provider = new ethers.providers.JsonRpcProvider("http://localhost:8545");

  // Get signer from provider
  const [deployer] = await provider.listAccounts().then(accounts => 
    accounts.length > 0 ? [provider.getSigner(accounts[0])] : []
  );

  if (!deployer) {
    throw new Error("No accounts found on the devnet.");
  }

  console.log("Deploying contracts with the account:", await deployer.getAddress());

  // Connect contract factory with the deployer signer
  const Lock = await ethers.getContractFactory("Lock", deployer);

  const lock = await Lock.deploy();

  await lock.deployed();

  console.log("Lock deployed to:", lock.address);
}

main()
  .then(() => process.exit(0))
  .catch(error => {
    console.error("Deployment failed:", error);
    process.exit(1);
  });
