// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract BalanceReader {
    function getExternalBalance(address account) external view returns (uint256) {
        return account.balance;
    }
}
