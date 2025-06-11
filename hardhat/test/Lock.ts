import { expect } from "chai";
import hre from "hardhat";

describe("Lock (Geth-compatible)", function () {
  it("Deploys and allows withdrawal after unlockTime", async function () {
    const ONE_GWEI = 1_000_000_000;
    const unlockTime = Math.floor(Date.now() / 1000) + 2; // +2 sec from now

    const [owner] = await hre.ethers.getSigners();
    const Lock = await hre.ethers.getContractFactory("Lock");
    const lock = await Lock.deploy(unlockTime, { value: ONE_GWEI });

    // Wait 3 seconds to ensure unlockTime has passed
    await new Promise((res) => setTimeout(res, 3000));

    await expect(lock.withdraw()).to.not.be.reverted;
  });
});
