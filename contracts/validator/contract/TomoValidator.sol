pragma solidity ^0.4.21;

import "./interfaces/IValidator.sol";
import "./libs/SafeMath.sol";

contract TomoValidator is IValidator {
    using SafeMath for uint256;

    struct ValidatorState {
        bool isValidator;
        bool isCandidate;
        uint256 cap;
    }

    mapping(address => ValidatorState) validatorsState;
    uint256 threshold = 1000 * 10** 18; // 1000 TOMO

    function TomoValidator(address[] _validators, uint256[] _caps) public {

        for (uint256 i = 0; i < _validators.length; i++) {
            validatorsState[_validators[i]] = ValidatorState({
                isValidator: true,
                isCandidate: true,
                cap: _caps[i]
            });
        }

    }

    function propose(address _candidate) external payable {
        // only validator can propose a candidate
        require(validatorsState[msg.sender].isValidator);
        validatorsState[_candidate] = ValidatorState({
            isValidator: false,
            isCandidate: true,
            cap: msg.value
        });
        
    }

    function vote(address _candidate) public payable {
        // only vote for candidate proposed by a validator
        require(validatorsState[_candidate].isCandidate);
        validatorsState[_candidate].cap = validatorsState[_candidate].cap.add(msg.value);
        if (validatorsState[_candidate].cap >= threshold) {
            validatorsState[_candidate].isValidator = true;
        }
    }
}
