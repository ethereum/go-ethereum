pragma solidity ^0.6.0;
contract Test {
    string[] public names_A_to_F;
    function test() public {
        // invalid syntax
        uint length = names_A_to_F.push("Alice"); // invalid
        names_A_to_F.length++; // invalid as length is now read only
        // correct syntax
        names_A_to_F.push(); // increase array length
        names_A_to_F.push("Alice"); // add item to array
        names_A_to_F.pop(); // reduce array length by removing item
        uint length = names_A_to_F.length; // access array length
    }
}

