/// @title Swarm Distributed Preimage Archive
/// @author Daniel A. Nagy <daniel@ethdev.com>
contract Swarm
{

  uint constant GRACE = 50; // grace period for lost information in blocks

  bytes32 constant MAGIC_NUMBER = "Swarm receipt";

  struct Bee {
    uint deposit;    // amount deposited by this member
    uint expiry;     // expiration time of the deposit
    uint256 missing; // member accused of losing this swarm chunk
    uint deadline;   // block number before which chunk must be presented
  }

  mapping (address => Bee) swarm;

  function max(uint a, uint b) private returns (uint c) {
    if(a >= b) return a;
    return b;
  }

  /// @notice Sign up as a Swarm node for `time` seconds.
  /// No term extension for nodes with non-clean status.
  ///
  /// @dev Guards against term overflow and unauthorized extension,
  /// but all funds are added to deposite irrespective of status.
  ///
  /// @param time term of Swarm membership in seconds from now.
  function signup(uint time) {
    Bee b = swarm[msg.sender];
    if(isClean(msg.sender) && now + time > now) {
      b.expiry = max(b.expiry, now + time);
    }
    b.deposit += msg.value;
  }

  /// @notice Withdraw from Swarm, refund deposit.
  ///
  /// @dev Only allowed with clean status and expired term.
  function withdraw() {
    Bee b = swarm[msg.sender];
    if(now > b.expiry && isClean(msg.sender)) {
	msg.sender.send(b.deposit);
	b.deposit = 0;
    }
  }

  /// @notice Total deposit for address `addr`.
  /// No change in state.
  ///
  /// @dev Not meaningful for "Guilty" status.
  ///
  /// @param addr queried address.
  ///
  /// @return balance of queried address.
  function balance(address addr) returns (uint d) {
    Bee b = swarm[addr];
    return b.deposit;
  }

  /// @notice Determine clean status of address `addr`.
  /// No change in state.
  ///
  /// @dev Defined as no signed receipt has been presented for missing chunk.
  ///
  /// @param addr queried address.
  ///
  /// @return true if status is "Clean".
  function isClean(address addr) returns (bool s) {
    Bee b = swarm[addr];
    return b.missing == 0; // nothing they signed is missing
  }

  /// @notice Determine if the deposit for `addr` is unaccessible until `time`.
  /// No change in state.
  ///
  /// @param addr queried address.
  ///
  /// @param time queried time.
  ///
  /// @return true if deposit expires after queried time.
  function expiresAfter(address addr, uint time) returns (bool s) {
    Bee b = swarm[addr];
    return b.expiry > time;
  }
}
