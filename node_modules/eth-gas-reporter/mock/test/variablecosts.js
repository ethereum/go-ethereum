const random = require("./random");
const VariableCosts = artifacts.require("./VariableCosts.sol");
const Wallet = artifacts.require("./Wallet.sol");

contract("VariableCosts", accounts => {
  const one = [1];
  const three = [2, 3, 4];
  const five = [5, 6, 7, 8, 9];
  let instance;
  let walletB;

  beforeEach(async () => {
    instance = await VariableCosts.new();
    walletB = await Wallet.new();
  });

  it("should add one", async () => {
    await instance.addToMap(one);
  });

  it("should add three", async () => {
    await instance.addToMap(three);
  });

  it("should add even 5!", async () => {
    await instance.addToMap(five);
  });

  it("should delete one", async () => {
    await instance.removeFromMap(one);
  });

  it("should delete three", async () => {
    await instance.removeFromMap(three);
  });

  it("should delete five", async () => {
    await instance.removeFromMap(five);
  });

  it("should add five and delete one", async () => {
    await instance.addToMap(five);
    await instance.removeFromMap(one);
  });

  it("should set a random length string", async () => {
    await instance.setString(random());
    await instance.setString(random());
    await instance.setString(random());
  });

  it("methods that do not throw", async () => {
    await instance.methodThatThrows(false);
  });

  it("methods that throw", async () => {
    try {
      await instance.methodThatThrows(true);
    } catch (e) {}
  });

  it("methods that call methods in other contracts", async () => {
    await instance.otherContractMethod();
  });

  it("prints a table at end of test suites with failures", async () => {
    assert(false);
  });

  // VariableCosts is Wallet. We also have Wallet tests. So we should see
  // separate entries for `sendPayment` / `transferPayment` under VariableCosts
  // and Wallet in the report
  it("should allow contracts to have identically named methods", async () => {
    await instance.sendTransaction({
      value: 100,
      from: accounts[0]
    });
    await instance.sendPayment(50, walletB.address, {
      from: accounts[0]
    });
    await instance.transferPayment(50, walletB.address, {
      from: accounts[0]
    });
    const balance = await walletB.getBalance();
    assert.equal(parseInt(balance.toString()), 100);
  });
});
