pragma solidity ^0.6.0;

/**
 * @title CheckpointOracle
 * @author Gary Rong<garyrong@ethereum.org>, Martin Swende <martin.swende@ethereum.org>
 * @dev Implementation of the blockchain checkpoint registrar.
 */
contract CheckpointOracle {
    /*
        Events
    */

    // NewCheckpointVote is emitted when a new checkpoint proposal receives a vote.
    event NewCheckpointVote(uint64 indexed index, bytes32 checkpointHash, uint8 v, bytes32 r, bytes32 s);

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
    returns(uint64, bytes32, uint) {
        return (sectionIndex, hash, height);
    }

    // SetCheckpoint sets  a new checkpoint. It accepts a list of signatures
    // @_recentNumber: a recent blocknumber, for replay protection
    // @_recentHash : the hash of `_recentNumber`
    // @_hash : the hash to set at _sectionIndex
    // @_sectionIndex : the section index to set
    // @v : the list of v-values
    // @r : the list or r-values
    // @s : the list of s-values
    function SetCheckpoint(
        uint _recentNumber,
        bytes32 _recentHash,
        bytes32 _hash,
        uint64 _sectionIndex,
        uint8[] memory v,
        bytes32[] memory r,
        bytes32[] memory s)
        public
        returns (bool)
    {
        // Ensure the sender is authorized.
        require(admins[msg.sender]);

        // These checks replay protection, so it cannot be replayed on forks,
        // accidentally or intentionally
        require(blockhash(_recentNumber) == _recentHash);

        // Ensure the batch of signatures are valid.
        require(v.length == r.length);
        require(v.length == s.length);

        // Filter out "future" checkpoint.
        if (block.number < (_sectionIndex+1)*sectionSize+processConfirms) {
            return false;
        }
        // Filter out "old" announcement
        if (_sectionIndex < sectionIndex) {
            return false;
        }
        // Filter out "stale" announcement
        if (_sectionIndex == sectionIndex && (_sectionIndex != 0 || height != 0)) {
            return false;
        }
        // Filter out "invalid" announcement
        if (_hash == ""){
            return false;
        }

        // EIP 191 style signatures
        //
        // Arguments when calculating hash to validate
        // 1: byte(0x19) - the initial 0x19 byte
        // 2: byte(0) - the version byte (data with intended validator)
        // 3: this - the validator address
        // --  Application specific data
        // 4 : checkpoint section_index(uint64)
        // 5 : checkpoint hash (bytes32)
        //     hash = keccak256(checkpoint_index, section_head, cht_root, bloom_root)
        bytes32 signedHash = keccak256(abi.encodePacked(byte(0x19), byte(0), this, _sectionIndex, _hash));

        address lastVoter = address(0);

        // In order for us not to have to maintain a mapping of who has already
        // voted, and we don't want to count a vote twice, the signatures must
        // be submitted in strict ordering.
        for (uint idx = 0; idx < v.length; idx++){
            address signer = ecrecover(signedHash, v[idx], r[idx], s[idx]);
            require(admins[signer]);
            require(uint256(signer) > uint256(lastVoter));
            lastVoter = signer;
            emit NewCheckpointVote(_sectionIndex, _hash, v[idx], r[idx], s[idx]);

            // Sufficient signatures present, update latest checkpoint.
            if (idx+1 >= threshold){
                hash = _hash;
                height = block.number;
                sectionIndex = _sectionIndex;
                return true;
            }
        }
        // We shouldn't wind up here, reverting un-emits the events
        revert();
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

    /*
        Fields
    */
    // A map of admin users who have the permission to update CHT and bloom Trie root
    mapping(address => bool) admins;

    // A list of admin users so that we can obtain all admin users.
    address[] adminList;

    // Latest stored section id
    uint64 sectionIndex;

    // The block height associated with latest registered checkpoint.
    uint height;

    // The hash of latest registered checkpoint.
    bytes32 hash;

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
