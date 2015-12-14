/// @title Swarm Distributed Preimage Archive
/// @author Daniel A. Nagy <daniel@ethdev.com>
contract Swarm
{

  uint constant GRACE = 50; // grace period for lost information in blocks
  uint constant REWARD_FRACTION = 10; // this fraction of a deposit is paid as reward

  bytes32 constant MAGIC_NUMBER = "Swarm receipt";

  struct Bee {
    uint deposit;    // amount deposited by this member
    uint expiry;     // expiration time of the deposit
    bytes32 missing; // member accused of losing this swarm chunk
    uint deadline;   // block number before which chunk must be presented
    address reporter; // receipt reported by this address
  }

  mapping (address => Bee) swarm;

  // block number of transactions presenting chunks
  mapping (bytes32 => uint) presentedChunks;

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
    Bee b = swarm[tx.origin];
    if(isClean(msg.sender) && now + time > now) {
      b.expiry = max(b.expiry, now + time);
    }
    b.deposit += msg.value;
  }

  /// @notice Withdraw from Swarm, refund deposit.
  ///
  /// @dev Only allowed with clean status and expired term.
  function withdraw() {
    Bee b = swarm[tx.origin];
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
  /// Changes the state, but only as a matter of optimization.
  /// Works as accessor.
  ///
  /// @dev Defined as no signed receipt has been presented for missing chunk.
  ///
  /// @param addr queried address.
  ///
  /// @return true if status is "Clean".
  function isClean(address addr) returns (bool s) {
    Bee b = swarm[addr];
    if(b.missing != 0 && presentedChunks[b.missing] != 0) b.missing = 0;
    return b.missing == 0; // nothing they signed is missing
  }

  /// @param suspect address of reported Swarm node
  event Report(address suspect);

  /// @notice Find out what is missing in case of a Report event.
  ///
  /// @return 0 if nothing is missing, swarm hash otherwise
  function whatIsMissing() returns (bytes32 h) {
      bytes32 missing = swarm[tx.origin].missing;
      if(presentedChunks[missing] != 0) missing = 0;
      return missing;
  }

  /// @notice Report chunk `swarmHash` as missing.
  ///
  /// @param swarmHash sha3 hash of the missing chunk
  /// @param expiry expiration time of receipt
  /// @param sig_v signature parameter v
  /// @param sig_r signature parameter r
  /// @param sig_s signature parameter s
  function reportMissingChunk(bytes32 swarmHash, uint expiry,
        uint8 sig_v, bytes32 sig_r, bytes32 sig_s) {
      if(expiry < now) return;
      bytes32 recptHash = sha3(MAGIC_NUMBER, swarmHash, expiry);
      address signer = ecrecover(recptHash, sig_v, sig_r, sig_s);
      if(!isClean(signer) || !expiresAfter(signer, now)) return;
      Bee b = swarm[signer];
      b.missing = swarmHash;
      b.deadline = block.number + GRACE;
      b.reporter = msg.sender;
      Report(signer);
  }

  /// @notice Present a chunk in order to avoid losing deposit.
  ///
  /// @param chunk chunk data
  function presentMissingChunk(bytes chunk) external {
      bytes32 swarmHash = sha3(chunk);
      presentedChunks[swarmHash] = block.number;
  }

  /// @notice Determine guilty status of address `addr`.
  /// No change in state.
  ///
  /// @dev Definition of guilty is failing to present missing chunk within grace period.
  ///
  /// @param addr queried address.
  ///
  /// @return true, if status is "Guilty".
  function isGuilty(address addr) returns (bool g){
      if(isClean(addr)) return false;
      Bee b = swarm[addr];
      return b.deadline < block.number;
  }

  /// @notice Collect rewards for successfully prosecuting `addr`.
  ///
  /// @dev This implies burning 9/10 of the security deposit.
  ///
  /// @param addr guilty defendant address
  function claimReporterReward(address addr) {
      if(!isGuilty(addr)) return;
      Bee b = swarm[addr];
      msg.sender.send(b.deposit / REWARD_FRACTION); // reporter rewarded
      delete swarm[addr]; // rest of deposit burnt
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
