# multiaddr format
A validation checker for multiaddrs. Some basic validators for common address
types are provided, but creating your own combinations is easy.

Usage:
```go
a, _ := ma.NewMultiaddr("/ip4/5.2.67.3/tcp/1708")
TCP.Matches(a) // returns true
```

Making your own validators is easy, for example, the `Reliable` multiaddr is
defined as follows:

```go
// Define IP as either ipv4 or ipv6
var IP = Or(Base(ma.P_IP4), Base(ma.P_IP6))

// Define TCP as 'tcp' on top of either ipv4 or ipv6
var TCP = And(IP, Base(ma.P_TCP))

// Define UDP as 'udp' on top of either ipv4 or ipv6
var UDP = And(IP, Base(ma.P_UDP))

// Define UTP as 'utp' on top of udp (on top of ipv4 or ipv6)
var UTP = And(UDP, Base(ma.P_UTP))

// Now define a Reliable transport as either tcp or utp
var Reliable = Or(TCP, UTP)

// From here, we can easily define multiaddrs for protocols that can run on top
// of any 'reliable' transport (such as ipfs)
```

NOTE: the above patterns are already implemented in package
