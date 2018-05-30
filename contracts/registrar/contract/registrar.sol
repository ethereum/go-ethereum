pragma solidity ^0.4.24;

/**
 * @title Registrar
 * @author Gary Rong<garyrong0905@gmail.com>
 * @dev Implementation of the blockchain checkpoint information registrar.
 */
contract Registrar {
    /*
        Definitions
    */

    // Checkpoint represents a set of post-processed trie roots (CHT and BloomTrie)
    // associated with the appropriate section head hash.
    //
    // It is used to start light syncing from this checkpoint
    // and avoid downloading the entire header chain while still being able to securely
    // access old headers/logs.
    struct Checkpoint {
        bytes32 sectionHead;
        bytes32 chtRoot;
        bytes32 bloomTrieRoot;
    }

    /*
        Modifiers
    */

    /**
     * @dev Check whether the message sender is authorized.
     */
    modifier OnlyAuthorized() {
        require(admins[msg.sender] > 0);
        _;
    }

    /*
        Events
    */

    // NewCheckpointEvent is emitted when new checkpoint is registered.
    event NewCheckpointEvent(uint indexed index, bytes32 sectionHead, bytes32 chtRoot, bytes32 bloomTrieRoot);

    // AddAdminEvent is emitted when new address is accepted as admin.
    event AddAdminEvent(address addr);

    // RemoveAdminEvent is emitted when an admin is removed.
    event RemoveAdminEvent(address addr);

    /*
        Public Functions
    */
    constructor(address[] _adminlist) public {
        for (uint i = 0; i < _adminlist.length; i++) {
            admins[_adminlist[i]] = 1;
            adminList.push(_adminlist[i]);
        }
    }

    /**
     * @dev Get latest stable checkpoint information.
     * @return section index
     * @return section head
     * @return cht root hash
     * @return bloom trie root hash
     */
    function GetLatestCheckpoint()
    view
    public
    returns(uint, bytes32, bytes32, bytes32) {
        (bytes32 sectionHead, bytes32 chtRoot, bytes32 bloomRoot) = GetCheckpoint(latest);
        return (latest, sectionHead, chtRoot, bloomRoot);
    }

    /**
     * @dev Get a stable checkpoint information with specified section index.
     * @param _sectionIndex section index
     * @return section head
     * @return cht root hash
     * @return bloom trie root hash
     */
    function GetCheckpoint(uint _sectionIndex)
    view
    public
    returns(bytes32, bytes32, bytes32)
    {
        Checkpoint memory checkpoint = checkpoints[_sectionIndex];
        return (checkpoint.sectionHead, checkpoint.chtRoot, checkpoint.bloomTrieRoot);
    }

    /**
     * @dev Set stable checkpoint information.
     *
     * Note we trust the given information here provided by foundation,
     * need a trust less version for future.
     * @param _sectionIndex section index
     * @param _sectionHead section header
     * @param _chtRoot cht root hash
     * @param _bloomTrieRoot bloom trie root hash
     * @return indicator whether set checkpoint successfully
     */
    function SetCheckpoint(
        uint _sectionIndex,
        bytes32 _sectionHead,
        bytes32 _chtRoot,
        bytes32 _bloomTrieRoot
    )
    OnlyAuthorized
    public
    returns(bool)
    {
        // Ensure the checkpoint information provided is strictly continuous with previous one.
        if (_sectionIndex != latest + 1 && latest != 0) {
            return false;
        }
        checkpoints[_sectionIndex] = Checkpoint({
            sectionHead:   _sectionHead,
            chtRoot:       _chtRoot,
            bloomTrieRoot: _bloomTrieRoot
        });
        latest = _sectionIndex;

        emit NewCheckpointEvent(_sectionIndex, _sectionHead, _chtRoot, _bloomTrieRoot);
    }

    /**
     * @dev Add a new address to admin list
     * @param _addr specified new admin address.
     * @return indicator whether add new admin successfully
     */
    function AddAdmin(address _addr)
    OnlyAuthorized
    public
    returns(bool)
    {
        // Ensure the specified address is not admin yet.
        if (admins[_addr] > 0) {
            return false;
        }
        admins[_addr] = 1;
        adminList.push(_addr);

        emit AddAdminEvent(_addr);
        return true;
    }

    /**
     * @dev Remove a admin from the list
     * @param _addr specified admin address to remove.
     * @return indicator whether remove admin successfully
     */
    function RemoveAdmin(address _addr)
    OnlyAuthorized
    public
    returns(bool)
    {
        // Ensure the specified address is admin.
        if (admins[_addr] == 0) {
            return false;
        }
        delete admins[_addr];
        for (uint i = 0; i < adminList.length; i++) {
            if (adminList[i] == _addr) {
                // Not leave a gap
                for (uint idx = i; idx < adminList.length-1; idx++){
                    adminList[idx] = adminList[idx+1];
                }
                delete adminList[adminList.length-1];
                adminList.length -= 1;
                break;
            }
        }

        emit RemoveAdminEvent(_addr);
        return true;
    }


    /**
     * @dev Get all admin addresses
     * @return address list
     */
    function GetAllAdmin()
    public
    view
    returns(address[])
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
    mapping(address => uint) admins;

    // A list of admin users so that we can obtain all admin users.
    address[] adminList;

    // Registered checkpoint information
    mapping(uint => Checkpoint) checkpoints;

    // Latest stored section id
    // Note all registered checkpoint information should continuous with previous one.
    uint latest;
}

