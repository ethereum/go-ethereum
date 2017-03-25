// Ethereum Name Service contracts by Nick Johnson <nick@ethereum.org>
// 
// To the extent possible under law, the person who associated CC0 with
// ENS contracts has waived all copyright and related or neighboring rights
// to ENS.
// 
// You should have received a copy of the CC0 legalcode along with this
// work.  If not, see <http://creativecommons.org/publicdomain/zero/1.0/>.

/**
 * The ENS registry contract.
 */
contract ENS {
    struct Record {
        address owner;
        address resolver;
    }
    
    mapping(bytes32=>Record) records;
    
    // Logged when the owner of a node assigns a new owner to a subnode.
    event NewOwner(bytes32 indexed node, bytes32 indexed label, address owner);

    // Logged when the owner of a node transfers ownership to a new account.
    event Transfer(bytes32 indexed node, address owner);

    // Logged when the owner of a node changes the resolver for that node.
    event NewResolver(bytes32 indexed node, address resolver);
    
    // Permits modifications only by the owner of the specified node.
    modifier only_owner(bytes32 node) {
        if(records[node].owner != msg.sender) throw;
        _
    }
    
    /**
     * Constructs a new ENS registrar, with the provided address as the owner of the root node.
     */
    function ENS(address owner) {
        records[0].owner = owner;
    }
    
    /**
     * Returns the address that owns the specified node.
     */
    function owner(bytes32 node) constant returns (address) {
        return records[node].owner;
    }
    
    /**
     * Returns the address of the resolver for the specified node.
     */
    function resolver(bytes32 node) constant returns (address) {
        return records[node].resolver;
    }

    /**
     * Transfers ownership of a node to a new address. May only be called by the current
     * owner of the node.
     * @param node The node to transfer ownership of.
     * @param owner The address of the new owner.
     */
    function setOwner(bytes32 node, address owner) only_owner(node) {
        Transfer(node, owner);
        records[node].owner = owner;
    }

    /**
     * Transfers ownership of a subnode sha3(node, label) to a new address. May only be
     * called by the owner of the parent node.
     * @param node The parent node.
     * @param label The hash of the label specifying the subnode.
     * @param owner The address of the new owner.
     */
    function setSubnodeOwner(bytes32 node, bytes32 label, address owner) only_owner(node) {
        var subnode = sha3(node, label);
        NewOwner(node, label, owner);
        records[subnode].owner = owner;
    }

    /**
     * Sets the resolver address for the specified node.
     * @param node The node to update.
     * @param resolver The address of the resolver.
     */
    function setResolver(bytes32 node, address resolver) only_owner(node) {
        NewResolver(node, resolver);
        records[node].resolver = resolver;
    }
}

/**
 * A registrar that allocates subdomains to the first person to claim them. It also deploys
 * a simple resolver contract and sets that as the default resolver on new names for
 * convenience.
 */
contract FIFSRegistrar {
    ENS ens;
    PublicResolver defaultResolver;
    bytes32 rootNode;
    
    /**
     * Constructor.
     * @param ensAddr The address of the ENS registry.
     * @param node The node that this registrar administers.
     */
    function FIFSRegistrar(address ensAddr, bytes32 node) {
        ens = ENS(ensAddr);
        defaultResolver = new PublicResolver(ensAddr);
        rootNode = node;
    }

    /**
     * Register a name, or change the owner of an existing registration.
     * @param subnode The hash of the label to register.
     * @param owner The address of the new owner.
     */    
    function register(bytes32 subnode, address owner) {
        var node = sha3(rootNode, subnode);
        var currentOwner = ens.owner(node);
        if(currentOwner != 0 && currentOwner != msg.sender)
            throw;

        // Temporarily set ourselves as the owner
        ens.setSubnodeOwner(rootNode, subnode, this);
        // Set up the default resolver
        ens.setResolver(node, defaultResolver);
        // Set the owner to the real owner
        ens.setOwner(node, owner);
    }
}

contract Resolver {
    event AddrChanged(bytes32 indexed node, address a);
    event ContentChanged(bytes32 indexed node, bytes32 hash);

    function has(bytes32 node, bytes32 kind) returns (bool);
    function addr(bytes32 node) constant returns (address ret);
    function content(bytes32 node) constant returns (bytes32 ret);
}

/**
 * A simple resolver anyone can use; only allows the owner of a node to set its
 * address.
 */
contract PublicResolver is Resolver {
    ENS ens;
    mapping(bytes32=>address) addresses;
    mapping(bytes32=>bytes32) contents;
    
    modifier only_owner(bytes32 node) {
        if(ens.owner(node) != msg.sender) throw;
        _
    }

    /**
     * Constructor.
     * @param ensAddr The ENS registrar contract.
     */
    function PublicResolver(address ensAddr) {
        ens = ENS(ensAddr);
    }

    /**
     * Fallback function.
     */
    function() {
        throw;
    }

    /**
     * Returns true if the specified node has the specified record type.
     * @param node The ENS node to query.
     * @param kind The record type name, as specified in EIP137.
     * @return True if this resolver has a record of the provided type on the
     *         provided node.
     */
    function has(bytes32 node, bytes32 kind) returns (bool) {
        return (kind == "addr" && addresses[node] != 0) ||
               (kind == "content" && contents[node] != 0);
    }
    
    /**
     * Returns the address associated with an ENS node.
     * @param node The ENS node to query.
     * @return The associated address.
     */
    function addr(bytes32 node) constant returns (address ret) {
        ret = addresses[node];
        if(ret == 0)
            throw;
    }
    
    /**
     * Returns the content hash associated with an ENS node.
     * @param node The ENS node to query.
     * @return The associated content hash.
     */
    function content(bytes32 node) constant returns (bytes32 ret) {
        ret = contents[node];
        if(ret == 0)
            throw;
    }

    /**
     * Sets the address associated with an ENS node.
     * May only be called by the owner of that node in the ENS registry.
     * @param node The node to update.
     * @param addr The address to set.
     */
    function setAddr(bytes32 node, address addr) only_owner(node) {
        addresses[node] = addr;
        AddrChanged(node, addr);
    }
    
    /**
     * Sets the content hash associated with an ENS node.
     * May only be called by the owner of that node in the ENS registry.
     * @param node The node to update.
     * @param hash The content hash to set.
     */
    function setContent(bytes32 node, bytes32 hash) only_owner(node) {
        contents[node] = hash;
        ContentChanged(node, hash);
    }
}
