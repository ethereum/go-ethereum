package investigate

import (
	"fmt"
	"net"
	"testing"
)

func TestDNS(t *testing.T) {
	for i := 0; i < 100; i++ {
		ips, err := net.LookupIP("invalid.")
		fmt.Println(i, "result", ips, "err", err)
	}
}
