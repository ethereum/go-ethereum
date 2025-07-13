const { ethers } = require("hardhat");

async function main() {
  const [deployer] = await ethers.getSigners();

  console.log("Deploying contracts with the account:", deployer.address);

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