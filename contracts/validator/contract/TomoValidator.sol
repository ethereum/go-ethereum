pragma solidity ^0.4.21;

import "./interfaces/IValidator.sol";
import "./libs/SafeMath.sol";

contract TomoValidator is IValidator {
    using SafeMath for uint256;

    event Vote(address _candidate, uint256 _cap);
    event Unvote(address _candidate, uint256 _cap);

    struct ValidatorState {
        bool isCandidate;
        uint256 cap;
        mapping(address => uint256) voters;
    }

    mapping(address => ValidatorState) validatorsState;
    address[] public candidates;
    uint256 candidateCount = 0;
    uint256 public constant minCandidateCap = 10000 ether;
    uint256 public constant maxCandidateNumber = 500;
    uint256 public constant maxValidatorNumber = 99;

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

    function propose() external payable {
        // anyone can deposit 10000 TOMO to become a candidate
        require(msg.value >= minCandidateCap);
        require(!validatorsState[msg.sender].isCandidate);
        require(candidateCount <= maxCandidateNumber);
        candidates.push(msg.sender);
        validatorsState[msg.sender] = ValidatorState({
            isCandidate: true,
            cap: msg.value
        });
        candidateCount = candidateCount + 1;
    }

    function vote(address _candidate) external payable {
        require(validatorsState[_candidate].isCandidate);
        validatorsState[_candidate].cap = validatorsState[_candidate].cap.add(msg.value);
        validatorsState[_candidate].voters[msg.sender] = validatorsState[_candidate].voters[msg.sender].add(msg.value);
        emit Vote(_candidate, msg.value);
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
        emit Unvote(_candidate, _cap);
    }
}
