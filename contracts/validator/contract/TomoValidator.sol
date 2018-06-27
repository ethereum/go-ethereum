pragma solidity ^0.4.21;

import "./interfaces/IValidator.sol";
import "./libs/SafeMath.sol";

contract TomoValidator is IValidator {
    using SafeMath for uint256;

    event Vote(address _voter, address _candidate, uint256 _cap);
    event Unvote(address _voter, address _candidate, uint256 _cap);
    event Propose(address _owner, address _candidate, uint256 _cap);
    event Resign(address _owner, address _candidate);
    event SetNodeUrl(address _owner, address _candidate, string _nodeUrl);
    event Withdraw(address _owner, address _candidate, uint256 _cap);

    struct ValidatorState {
        address owner;
        string nodeUrl;
        bool isCandidate;
        uint256 cap;
        uint256 withdrawBlockNumber;
        mapping(address => uint256) voters;
    }

    mapping(address => ValidatorState) validatorsState;
    mapping(address => address[]) voters;
    address[] public candidates = [
        0xf99805B536609cC03AcBB2604dFaC11E9E54a448,
        0x31b249fE6F267aa2396Eb2DC36E9c79351d97Ec5,
        0xfC5571921c6d3672e13B58EA23DEA534f2b35fA0
    ];
    uint256 candidateCount = 3;
    uint256 public minCandidateCap;
    uint256 public maxValidatorNumber;
    uint256 public candidateWithdrawDelay; // blocks

    modifier onlyValidCandidateCap {
        // anyone can deposit X TOMO to become a candidate
        require(msg.value >= minCandidateCap);
        _;
    }

    modifier onlyOwner(address _candidate) {
        require(validatorsState[_candidate].owner == msg.sender);
        _;
    }

    modifier onlyCandidate(address _candidate) {
        require(validatorsState[_candidate].isCandidate);
        _;
    }

    modifier onlyAlreadyResigned(address _candidate) {
        require(validatorsState[_candidate].withdrawBlockNumber > 0);
        require(block.number >= validatorsState[_candidate].withdrawBlockNumber);
        _;
    }

    modifier onlyValidCandidate (address _candidate) {
        require(validatorsState[_candidate].isCandidate);
        _;
    }

    modifier onlyNotCandidate (address _candidate) {
        require(!validatorsState[_candidate].isCandidate);
        _;
    }

    modifier onlyValidVote (address _candidate, uint256 _cap) {
        require(validatorsState[_candidate].voters[msg.sender] >= _cap);
        _;
    }

    function TomoValidator (
        uint256 _minCandidateCap,
        uint256 _maxValidatorNumber,
        uint256 _candidateWithdrawDelay
    ) public {
        minCandidateCap = _minCandidateCap;
        maxValidatorNumber = _maxValidatorNumber;
        candidateWithdrawDelay = _candidateWithdrawDelay;

        for (uint256 i = 0; i < candidates.length; i++) {
            validatorsState[candidates[i]] = ValidatorState({
                owner: 0x487d62d33467c4842c5e54Eb370837E4E88BBA0F,
                nodeUrl: '',
                isCandidate: true,
                withdrawBlockNumber: 0,
                cap: minCandidateCap
            });
        }
    }

    function propose(address _candidate, string _nodeUrl) external payable onlyValidCandidateCap onlyNotCandidate(_candidate) {
        candidates.push(_candidate);
        validatorsState[_candidate] = ValidatorState({
            owner: msg.sender,
            nodeUrl: _nodeUrl,
            isCandidate: true,
            withdrawBlockNumber: 0,
            cap: msg.value
        });
        validatorsState[_candidate].voters[msg.sender] = msg.value;
        candidateCount = candidateCount + 1;
        emit Propose(msg.sender, _candidate, msg.value);
    }

    function vote(address _candidate) external payable onlyValidCandidate(_candidate) {
        validatorsState[_candidate].cap = validatorsState[_candidate].cap.add(msg.value);
        if (validatorsState[_candidate].voters[msg.sender] == 0) {
            voters[_candidate].push(msg.sender);
        }
        validatorsState[_candidate].voters[msg.sender] = validatorsState[_candidate].voters[msg.sender].add(msg.value);
        emit Vote(msg.sender, _candidate, msg.value);
    }

    function getCandidates() public view returns(address[]) {
        return candidates;
    }

    function getCandidateCap(address _candidate) public view returns(uint256) {
        return validatorsState[_candidate].cap;
    }

    function getCandidateNodeUrl(address _candidate) public view returns(string) {
        return validatorsState[_candidate].nodeUrl;
    }

    function getCandidateOwner(address _candidate) public view returns(address) {
        return validatorsState[_candidate].owner;
    }

    function getCandidateWithdrawBlockNumber(address _candidate) public view returns(uint256) {
        return validatorsState[_candidate].withdrawBlockNumber;
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

    function unvote(address _candidate, uint256 _cap) public onlyValidVote(_candidate, _cap) {
        validatorsState[_candidate].cap = validatorsState[_candidate].cap.sub(_cap);
        validatorsState[_candidate].voters[msg.sender] = validatorsState[_candidate].voters[msg.sender].sub(_cap);
        // refunding to user after unvoting
        msg.sender.transfer(_cap);
        emit Unvote(msg.sender, _candidate, _cap);
    }

    function setNodeUrl(address _candidate, string _nodeUrl) public onlyOwner(_candidate) {
        validatorsState[_candidate].nodeUrl = _nodeUrl;
        emit SetNodeUrl(msg.sender, _candidate, _nodeUrl);
    }

    function resign(address _candidate) public onlyOwner(_candidate) onlyCandidate(_candidate) {
        validatorsState[_candidate].isCandidate = false;
        candidateCount = candidateCount - 1;
        for (uint256 i = 0; i < candidates.length; i++) {
            if (candidates[i] == _candidate) {
                delete candidates[i];
                break;
            }
        }
        // refunding after retiring X blocks
        validatorsState[_candidate].withdrawBlockNumber = validatorsState[_candidate].withdrawBlockNumber.add(block.number).add(candidateWithdrawDelay);
        emit Resign(msg.sender, _candidate);
    }

    function withdraw(address _candidate) public onlyOwner(_candidate) onlyNotCandidate(_candidate) onlyAlreadyResigned(_candidate) {
        uint256 cap = validatorsState[_candidate].voters[msg.sender];
        validatorsState[_candidate].cap = validatorsState[_candidate].cap.sub(cap);
        validatorsState[_candidate].voters[msg.sender] = 0;
        validatorsState[_candidate].withdrawBlockNumber = 0;
        msg.sender.transfer(cap);
        emit Withdraw(msg.sender, _candidate, cap);
    }
}
