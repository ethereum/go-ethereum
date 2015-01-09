package p2p

import (
	"fmt"
	"net"
)

func ParseNAT(natType string, gateway string) (nat NAT, err error) {
	switch natType {
	case "UPNP":
		nat = UPNP()
	case "PMP":
		ip := net.ParseIP(gateway)
		if ip == nil {
			return nil, fmt.Errorf("cannot resolve PMP gateway IP %s", gateway)
		}
		nat = PMP(ip)
	case "":
	default:
		return nil, fmt.Errorf("unrecognised NAT type '%s'", natType)
	}
	return
}
