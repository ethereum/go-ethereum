pragma solidity ^0.6.0;
contract Test {
    address x;
}
contract TestOverride is Test {
// This declaration would throw an error because its shadowing a
// state variable already in its base inherited contract Test
    address x;
}
