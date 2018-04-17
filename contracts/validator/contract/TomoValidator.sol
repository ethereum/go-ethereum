pragma solidity ^0.4.21;

import "./interfaces/IValidator.sol";
import "./libs/SafeMath.sol";

contract TomoValidator is IValidator {
    using SafeMath for uint256;

    struct ValidatorState {
        bool isCandidate;
        uint256 cap;
        mapping(address => uint256) voters;
    }

    mapping(address => ValidatorState) validatorsState;
    address[] public candidates;
    uint256 candidateCount = 0;

    function TomoValidator(address[] _candidates, uint256[] _caps) public {
        candidates = _candidates;
        
        for (uint256 i = 0; i < _candidates.length; i++) {
            validatorsState[_candidates[i]] = ValidatorState({
                isCandidate: true,
                cap: _caps[i]
            });
            candidateCount = candidateCount + 1;
        }

    }

    function propose(address _candidate) external payable {
        // TOMO: only validator can propose a candidate
        if (!validatorsState[_candidate].isCandidate) {
            candidates.push(_candidate);
        }
        validatorsState[_candidate] = ValidatorState({
            isCandidate: true,
            cap: msg.value
        });
        candidateCount = candidateCount + 1;
    }

    function vote(address _candidate) public payable {
        // only vote for candidate proposed by a validator
        require(validatorsState[_candidate].isCandidate);
        validatorsState[_candidate].cap = validatorsState[_candidate].cap.add(msg.value);
        validatorsState[_candidate].voters[msg.sender] = validatorsState[_candidate].voters[msg.sender].add(msg.value);
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

    function isCandidate(address _candidate) public view returns(bool) {
       return validatorsState[_candidate].isCandidate;
    }

    function unvote(address _candidate, uint256 _cap) public {
        require(validatorsState[_candidate].isCandidate);
        require(validatorsState[_candidate].voters[msg.sender] >= _cap);
        validatorsState[_candidate].cap = validatorsState[_candidate].cap.sub(_cap);
        validatorsState[_candidate].voters[msg.sender] = validatorsState[_candidate].voters[msg.sender].sub(_cap);
        // refunding to user after unvoting
        msg.sender.transfer(_cap);
    }
}
