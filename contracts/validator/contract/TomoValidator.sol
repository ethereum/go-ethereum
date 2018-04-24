pragma solidity ^0.4.21;

import "./interfaces/IValidator.sol";
import "./libs/SafeMath.sol";

contract TomoValidator is IValidator {
    using SafeMath for uint256;

    event Vote(address _candidate, uint256 _cap);
    event Unvote(address _candidate, uint256 _cap);
    event Propose(address _candidate, uint256 _cap);
    event Retire(address _candidate, uint256 _cap);

    struct ValidatorState {
        bool isCandidate;
        uint256 cap;
        mapping(address => uint256) voters;
    }

    mapping(address => ValidatorState) validatorsState;
    mapping(address => address[]) voters;
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
        validatorsState[msg.sender].voters[msg.sender] = msg.value;
        candidateCount = candidateCount + 1;
        emit Propose(msg.sender, msg.value);
    }

    function vote(address _candidate) external payable {
        require(validatorsState[_candidate].isCandidate);
        validatorsState[_candidate].cap = validatorsState[_candidate].cap.add(msg.value);
        validatorsState[_candidate].voters[msg.sender] = validatorsState[_candidate].voters[msg.sender].add(msg.value);
        voters[_candidate].push(msg.sender);
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

    function getVoters(address _candidate) public view returns(address[]) {
        return voters[_candidate];
    }

    function isCandidate(address _candidate) public view returns(bool) {
       return validatorsState[_candidate].isCandidate;
    }

    function unvote(address _candidate, uint256 _cap) public {
        require(validatorsState[_candidate].voters[msg.sender] >= _cap);
        validatorsState[_candidate].cap = validatorsState[_candidate].cap.sub(_cap);
        validatorsState[_candidate].voters[msg.sender] = validatorsState[_candidate].voters[msg.sender].sub(_cap);
        // refunding to user after unvoting
        msg.sender.transfer(_cap);
        emit Unvote(_candidate, _cap);
    }

    function retire() public {
        require(validatorsState[msg.sender].isCandidate);
        uint256 cap = validatorsState[msg.sender].voters[msg.sender];
        validatorsState[msg.sender].cap = validatorsState[msg.sender].cap.sub(cap);
        validatorsState[msg.sender].voters[msg.sender] = 0;
        validatorsState[msg.sender].isCandidate = false;
        candidateCount = candidateCount - 1;
        for (uint256 i = 0; i < candidates.length; i++) {
            if (candidates[i] == msg.sender) {
                delete candidates[i];
                break;
            }
        }
        // refunding to user after retiring
        msg.sender.transfer(cap);
        emit Retire(msg.sender, cap);
    }

}
