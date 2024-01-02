pragma solidity ^0.8.0;

contract ExampleSHA256 {
    function hashSha256(uint256 numberToHash) public view returns (bytes32 h) {
        (bool ok, bytes memory out) = address(2).staticcall(abi.encode(numberToHash));
        require(ok, "precompile call failed");
        h = abi.decode(out, (bytes32));
    }
}
