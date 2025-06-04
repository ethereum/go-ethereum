const { expect } = require("chai");
const fs = require("fs");

describe("Lock (deployed contract)", function () {
  let lock;

  before(async () => {
    const { address } = JSON.parse(fs.readFileSync("deployment-output.json", "utf-8"));
    const Lock = await ethers.getContractFactory("Lock");
    lock = Lock.attach(address);
  });

  it("Should read unlock time", async () => {
    const unlockTime = await lock.unlockTime();
    expect(unlockTime).to.be.a("bigint");
  });
});
