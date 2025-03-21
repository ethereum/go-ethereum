// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;


//  L1
//   \
//     L2  L3   L1
//      \ /    /
//       L4   /
//        \  /
//         C1
//
library L1 {
    function Do(uint256 val) public pure returns (uint256) {
        return uint256(1);
    }
}

library L2 {
    function Do(uint256 val) public pure returns (uint256) {
        return L1.Do(val) + uint256(1);
    }
}

library L3 {
    function Do(uint256 val) public pure returns (uint256) {
        return uint256(1);
    }
}

library L4 {
    function Do(uint256 val) public pure returns (uint256) {
        return L2.Do(uint256(val)) + L3.Do(uint256(val)) + uint256(1);
    }
}

contract C1 {
    function Do(uint256 val) public pure  returns (uint256 res) {
        return L4.Do(uint256(val)) + L1.Do(uint256(0)) + uint256(1);
    }

    constructor(uint256 v1, uint256 v2) {
        // do something with these
    }
}

// second contract+libraries: slightly different library deps than V1, but sharing several
//  L1
//   \
//     L2b  L3  L1
//      \  /   /
//       L4b  /
//        \  /
//         C2
//
library L4b {
        function Do(uint256 val) public pure returns (uint256) {
            return L2b.Do(uint256(val)) + uint256(1);
        }
}

library L2b {
        function Do(uint256 val) public pure returns (uint256) {
            return L1.Do(uint256(val)) + uint256(1);
        }
}

contract C2 {
    function Do(uint256 val) public pure  returns (uint256 res) {
        return L4b.Do(uint256(val)) + L1.Do(uint256(0)) + uint256(1);
    }

    constructor(uint256 v1, uint256 v2) {
        // do something with these
    }
}