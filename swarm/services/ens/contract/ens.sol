import "mortal";
/// @title Swarm Distributed Preimage Archive
/// @author Viktor Tron <viktor@ethdev.com>
contract ENS is mortal
{

  mapping (bytes32 => bytes32) public Registry;
  mapping (bytes32 => address) public Owners;

  function Set(bytes32 host, bytes32 content) {
    if (Owners[host] == 0x0) {
      Owners[host] = tx.origin;
    }
    if (Owners[host] == tx.origin) {
      Registry[host] = content;
    }
  }

}