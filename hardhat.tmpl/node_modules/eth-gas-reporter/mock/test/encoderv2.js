var EncoderV2 = artifacts.require("./EncoderV2.sol");

contract("EncoderV2", function(accounts) {
  let instance;

  beforeEach(async function() {
    instance = await EncoderV2.new();
  });

  it("should get & set an Asset with a struct", async function() {
    const asset = {
      a: "5",
      b: "7",
      c: "wowshuxkluh"
    };

    await instance.setAsset44("44", asset);
    const _asset = await instance.getAsset();

    assert.equal(_asset.a, asset.a);
    assert.equal(_asset.b, asset.b);
    assert.equal(_asset.c, asset.c);
  });
});
