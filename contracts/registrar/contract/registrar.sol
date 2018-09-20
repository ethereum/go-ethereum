pragma solidity ^0.5.2;

/**
 * @title Registrar
 * @author Gary Rong<garyrong@ethereum.org>
 * @dev Implementation of the blockchain checkpoint registrar.
 */
contract Registrar {
    /* 
       Definitions
    */

    // Vote represents a checkpoint announcement from a trusted signer.
    struct Vote {
        address addr;
        bytes   sig;
    }

    // PendingProposal represents a tally for current pending new checkpoint
    // proposal.
    struct PendingProposal {
        uint index; // Checkpoint section index
        uint count; // Number of signers who have submitted checkpoint announcement
        mapping(address => bytes32) usermap; // map between signer address and advertised checkpoint hash
        mapping(bytes32 => Vote[]) votemap; // map between checkpoint hash and relative signer announcements.
    }

    /*
        Modifiers
    */

    /**
     * @dev Check whether the message sender is authorized.
     */
    modifier OnlyAuthorized() {
        require(admins[msg.sender]);
        _;
    }

    /*
        Events
    */

    // NewCheckpointEvent is emitted when a new checkpoint proposal receive enough approvals.
    // We use checkpoint hash instead of the full checkpoint to make the transaction cheaper.
    event NewCheckpointEvent(uint indexed index, bytes32 checkpointHash, bytes signature);

    /*
        Public Functions
    */
    constructor(address[] memory _adminlist, uint _sectionSize, uint _processConfirms, uint _threshold) public {
        for (uint i = 0; i < _adminlist.length; i++) {
            admins[_adminlist[i]] = true;
            adminList.push(_adminlist[i]);
        }
        sectionSize = _sectionSize;
        processConfirms = _processConfirms;
        threshold = _threshold;
    }

    /**
     * @dev Get latest stable checkpoint information.
     * @return section index
     * @return checkpoint hash
     * @return block height associated with checkpoint
     */
    function GetLatestCheckpoint()
    view
    public
    returns(uint, bytes32, uint) {
        (bytes32 hash, uint height) = GetCheckpoint(latest);
        return (latest, hash, height);
    }

    /**
     * @dev Get a stable checkpoint information with specified section index.
     * @param _sectionIndex section index
     * @return checkpoint hash
     * @return the associated register block height
     */
    function GetCheckpoint(uint _sectionIndex)
    view
    public
    returns(bytes32, uint)
    {
        return (checkpoints[_sectionIndex], register_height[_sectionIndex]);
    }

    /**
     * @dev Submit a new checkpoint announcement
     * Checkpoint represents a set of post-processed trie roots (CHT and BloomTrie)
     * associated with the appropriate section head hash.
     *
     * It is used to start light syncing from this checkpoint and avoid downloading
     * the entire header chain while still being able to securely access old headers/logs.
     *
     * @param _sectionIndex section index
     * @param _hash checkpoint hash calculated in the client side
     * @param _sig admin's signature for checkpoint hash
     *         `checkpoint_hash = Hash(index, sectionHead, chtRoot, bloomRoot)`
     *         `_sig = Sign(privateKey, checkpoint_hash)`
     * @return indicator whether set checkpoint successfully
     */
    function SetCheckpoint(
        uint _sectionIndex,
        bytes32 _hash,
        bytes memory _sig
    )
    OnlyAuthorized
    public
    returns(bool)
    {
        // Checkpoint register/modification time window: [(secIndex+1)*size + confirms, (secIndex+2)*size)
        if (block.number < (_sectionIndex+1)*sectionSize+processConfirms || block.number >= (_sectionIndex+2)*sectionSize) {
            return false;
        }
        // Filter out stale announcement
        if (_sectionIndex == latest && (latest != 0 || register_height[0] != 0)) {
            return false;
        }
        // Filter out invalid announcement
        if (_hash == "" || _sig.length == 0) {
            return false;
        }
        // Delete stale pending proposal silently
        if (pending_proposal.index != _sectionIndex) {
            deletePending();
        }
        bytes32 old = pending_proposal.usermap[msg.sender];
        // Filter out duplicate announcement
        if (old == _hash) {
            return false;
        }
        bool isNew = (old == "");
        pending_proposal.usermap[msg.sender] = _hash;

        if (!isNew) {
            // Checkpoint modification
            Vote[] storage votes = pending_proposal.votemap[old];
            for (uint i = 0; i < votes.length - 1; i++) {
                if (votes[i].addr == msg.sender) {
                    votes[i] = votes[votes.length - 1];
                    break;
                }
            }
            delete votes[votes.length-1];
            votes.length -= 1;
            pending_proposal.votemap[_hash].push(Vote({
                addr: msg.sender,
                sig:  _sig
            }));
        } else {
            // New checkpoint announcement
            pending_proposal.count += 1;
            pending_proposal.index = _sectionIndex;
            pending_proposal.votemap[_hash].push(Vote({
                addr: msg.sender,
                sig:  _sig
            }));
        }
        if (pending_proposal.votemap[_hash].length < threshold) {
           return true;
        }
        checkpoints[_sectionIndex] = _hash;
        register_height[_sectionIndex] = block.number;
        latest = _sectionIndex;

        bytes memory sigs;
        for (uint idx = 0; idx < threshold; idx++) {
            sigs = abi.encodePacked(sigs, pending_proposal.votemap[_hash][idx].sig);
        }
        emit NewCheckpointEvent(_sectionIndex, _hash, sigs);
        deletePending();
        return true;
    }

    /**
     * @dev Get all admin addresses
     * @return address list
     */
    function GetAllAdmin()
    public
    view
    returns(address[] memory)
    {
        address[] memory ret = new address[](adminList.length);
        for (uint i = 0; i < adminList.length; i++) {
            ret[i] = adminList[i];
        }
        return ret;
    }

    /**
     * @dev Get the detail of pending proposal
     * @return checkpoint index
     * @return signers who have submitted checkpoint announcement
     * @return hashes corresponding checkpoint hash
     */
    function GetPending()
    public
    view
    returns(uint, address[] memory, bytes32[] memory)
    {
        uint idx = 0;
        address[] memory addr = new address[](pending_proposal.count);
        bytes32[] memory hashes = new bytes32[](pending_proposal.count);
        for (uint i = 0; i < adminList.length; i++) {
            bytes32 h = pending_proposal.usermap[adminList[i]];
            if (h != "") {
                addr[idx] = adminList[i];
                hashes[idx] = h;
                idx += 1;
            }
        }
        return (pending_proposal.index, addr, hashes);
    }

    /**
     * @dev Clear pending proposal
     */
    function deletePending()
    private
    {
        for (uint i = 0; i < adminList.length; i++) {
            bytes32 h = pending_proposal.usermap[adminList[i]];
            if (h != "") {
                delete pending_proposal.votemap[h];
                delete pending_proposal.usermap[adminList[i]];
            }
        }
        delete pending_proposal;
    }

    /*
        Fields
    */
    // Inflight new stable checkpoint proposal.
    PendingProposal pending_proposal;

    // A map of admin users who have the permission to update CHT and bloom Trie root
    mapping(address => bool) admins;

    // A list of admin users so that we can obtain all admin users.
    address[] adminList;

    // Registered checkpoint information
    mapping(uint => bytes32) checkpoints;

    // Latest stored section id
    // Note all registered checkpoint information should continuous with previous one.
    uint latest;

    // The block height associated with latest registered checkpoint.
    mapping(uint => uint) register_height;

    // The frequency for creating a checkpoint
    //
    // The default value should be the same as the checkpoint size(32768) in the ethereum.
    uint sectionSize;

    // The number of confirmations needed before a checkpoint can be registered.
    // We have to make sure the checkpoint registered will not be invalid due to
    // chain reorg.
    //
    // The default value should be the same as the checkpoint process confirmations(256) 
    // in the ethereum.
    uint processConfirms;
    
    // The required signatures to finalize a stable checkpoint.
    uint threshold;
}

