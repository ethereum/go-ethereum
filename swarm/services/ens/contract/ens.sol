/**
 * ENS resolver interface.
 */
contract Resolver {
    bytes32 constant TYPE_STAR = "*";
    
    // Response codes.
    uint16 constant RCODE_OK = 0;
    uint16 constant RCODE_NXDOMAIN = 3;

    // These methods are shared by all resolvers
    function findResolver(bytes12 nodeId, bytes32 label) constant
        returns (uint16 rcode, uint32 ttl, bytes12 rnode, address raddress);
    function resolve(bytes12 nodeId, bytes32 qtype, uint16 index) constant
        returns (uint16 rcode, bytes16 rtype, uint32 ttl, uint16 len,
                 bytes32 data);
    function getExtended(bytes32 id) constant returns (bytes data);

    // These methods are implemented by personal resolvers
    function isPersonalResolver() constant returns (bool);
    function setRR(bytes12 rootNodeId, string name, bytes16 rtype, uint32 ttl, uint16 len, bytes32 data);
    function setPrivateRR(bytes12 rootNodeId, bytes32[] name, bytes16 rtype, uint32 ttl, uint16 len, bytes32 data);
    function deleteRR(bytes12 rootNodeId, string name);
    function deletePrivateRR(bytes12 rootNodeId, bytes32[] name);

    // These methods are implemented by open registrar implementations.
    function register(bytes32 label, address resolver, bytes12 nodeId);
    function setOwner(bytes32 label, address newOwner);
    function setResolver(bytes32 label, address resolver, bytes12 nodeId);
    function getOwner(bytes32 label) constant returns (address);
}
