import { ethers } from "hardhat";

async function main() {
  const unlockTime = Math.floor(Date.now() / 1000) + 60;
  const lockedAmount = ethers.parseEther("0.01");

  const Lock = await ethers.getContractFactory("Lock");
  const lock = await Lock.deploy(unlockTime, { value: lockedAmount });

  await lock.waitForDeployment();
  const lockAddress = await lock.getAddress(); // Use getAddress() for type safety

  console.log(
    `Lock with ${ethers.formatEther(
      await ethers.provider.getBalance(lockAddress)
    )} ETH and unlock timestamp ${unlockTime} deployed to ${lockAddress}`
  );
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});