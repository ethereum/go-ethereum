const EtherRouter = artifacts.require("EtherRouter");
const Resolver = artifacts.require("Resolver");
const Factory = artifacts.require("Factory");
const VersionA = artifacts.require("VersionA");
const VersionB = artifacts.require("VersionB");

contract("EtherRouter Proxy", accounts => {
  let router;
  let resolver;
  let factory;
  let versionA;
  let versionB;

  beforeEach(async function() {
    router = await EtherRouter.new();
    resolver = await Resolver.new();
    factory = await Factory.new();
    versionA = await VersionA.new();

    // Emulate internal deployment
    await factory.deployVersionB();
    const versionBAddress = await factory.versionB();
    versionB = await VersionB.at(versionBAddress);
  });

  it("Resolves methods routed through an EtherRouter proxy", async function() {
    let options = {
      from: accounts[0],
      gas: 4000000,
      to: router.address,
      gasPrice: 20000000000
    };

    await router.setResolver(resolver.address);

    await resolver.register("setValue()", versionA.address);
    options.data = versionA.contract.methods.setValue().encodeABI();
    await web3.eth.sendTransaction(options);

    await resolver.register("setValue()", versionB.address);
    options.data = versionB.contract.methods.setValue().encodeABI();
    await web3.eth.sendTransaction(options);
  });
});
