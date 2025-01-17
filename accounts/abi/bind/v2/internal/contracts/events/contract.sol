// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

contract C {
    event basic1(
        uint256 indexed id,
        uint256 data
    );
    event basic2(
        bool indexed flag,
        uint256 data
    );

    function EmitOne() public {
        emit basic1(
            uint256(1),
            uint256(2));
    }

    // emit multiple events, different types
    function EmitMulti() public {
        emit basic1(
            uint256(1),
            uint256(2));
        emit basic1(
            uint256(3),
            uint256(4));
        emit basic2(
            false,
            uint256(1));
    }

    constructor() {
        // do something with these
    }
}