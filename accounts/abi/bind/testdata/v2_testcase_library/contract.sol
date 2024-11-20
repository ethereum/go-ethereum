// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

library RecursiveDep {
    function AddOne(uint256 val) public pure returns (uint256 ret)  {
        return val + 1;
    }
}

// Array function to delete element at index and re-organize the array
// so that there are no gaps between the elements.
library Array {
    using RecursiveDep for uint256;

    function remove(uint256[] storage arr, uint256 index) public {
        // Move the last element into the place to delete
        require(arr.length > 0, "Can't remove from empty array");
        arr[index] = arr[arr.length - 1];
        arr[index] = arr[index].AddOne();
        arr.pop();
    }
}

contract TestArray {
    using Array for uint256[];

    uint256[] public arr;

    function testArrayRemove(uint256 value) public {
        for (uint256 i = 0; i < 3; i++) {
            arr.push(i);
        }

        arr.remove(1);

        assert(arr.length == 2);
        assert(arr[0] == 0);
        assert(arr[1] == 2);
    }

    constructor(uint256 foobar) {

    }
}
