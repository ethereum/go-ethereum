
pragma solidity ^0.4.21;

import "./libs/SafeMath.sol";


contract XDCValidator {
    using SafeMath for uint256;

    event Vote(address _voter, address _candidate, uint256 _cap);
    event Unvote(address _voter, address _candidate, uint256 _cap);
    event Propose(address _owner, address _candidate, uint256 _cap);
    event Resign(address _owner, address _candidate);
    event Withdraw(address _owner, uint256 _blockNumber, uint256 _cap);
    event UploadedKYC(address _owner,string kycHash);
    event InvalidatedNode(address _masternodeOwner, address[] _masternodes);

    struct ValidatorState {
        address owner;
        bool isCandidate;
        uint256 cap;
        mapping(address => uint256) voters;
    }

    struct WithdrawState {
      mapping(uint256 => uint256) caps;
      uint256[] blockNumbers;
    }

    mapping(address => WithdrawState) withdrawsState;

    mapping(address => ValidatorState) validatorsState;
    mapping(address => address[]) voters;

    // Mapping structures added for KYC feature.
    mapping(address => string[]) public KYCString;
    mapping(address => uint) public invalidKYCCount;
    mapping(address => mapping(address => bool)) public hasVotedInvalid;
    mapping(address => address[]) public ownerToCandidate;
    address[] public owners;

    address[] public candidates;

    uint256 public candidateCount = 0;
    uint256 public ownerCount =0;
    uint256 public minCandidateCap;
    uint256 public minVoterCap;
    uint256 public maxValidatorNumber;
    uint256 public candidateWithdrawDelay;
    uint256 public voterWithdrawDelay;

    modifier onlyValidCandidateCap {
        // anyone can deposit X XDC to become a candidate
        require(msg.value >= minCandidateCap);
        _;
    }

    modifier onlyValidVoterCap {

        require(msg.value >= minVoterCap);
        _;
    }

    modifier onlyKYCWhitelisted {
       require(KYCString[msg.sender].length!=0 || ownerToCandidate[msg.sender].length>0);
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
        if (validatorsState[_candidate].owner == msg.sender) {
            require(validatorsState[_candidate].voters[msg.sender].sub(_cap) >= minCandidateCap);
        }
        _;
    }

    modifier onlyValidWithdraw (uint256 _blockNumber, uint _index) {
        require(_blockNumber > 0);
        require(block.number >= _blockNumber);
        require(withdrawsState[msg.sender].caps[_blockNumber] > 0);
        require(withdrawsState[msg.sender].blockNumbers[_index] == _blockNumber);
        _;
    }

    function XDCValidator (
        address[] _candidates,
        uint256[] _caps,
        address _firstOwner,
        uint256 _minCandidateCap,
        uint256 _minVoterCap,
        uint256 _maxValidatorNumber,
        uint256 _candidateWithdrawDelay,
        uint256 _voterWithdrawDelay
    ) public {
        minCandidateCap = _minCandidateCap;
        minVoterCap = _minVoterCap;
        maxValidatorNumber = _maxValidatorNumber;
        candidateWithdrawDelay = _candidateWithdrawDelay;
        voterWithdrawDelay = _voterWithdrawDelay;
        candidateCount = _candidates.length;
        owners.push(_firstOwner);
        ownerCount++;
        for (uint256 i = 0; i < _candidates.length; i++) {
            candidates.push(_candidates[i]);
            validatorsState[_candidates[i]] = ValidatorState({
                owner: _firstOwner,
                isCandidate: true,
                cap: _caps[i]
            });
            voters[_candidates[i]].push(_firstOwner);
            ownerToCandidate[_firstOwner].push(_candidates[i]);
            validatorsState[_candidates[i]].voters[_firstOwner] = minCandidateCap;
        }
    }


    // uploadKYC : anyone can upload a KYC; its not equivalent to becoming an owner.
    function uploadKYC(string kychash) external {
        KYCString[msg.sender].push(kychash);
        emit UploadedKYC(msg.sender,kychash);
    }

    // propose : any non-candidate who has uploaded its KYC can become an owner by proposing a candidate.
    function propose(address _candidate) external payable onlyValidCandidateCap onlyKYCWhitelisted onlyNotCandidate(_candidate) {
        uint256 cap = validatorsState[_candidate].cap.add(msg.value);
        candidates.push(_candidate);
        validatorsState[_candidate] = ValidatorState({
            owner: msg.sender,
            isCandidate: true,
            cap: cap
        });
        validatorsState[_candidate].voters[msg.sender] = validatorsState[_candidate].voters[msg.sender].add(msg.value);
        candidateCount = candidateCount.add(1);
        if (ownerToCandidate[msg.sender].length ==0){
            owners.push(msg.sender);
            ownerCount++;
        }
        ownerToCandidate[msg.sender].push(_candidate);
        voters[_candidate].push(msg.sender);
        emit Propose(msg.sender, _candidate, msg.value);
    }

    function vote(address _candidate) external payable onlyValidVoterCap onlyValidCandidate(_candidate) {
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

    function getCandidateOwner(address _candidate) public view returns(address) {
        return validatorsState[_candidate].owner;
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

    function getWithdrawBlockNumbers() public view returns(uint256[]) {
        return withdrawsState[msg.sender].blockNumbers;
    }

    function getWithdrawCap(uint256 _blockNumber) public view returns(uint256) {
        return withdrawsState[msg.sender].caps[_blockNumber];
    }

    function unvote(address _candidate, uint256 _cap) public onlyValidVote(_candidate, _cap) {
        validatorsState[_candidate].cap = validatorsState[_candidate].cap.sub(_cap);
        validatorsState[_candidate].voters[msg.sender] = validatorsState[_candidate].voters[msg.sender].sub(_cap);

        // refund after delay X blocks
        uint256 withdrawBlockNumber = voterWithdrawDelay.add(block.number);
        withdrawsState[msg.sender].caps[withdrawBlockNumber] = withdrawsState[msg.sender].caps[withdrawBlockNumber].add(_cap);
        withdrawsState[msg.sender].blockNumbers.push(withdrawBlockNumber);

        emit Unvote(msg.sender, _candidate, _cap);
    }

    function resign(address _candidate) public onlyOwner(_candidate) onlyCandidate(_candidate) {
        validatorsState[_candidate].isCandidate = false;
        candidateCount = candidateCount.sub(1);
        for (uint256 i = 0; i < candidates.length; i++) {
            if (candidates[i] == _candidate) {
                delete candidates[i];
                break;
            }
        }
        uint256 cap = validatorsState[_candidate].voters[msg.sender];
        validatorsState[_candidate].cap = validatorsState[_candidate].cap.sub(cap);
        validatorsState[_candidate].voters[msg.sender] = 0;
        // refunding after resigning X blocks
        uint256 withdrawBlockNumber = candidateWithdrawDelay.add(block.number);
        withdrawsState[msg.sender].caps[withdrawBlockNumber] = withdrawsState[msg.sender].caps[withdrawBlockNumber].add(cap);
        withdrawsState[msg.sender].blockNumbers.push(withdrawBlockNumber);
        emit Resign(msg.sender, _candidate);
    }

    // voteInvalidKYC : any candidate can vote for invalid KYC i.e. a particular candidate's owner has uploaded a bad KYC.
    // On securing 75% votes against an owner ( not candidate ), owner & all its candidates will lose their funds.
    function voteInvalidKYC(address _invalidCandidate) onlyValidCandidate(msg.sender) onlyValidCandidate(_invalidCandidate) public {
        address candidateOwner = getCandidateOwner(msg.sender);
        address _invalidMasternode = getCandidateOwner(_invalidCandidate);
        require(!hasVotedInvalid[candidateOwner][_invalidMasternode]);
        hasVotedInvalid[candidateOwner][_invalidMasternode] = true;
        invalidKYCCount[_invalidMasternode] += 1;
        if( invalidKYCCount[_invalidMasternode]*100/getOwnerCount() >= 75 ){
            // 75% owners say that the KYC is invalid
            address[] memory allMasternodes = new address[](candidates.length-1) ;
            uint count=0;
            for (uint i=0;i<candidates.length;i++){
                if (getCandidateOwner(candidates[i])==_invalidMasternode){
                    // logic to remove cap.
                    candidateCount = candidateCount.sub(1);
                    allMasternodes[count++] = candidates[i];
                    delete candidates[i];
                    delete validatorsState[candidates[i]];
                    delete KYCString[_invalidMasternode];
                    delete ownerToCandidate[_invalidMasternode];
                    delete invalidKYCCount[_invalidMasternode];
                }
            }
            for(uint k=0;k<owners.length;k++){
                        if (owners[k]==_invalidMasternode){
                            delete owners[k];
                            ownerCount--;
                            break;
                } 
            }
            emit InvalidatedNode(_invalidMasternode,allMasternodes);
        }
    }

    // invalidPercent : get votes against an owner in percentage.
    function invalidPercent(address _invalidCandidate) onlyValidCandidate(_invalidCandidate) view public returns(uint){
        address _invalidMasternode = getCandidateOwner(_invalidCandidate);
        return (invalidKYCCount[_invalidMasternode]*100/getOwnerCount());
    }


    // getOwnerCount : get count of total owners; accounts who own atleast one masternode.
    function getOwnerCount() view public returns (uint){
        return ownerCount;
    }
    
    // getKYC : get KYC uploaded of the owner of the given masternode or the owner themselves
    function getLatestKYC(address _address) view public  returns (string) {
        if(isCandidate(_address)){
        return KYCString[getCandidateOwner(_address)][KYCString[getCandidateOwner(_address)].length-1];
        }
        else{
            return KYCString[_address][KYCString[_address].length-1];
        }
    }
    
    function getHashCount(address _address) view public returns(uint){
        return KYCString[_address].length;
    }

    function withdraw(uint256 _blockNumber, uint _index) public onlyValidWithdraw(_blockNumber, _index) {
        uint256 cap = withdrawsState[msg.sender].caps[_blockNumber];
        delete withdrawsState[msg.sender].caps[_blockNumber];
        delete withdrawsState[msg.sender].blockNumbers[_index];
        msg.sender.transfer(cap);
        emit Withdraw(msg.sender, _blockNumber, cap);
    }
}