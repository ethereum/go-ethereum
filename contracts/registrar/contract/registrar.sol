pragma solidity ^0.4.24;

/**
 * @title Registrar
 * @author Gary Rong<garyrong0905@gmail.com>
 * @dev Implementation of the blockchain checkpoint information registrar.
 */
contract Registrar {
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
    // Grantor indicates the people register the checkpoint.
    // We use checkpoint hash instead of the full checkpoint to make the transaction cheaper.
    event NewCheckpointEvent(uint indexed index, address grantor, bytes32 checkpointHash);

    // AddAdminEvent is emitted when new address is accepted as admin.
    // Grantor indicates who authorizes the add admin operation.
    event AddAdminEvent(address addr, address grantor, string description);

    // RemoveAdminEvent is emitted when an admin is removed.
    // Grantor indicates who authorizes the remove admin operation.
    event RemoveAdminEvent(address addr, address grantor, string reason);

    /*
        Public Functions
    */
    constructor(address[] _adminlist) public {
        // regard contract creator as a default admin.
        admins[msg.sender] = 1;
        adminList.push(msg.sender);
        for (uint i = 0; i < _adminlist.length; i++) {
            admins[_adminlist[i]] = 1;
            adminList.push(_adminlist[i]);
        }
    }

    /**
     * @dev Get latest stable checkpoint information.
     * @return section index
     * @return checkpoint hash
     */
    function GetLatestCheckpoint()
    view
    public
    returns(uint, bytes32) {
        bytes32 hash = GetCheckpoint(latest);
        return (latest, hash);
    }

    /**
     * @dev Get a stable checkpoint information with specified section index.
     * @param _sectionIndex section index
     * @return checkpoint hash
     */
    function GetCheckpoint(uint _sectionIndex)
    view
    public
    returns(bytes32)
    {
        return checkpoints[_sectionIndex];
    }

    /**
     * @dev Set stable checkpoint information.
     * Checkpoint represents a set of post-processed trie roots (CHT and BloomTrie)
     * associated with the appropriate section head hash.
     *
     * It is used to start light syncing from this checkpoint
     * and avoid downloading the entire header chain while still being able to securely
     * access old headers/logs.
     *
     * Note we trust the given information here provided by foundation,
     * need a trust less version for future.
     * @param _sectionIndex section index
     * @param _hash checkpoint hash calculated in the client side
     * @return indicator whether set checkpoint successfully
     */
    function SetCheckpoint(
        uint _sectionIndex,
        bytes32 _hash
    )
    OnlyAuthorized
    public
    returns(bool)
    {
        // Ensure the checkpoint information provided is strictly continuous with previous one.
        // But the latest checkpoint modification is allowed.
        if (_sectionIndex != latest && _sectionIndex != latest + 1 && latest != 0) {
            return false;
        }
        // Ensure the checkpoint is stable enough to be registered.
        if (block.number < (_sectionIndex+1)*sectionSize+confirmations) {
            return false;
        }

        checkpoints[_sectionIndex] = _hash;
        latest = _sectionIndex;

        emit NewCheckpointEvent(_sectionIndex, msg.sender, _hash);
    }

    /**
     * @dev Add a new address to admin list
     * @param _addr specified new admin address.
     * @return indicator whether add new admin successfully
     */
    function AddAdmin(address _addr, string _description)
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

        emit AddAdminEvent(_addr, msg.sender, _description);
        return true;
    }

    /**
     * @dev Remove a admin from the list
     * @param _addr specified admin address to remove.
     * @return indicator whether remove admin successfully
     */
    function RemoveAdmin(address _addr, string _reason)
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

        emit RemoveAdminEvent(_addr, msg.sender, _reason);
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
    mapping(uint => bytes32) checkpoints;

    // Latest stored section id
    // Note all registered checkpoint information should continuous with previous one.
    uint latest;

    // The frequency for creating a checkpoint
    uint constant sectionSize = 32768;

    // The number of confirmations needed before a checkpoint can be registered.
    // We have to make sure the checkpoint registered will not be invalid due to
    // chain reorg.
    uint constant confirmations = 256;
}

