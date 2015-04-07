contract Swarm
{

  struct Bee {
    uint deposit;
    uint expiry;
  }

  mapping (address => Bee) swarm;

  function max(uint a, uint b) private returns (uint c) {
    if(a >= b) return a;
    return b;
  }

  function signup(uint time) {
    Bee b = swarm[msg.sender];
    b.expiry = max(b.expiry, now) + time;
    b.deposit += msg.value;
  }

  function withdraw() {
    Bee b = swarm[msg.sender];
    if(now > b.expiry) {
	msg.sender.send(b.deposit);
    }
  }

}
