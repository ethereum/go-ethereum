pragma solidity ^0.4.0;

contract Resolver {
    function supportsInterface(bytes4 interfaceID) constant returns (bool);
    function addr(bytes32 node) constant returns (address ret);
    function dnsrr(bytes32 node, uint16 qtype, uint16 qclass, uint32 index)
        constant returns (uint16 rtype, uint16 rclass, bytes data);
}
