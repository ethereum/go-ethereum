pragma solidity ^0.4.21;

interface IValidator {
    function propose() external payable;
    function vote(address) external payable;
}
