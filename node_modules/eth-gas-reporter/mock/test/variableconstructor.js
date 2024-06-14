const random = require("./random");
const VariableConstructor = artifacts.require("./VariableConstructor.sol");

contract("VariableConstructor", accounts => {
  it("should should initialize with a short string", async () => {
    await VariableConstructor.new("Exit Visa");
  });

  it("should should initialize with a medium length string", async () => {
    await VariableConstructor.new("Enclosed is my application for residency");
  });

  it("should should initialize with a long string", async () => {
    let msg =
      "Enclosed is my application for permanent residency in NewZealand.";
    msg += "I am a computer programmer.";
    await VariableConstructor.new(msg);
  });

  it("should should initialize with a random length string", async () => {
    await VariableConstructor.new(random());
  });
});
