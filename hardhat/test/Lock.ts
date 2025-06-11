import { expect } from "chai";
import hre from "hardhat";

describe("Lock", function () {
  const ONE_YEAR_IN_SECS = 365 * 24 * 60 * 60;
  const ONE_GWEI = 1_000_000_000;

  let lock: any;
  let unlockTime: number;
  let lockedAmount: number;
  let owner: any;
  let otherAccount: any;

  beforeEach(async function () {
    // Use block.timestamp + ONE_YEAR_IN_SECS
    const latestBlock = await hre.ethers.provider.getBlock("latest");
    unlockTime = latestBlock.timestamp + ONE_YEAR_IN_SECS;
    lockedAmount = ONE_GWEI;

    [owner, otherAccount] = await hre.ethers.getSigners();

    const Lock = await hre.ethers.getContractFactory("Lock");
    lock = await Lock.connect(owner).deploy(unlockTime, {
      value: lockedAmount,
    });
    await lock.waitForDeployment();
  });

  it("Should set the right unlockTime", async function () {
    expect(await lock.unlockTime()).to.equal(unlockTime);
  });

  it("Should set the right owner", async function () {
    expect(await lock.owner()).to.equal(owner.address);
  });

  it("Should receive and store the funds to lock", async function () {
    const balance = await hre.ethers.provider.getBalance(lock.target);
    expect(balance).to.equal(lockedAmount);
  });

  it("Should fail if the unlockTime is not in the future", async function () {
    const latestBlock = await hre.ethers.provider.getBlock("latest");
    const Lock = await hre.ethers.getContractFactory("Lock");
    await expect(
      Lock.deploy(latestBlock.timestamp, { value: 1 })
    ).to.be.revertedWith("Unlock time should be in the future");
  });

  it("Should revert with the right error if called too soon", async function () {
    await expect(lock.withdraw()).to.be.revertedWith("You can't withdraw yet");
  });

  it("Should revert if called from another account", async function () {
    // simulate time passing (skip in real Geth, or deploy with unlockTime already in past)
    // For now, assume unlockTime is satisfied
    const fakeUnlockTime = unlockTime - ONE_YEAR_IN_SECS - 1;
    const Lock = await hre.ethers.getContractFactory("Lock");
    const lockNow = await Lock.connect(owner).deploy(fakeUnlockTime, {
      value: lockedAmount,
    });
    await lockNow.waitForDeployment();

    await expect(lockNow.connect(otherAccount).withdraw()).to.be.revertedWith(
      "You aren't the owner"
    );
  });

  it("Should not fail if unlockTime has passed and owner calls it", async function () {
    // simulate a contract deployed with past unlock time
    const pastTime = unlockTime - ONE_YEAR_IN_SECS - 1;
    const Lock = await hre.ethers.getContractFactory("Lock");
    const lockNow = await Lock.connect(owner).deploy(pastTime, {
      value: lockedAmount,
    });
    await lockNow.waitForDeployment();

    await expect(lockNow.withdraw()).not.to.be.reverted;
  });
});
