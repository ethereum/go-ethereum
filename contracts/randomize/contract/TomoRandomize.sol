pragma solidity ^0.4.21;

import "./libs/SafeMath.sol";

contract TomoRandomize {
    using SafeMath for uint256;
    uint256 public randomNumber;

    mapping (address=>bytes32[]) randomSecret;
    mapping (address=>bytes32) randomOpening;

    function TomoRandomize (uint256 _randomNumber) public {
        randomNumber = _randomNumber;
    }

    function setSecret(bytes32[] _secret) public {
        randomSecret[msg.sender] = _secret;
    }

    function setOpening(bytes32 _opening) public {
        randomOpening[msg.sender] = _opening;
    }

    function getSecret(address _validator) public view returns(bytes32[]) {
        return randomSecret[_validator];
    }

    function getOpening(address _validator) public view returns(bytes32) {
        return randomOpening[_validator];
    }
}
