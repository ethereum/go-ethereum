# go-multiaddr-filter -- CIDR netmasks with multiaddr

This module creates very simple [multiaddr](https://github.com/jbenet/go-multiaddr) formatted cidr netmasks.

It doesn't do full multiaddr parsing to save on vendoring things and perf. The `net` package will take care of verifying the validity of the network part anyway.

## Usage

```go

import filter "github.com/whyrusleeping/multiaddr-filter"

filter.NewMask("/ip4/192.168.0.0/24") // ipv4
filter.NewMask("/ip6/fe80::/64") // ipv6
```
