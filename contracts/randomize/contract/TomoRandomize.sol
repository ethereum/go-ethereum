pragma solidity ^0.4.21;

import "./libs/SafeMath.sol";

contract TomoRandomize {
    using SafeMath for uint256;

    mapping (address=>bytes32[]) randomSecret;
    mapping (address=>bytes32) randomOpening;

    function TomoRandomize () public {
    }

    function setSecret(bytes32[] _secret) public {
        uint secretPoint =  block.number % 900;
        require(secretPoint >= 800);
        require(secretPoint < 850);
        randomSecret[msg.sender] = _secret;
    }

    function setOpening(bytes32 _opening) public {
        uint openingPoint =  block.number % 900;
        require(openingPoint >= 850);
        randomOpening[msg.sender] = _opening;
    }

    function getSecret(address _validator) public view returns(bytes32[]) {
        return randomSecret[_validator];
    }

    function getOpening(address _validator) public view returns(bytes32) {
        return randomOpening[_validator];
    }
}
