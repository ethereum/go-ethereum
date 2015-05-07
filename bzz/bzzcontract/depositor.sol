/// @title Swarm Distributed Preimage Archive
/// @author Daniel A. Nagy <daniel@ethdev.com>
contract Swarm
{

  enum Status {Clean, Suspect, Guilty}

  struct Bee {
    uint deposit;
    uint expiry;
    Status status;
  }

  mapping (address => mapping (address => Bee)) swarm;

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
  ///
  /// @param arbiter address of arbiter contract (or external arbiter entity)
  function signup(uint time, address arbiter) {
    Bee b = swarm[arbiter][msg.sender];
    if(b.status == Status.Clean && now + time > now) {
      b.expiry = max(b.expiry, now + time);
    }
    b.deposit += msg.value;
  }

  /// @notice Withdraw from Swarm, refund deposit.
  ///
  /// @dev Only allowed with clean status and expired term.
  ///
  /// @param arbiter address of arbiter contract (or external arbiter entity)
  function withdraw(address arbiter) {
    Bee b = swarm[arbiter][msg.sender];
    if(now > b.expiry && b.status == Status.Clean) {
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
  /// @param arbiter address of arbiter contract (or external arbiter entity)
  ///
  /// @return balance of queried address.
  function balance(address addr, address arbiter) returns (uint d) {
    Bee b = swarm[arbiter][addr];
    return b.deposit;
  }

  /// @notice Determine clean status of address `addr`.
  /// No change in state.
  ///
  /// @param addr queried address.
  ///
  /// @param arbiter address of arbiter contract (or external arbiter entity)
  ///
  /// @return true if status is "Clean".
  function isClean(address addr, address arbiter) returns (bool s) {
    Bee b = swarm[arbiter][addr];
    return b.status == Status.Clean;
  }

  /// @notice Determine suspect status of address `addr`.
  /// No change in state.
  ///
  /// @param addr queried address.
  ///
  /// @param arbiter address of arbiter contract (or external arbiter entity)
  ///
  /// @return true if status is "Suspect".
  function isSuspect(address addr, address arbiter) returns (bool s) {
    Bee b = swarm[arbiter][addr];
    return b.status == Status.Suspect;
  }

  /// @notice Determine guilty status of address `addr`.
  /// No change in state.
  ///
  /// @param addr queried address.
  ///
  /// @param arbiter address of arbiter contract (or external arbiter entity)
  ///
  /// @return true if status is "Guilty".
  function isGuilty(address addr, address arbiter) returns (bool s) {
    Bee b = swarm[arbiter][addr];
    return b.status == Status.Guilty;
  }

  /// @notice Determine if the deposit for `addr` is unaccessible until `time`.
  /// No change in state.
  ///
  /// @param addr queried address.
  ///
  /// @param time queried time.
  ///
  /// @param arbiter address of arbiter contract (or external arbiter entity)
  ///
  /// @return true if deposit expires after queried time.
  function expiresAfter(address addr, uint time, address arbiter) returns (bool s) {
    Bee b = swarm[arbiter][addr];
    return b.expiry > time;
  }
}
