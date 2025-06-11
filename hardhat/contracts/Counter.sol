// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract Counter {
    uint256 public count;

    function incBy(uint256 value) public {
        count += value;
    }
}
