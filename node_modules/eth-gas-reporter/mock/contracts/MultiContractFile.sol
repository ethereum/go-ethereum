pragma solidity ^0.5.0;

contract MultiContractFileA {
  uint x;

  function hello() public {
    x = 5;
  }
}

contract MultiContractFileB {
  uint x;

  function goodbye() public {
    x = 5;
  }
}