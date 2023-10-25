// SPDX-License-Identifier: MIT
pragma solidity ^0.8.18;

contract AlgorandInfo {
    event Log(string message, uint256 value);

    enum CmdType {
        AccountCmd
    }

    function getAccountBalance(
        string memory accountAddress
    ) public returns (uint256) {
        (bool ok, bytes memory data) = address(0xff).call(
            abi.encode(CmdType.AccountCmd, "Amount", accountAddress)
        );
        require(ok, "failed to get account balance");
        uint256 balance = abi.decode(data, (uint256));
        emit Log("balance", balance);
        return balance;
    }
}
