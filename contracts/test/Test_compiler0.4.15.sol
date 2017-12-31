pragma solidity ^0.4.15;

/**
 * @title Basic token
 * @dev Basic version of CLO_test
 */
contract CLO_test {
    string public STR = "Test Callisto NETwork";
    address public owner;
    
    function CLO_test()
    {
        owner = msg.sender;
    }
    
    function TEST_STR() constant returns (string)
    {
        return STR;
    }
    
    function SetOwner(address _owner)
    {
        assert(msg.sender == owner);
        owner = _owner;
    }
    
    function SUM(uint256 _first, uint256 _second) constant returns (uint256)
    {
        return (_first + _second);
    }
    
    function SetSTR(string _STR)
    {
        assert(msg.sender == owner);
        STR = _STR;
    }
 }
