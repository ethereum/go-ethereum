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
    if(b.status == Clean && now + time > now) {
      b.expiry = max(b.expiry, now + time);
    }
    b.deposit += msg.value;
  }

  /// @notice Withdraw from Swarm, refund deposit.
  ///
  /// @dev Only allowed with clean status and expired term.
  function withdraw() {
    Bee b = swarm[msg.sender];
    if(now > b.expiry && b.status == clean) {
	msg.sender.send(b.deposit);
	b.deposit = 0;
    }
  }

}
