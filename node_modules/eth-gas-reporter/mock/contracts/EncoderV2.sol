pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

contract EncoderV2 {
  uint id;

  struct Asset {
    uint a;
    uint b;
    string c;
  }

  Asset a;

  function setAsset44(uint _id, Asset memory _a) public {
    id = _id;
    a = _a;
  }

  function getAsset() public view returns (Asset memory) {
    return a;
  }
}
