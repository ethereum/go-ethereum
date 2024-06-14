pragma solidity ^0.5.0;

import "./Wallets/Wallet.sol";
import "./MultiContractFile.sol";

contract VariableCosts is Wallet {
  uint q;
  string someString;
  mapping(uint => address) map;
  MultiContractFileA multi;

  constructor() public {
    multi = new MultiContractFileA();
  }

  function pureFn(uint x) public pure returns (uint){
    return x;
  }

  function viewFn(uint x) public view returns (address){
    return map[x];
  }

  function constantFn(uint x) public view returns (address){
    return map[x];
  }

  function addToMap(uint[] memory adds) public {
    for(uint i = 0; i < adds.length; i++)
      map[adds[i]] = address(this);
  }

  function removeFromMap(uint[] memory dels) public {
    for(uint i = 0; i < dels.length; i++)
      delete map[dels[i]];
  }

  function unusedMethod(address a) public {
    map[1000] = a;
  }

  function setString(string memory _someString) public {
    someString = _someString;
  }

  function methodThatThrows(bool err) public {
    require(!err);
    q = 5;
  }

  function otherContractMethod() public {
    multi.hello(); // 20,000 gas (sets uint to 5 from zero)
    multi.hello(); //  5,000 gas (sets existing storage)
    multi.hello(); //  5,000 gas (sets existing storage)
  }
}
