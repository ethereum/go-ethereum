pragma solidity ^0.4.21;

import "./interfaces/IValidator.sol";
import "./libs/SafeMath.sol";

contract TomoValidator is IValidator {
    using SafeMath for uint256;

    struct ValidatorState {
        bool isValidator;
        bool isCandidate;
        uint256 cap;
        mapping(address => uint256) voters;
    }

    mapping(address => ValidatorState) validatorsState;
    address[] public validators;
    address[] public candidates;
    uint256 threshold = 1000 * 10** 18; // 1000 TOMO

    function TomoValidator(address[] _validators, uint256[] _caps) public {
        validators = _validators;
        candidates = _validators;
        
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
        if (!validatorsState[_candidate].isCandidate) {
            candidates.push(_candidate);
        }
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
            if (!validatorsState[_candidate].isValidator) {
                validators.push(_candidate);
            }
            validatorsState[_candidate].isValidator = true;
            validatorsState[_candidate].voters[msg.sender] = validatorsState[_candidate].voters[msg.sender].add(msg.value);
        }
    }

    function getValidators() public view returns(address[]) {
       return validators;
    }

    function getCandidates() public view returns(address[]) {
       return candidates;
    }

    function getCandidateCap(address _candidate) public view returns(uint256) {
        return validatorsState[_candidate].cap;
    }

    function getVoterCap(address _candidate, address _voter) public view returns(uint256) {
        return validatorsState[_candidate].voters[_voter];
    }

    function isValidator(address _candidate) public view returns(bool) {
       return validatorsState[_candidate].isValidator;
    }

    function isCandidate(address _candidate) public view returns(bool) {
       return validatorsState[_candidate].isCandidate;
    }

    function unvote(address _candidate, uint256 _cap) public {
        // only unvote for candidate who does not become validator yet
        require(!validatorsState[_candidate].isValidator);
        require(validatorsState[_candidate].isCandidate);
        require(validatorsState[_candidate].voters[msg.sender] >= _cap);
        validatorsState[_candidate].cap = validatorsState[_candidate].cap.sub(_cap);
        validatorsState[_candidate].voters[msg.sender] = validatorsState[_candidate].voters[msg.sender].sub(_cap);
        // refunding to user after unvoting
        msg.sender.transfer(_cap);
    }
}
