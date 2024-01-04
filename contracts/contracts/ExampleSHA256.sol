pragma solidity ^0.8.0;

import "hardhat/console.sol";

contract ExampleSHA256 {
    function hashSha256(uint256 numberToHash) public view returns (bytes32 h) {
        (bool ok, bytes memory out) = address(2).staticcall(abi.encode(numberToHash));
        require(ok, "precompile call failed");

        console.logString("log out:");
        console.logBytes(out);

        h = abi.decode(out, (bytes32));
    }
}
