// SPDX-License-Identifier: GPL-3.0

pragma solidity >=0.8.2 <0.9.0;

/**
 * @title Storage
 * @dev Store & retrieve value in a variable
 * @custom:dev-run-script ./scripts/deploy_with_ethers.ts
 */
contract Example {

    uint256 number;
    struct exampleStruct {
        int val1;
        int val2;
        string val3;
    }

    event Basic(uint indexed firstArg, string secondArg);
    event Struct(uint indexed firstArg, exampleStruct secondArg);

    /**
     * @dev Store value in variable
     * @param num value to store
     */
    function mutateStorageVal(uint256 num) public {
        number = num;
    }

    /**
     * @dev Return value 
     * @return value of 'number'
     */
    function retrieveStorageVal() public view returns (uint256){
        return number;
    }

    function emitEvent() public {
        emit Basic(123, "event");
    }

    function emitTwoEvents() public {
        emit Basic(123, "event1");
        emit Basic(123, "event2");
    }

    function emitEventsDiffTypes() public {
        emit Basic(123, "event1");
        emit Struct(123, exampleStruct({val1: 1, val2: 2, val3: "string"}));
    }
}
