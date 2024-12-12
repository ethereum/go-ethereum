// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

error BadThing(uint256 arg1, uint256 arg2, uint256 arg3, bool arg4);
error BadThing2(uint256 arg1, uint256 arg2, uint256 arg3, uint256 arg4);

contract C {
    function Foo() public pure {
        revert BadThing({
            arg1: uint256(0),
            arg2: uint256(1),
            arg3: uint256(2),
            arg4: false
        });
    }
    function Bar() public pure {
        revert BadThing2({
            arg1: uint256(0),
            arg2: uint256(1),
            arg3: uint256(2),
            arg4: uint256(3)
        });
    }
}

// purpose of this is to test that generation of metadata for contract that emits one error produces valid Go code
contract C2 {
    function Foo() public pure {
        revert BadThing({
            arg1: uint256(0),
            arg2: uint256(1),
            arg3: uint256(2),
            arg4: false
        });
    }
}