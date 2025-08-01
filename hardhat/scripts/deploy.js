const { ethers } = require("hardhat");

async function main() {
  const [deployer] = await ethers.getSigners();

  console.log("Deploying contracts with the account:", deployer.address);

  const unlockTime = Math.floor(Date.now() / 1000) + 3600; // 1 hour from now

  const Lock = await ethers.getContractFactory("Lock", deployer);
  const lock = await Lock.deploy(unlockTime);

  await lock.waitForDeployment();

  console.log("Lock deployed to:", lock.address);
}

main()
  .then(() => process.exit(0))
  .catch(error => {
    console.error("Deployment failed:", error);
    process.exit(1);
  });