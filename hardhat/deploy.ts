import { ethers } from "hardhat";

async function main() {
  const ContractFactory = await ethers.getContractFactory("YourContractName");
  const contract = await ContractFactory.deploy();
  await contract.deployed();

  console.log(`Deployed to: ${contract.address}`);
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
