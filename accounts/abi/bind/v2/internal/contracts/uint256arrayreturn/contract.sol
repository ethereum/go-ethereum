// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

contract MyContract {
    // emit multiple events, different types
    function GetNums() public pure returns (uint256[5] memory) {
        uint256[5] memory myNums = [uint256(0), uint256(1), uint256(2), uint256(3), uint256(4)];
        return myNums;
    }
}