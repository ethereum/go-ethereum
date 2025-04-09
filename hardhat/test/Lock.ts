import { anyValue } from "@nomicfoundation/hardhat-chai-matchers/withArgs";
import { expect } from "chai";
import hre from "hardhat";

describe("Lock", function () {
  const ONE_YEAR_IN_SECS = 365 * 24 * 60 * 60;
  const ONE_GWEI = 1_000_000_000;
  const lockedAmount = ONE_GWEI;

  async function deployLock(unlockTime: number) {
    const Lock = await hre.ethers.getContractFactory("Lock");
    const lock = await Lock.deploy(unlockTime, { value: lockedAmount });
    return lock;
  }

  describe("Deployment", function () {
    it("Should set the right unlockTime", async function () {
      const unlockTime = Math.floor(Date.now() / 1000) + ONE_YEAR_IN_SECS;
      const lock = await deployLock(unlockTime);
      expect(await lock.unlockTime()).to.equal(unlockTime);
    });

    it("Should set the right owner", async function () {
      const unlockTime = Math.floor(Date.now() / 1000) + ONE_YEAR_IN_SECS;
      const [owner] = await hre.ethers.getSigners();
      const lock = await deployLock(unlockTime);
      expect(await lock.owner()).to.equal(owner.address);
    });

    it("Should receive and store the funds to lock", async function () {
      const unlockTime = Math.floor(Date.now() / 1000) + ONE_YEAR_IN_SECS;
      const lock = await deployLock(unlockTime);
      expect(await hre.ethers.provider.getBalance(lock.target)).to.equal(
        lockedAmount
      );
    });

    it("Should fail if the unlockTime is not in the future", async function () {
      const latestTime = Math.floor(Date.now() / 1000);
      await expect(deployLock(latestTime)).to.be.revertedWith(
        "Unlock time should be in the future"
      );
    });
  });

  describe("Withdrawals", function () {
    it("Should revert with the right error if called too soon", async function () {
      const unlockTime = Math.floor(Date.now() / 1000) + ONE_YEAR_IN_SECS;
      const lock = await deployLock(unlockTime);
      await expect(lock.withdraw()).to.be.revertedWith("You can't withdraw yet");
    });

    it("Should revert with the right error if called from another account", async function () {
      const unlockTime = Math.floor(Date.now() / 1000) + 10; // Unlock in 10 seconds
      const lock = await deployLock(unlockTime);
      const [, otherAccount] = await hre.ethers.getSigners();

      // Wait for unlock time
      await new Promise((resolve) => setTimeout(resolve, 11000));

      await expect(lock.connect(otherAccount).withdraw()).to.be.revertedWith(
        "You aren't the owner"
      );
    });

    it("Shouldn't fail if the unlockTime has arrived and the owner calls it", async function () {
      const unlockTime = Math.floor(Date.now() / 1000) + 10; // Unlock in 10 seconds
      const lock = await deployLock(unlockTime);

      // Wait for unlock time
      await new Promise((resolve) => setTimeout(resolve, 11000));

      await expect(lock.withdraw()).not.to.be.reverted;
    });

    it("Should emit an event on withdrawals", async function () {
      const unlockTime = Math.floor(Date.now() / 1000) + 10; // Unlock in 10 seconds
      const lock = await deployLock(unlockTime);
      const [owner] = await hre.ethers.getSigners();

      // Wait for unlock time
      await new Promise((resolve) => setTimeout(resolve, 11000));

      await expect(lock.withdraw())
        .to.emit(lock, "Withdrawal")
        .withArgs(lockedAmount, anyValue);
    });

    it("Should transfer the funds to the owner", async function () {
      const unlockTime = Math.floor(Date.now() / 1000) + 10; // Unlock in 10 seconds
      const lock = await deployLock(unlockTime);
      const [owner] = await hre.ethers.getSigners();

      // Wait for unlock time
      await new Promise((resolve) => setTimeout(resolve, 11000));

      await expect(lock.withdraw()).to.changeEtherBalances(
        [owner, lock],
        [lockedAmount, -lockedAmount]
      );
    });
  });
});
