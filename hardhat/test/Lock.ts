import { expect } from "chai";
import hre from "hardhat";

describe("Lock", function () {
  const ONE_YEAR_IN_SECS = 365 * 24 * 60 * 60;
  const ONE_GWEI = 1_000_000_000;

  async function deployLockWithFutureUnlockTime() {
    const [owner, otherAccount] = await hre.ethers.getSigners();
    const block = await hre.ethers.provider.getBlock("latest");
    const unlockTime = block.timestamp + ONE_YEAR_IN_SECS;

    const Lock = await hre.ethers.getContractFactory("Lock");
    const lock = await Lock.deploy(unlockTime, { value: ONE_GWEI });
    await lock.waitForDeployment();

    return { lock, unlockTime, owner, otherAccount };
  }

  describe("Deployment", function () {
    it("Should set the right unlockTime", async function () {
      const { lock, unlockTime } = await deployLockWithFutureUnlockTime();
      expect(await lock.unlockTime()).to.equal(unlockTime);
    });

    it("Should set the right owner", async function () {
      const { lock, owner } = await deployLockWithFutureUnlockTime();
      expect(await lock.owner()).to.equal(owner.address);
    });

    it("Should receive and store the funds to lock", async function () {
      const { lock } = await deployLockWithFutureUnlockTime();
      const balance = await hre.ethers.provider.getBalance(await lock.getAddress());
      expect(balance).to.equal(ONE_GWEI);
    });

    it("Should fail if the unlockTime is not in the future", async function () {
      const block = await hre.ethers.provider.getBlock("latest");
      const currentTimestamp = block.timestamp;

      const Lock = await hre.ethers.getContractFactory("Lock");
      await expect(
        Lock.deploy(currentTimestamp, { value: ONE_GWEI })
      ).to.be.revertedWith("Unlock time should be in the future");
    });
  });

  describe("Withdrawals", function () {
    it("Should revert with the right error if called too soon", async function () {
      const { lock } = await deployLockWithFutureUnlockTime();
      await expect(lock.withdraw()).to.be.revertedWith("You can't withdraw yet");
    });

    it("Should revert with the right error if called from another account", async function () {
      const { unlockTime, otherAccount } = await deployLockWithFutureUnlockTime();

      const Lock = await hre.ethers.getContractFactory("Lock");
      const lock = await Lock.deploy(unlockTime, { value: ONE_GWEI });
      await lock.waitForDeployment();

      // Simulate time passing
      await new Promise((r) => setTimeout(r, 2000));

      await expect(lock.connect(otherAccount).withdraw()).to.be.revertedWith(
        "You aren't the owner"
      );
    });

    it("Should not fail if unlockTime has passed and owner calls it", async function () {
      const [owner] = await hre.ethers.getSigners();
      const block = await hre.ethers.provider.getBlock("latest");
      const unlockTime = block.timestamp + 1;

      const Lock = await hre.ethers.getContractFactory("Lock");
      const lock = await Lock.connect(owner).deploy(unlockTime, {
        value: ONE_GWEI,
      });
      await lock.waitForDeployment();

      // Wait 2 seconds to pass unlockTime
      await new Promise((r) => setTimeout(r, 2000));

      await expect(lock.withdraw()).not.to.be.reverted;
    });

    it("Should emit an event on withdrawals", async function () {
      const { lock, unlockTime } = await deployLockWithFutureUnlockTime();

      await new Promise((r) => setTimeout(r, 2000));

      await expect(lock.withdraw())
        .to.emit(lock, "Withdrawal")
        .withArgs(ONE_GWEI, anyValue); // `anyValue` works without Hardhat helpers
    });

    it("Should transfer the funds to the owner", async function () {
      const { lock, owner, unlockTime } = await deployLockWithFutureUnlockTime();

      const ownerBalanceBefore = await hre.ethers.provider.getBalance(owner.address);
      await new Promise((r) => setTimeout(r, 2000));

      const tx = await lock.withdraw();
      const receipt = await tx.wait();
      const gasUsed = receipt.gasUsed * tx.gasPrice;

      const ownerBalanceAfter = await hre.ethers.provider.getBalance(owner.address);

      expect(ownerBalanceAfter).to.be.closeTo(
        ownerBalanceBefore.add(ONE_GWEI).sub(gasUsed),
        ONE_GWEI // small margin
      );
    });
  });
});
