pragma solidity ^0.4.0;

import './AbstractENS.sol';

/**
 * A registrar that allocates subdomains to the first person to claim them.
 */
contract FIFSRegistrar {
    AbstractENS ens;
    bytes32 rootNode;

    modifier only_owner(bytes32 subnode) {
        var node = sha3(rootNode, subnode);
        var currentOwner = ens.owner(node);

        if (currentOwner != 0 && currentOwner != msg.sender) throw;

        _;
    }

    /**
     * Constructor.
     * @param ensAddr The address of the ENS registry.
     * @param node The node that this registrar administers.
     */
    function FIFSRegistrar(AbstractENS ensAddr, bytes32 node) {
        ens = ensAddr;
        rootNode = node;
    }

    /**
     * Register a name, or change the owner of an existing registration.
     * @param subnode The hash of the label to register.
     * @param owner The address of the new owner.
     */
    function register(bytes32 subnode, address owner) only_owner(subnode) {
        ens.setSubnodeOwner(rootNode, subnode, owner);
    }
}
