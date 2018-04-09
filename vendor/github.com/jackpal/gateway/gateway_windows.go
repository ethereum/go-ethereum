package gateway

import (
	"net"
	"os/exec"
	"syscall"
)

func DiscoverGateway() (ip net.IP, err error) {
	routeCmd := exec.Command("route", "print", "0.0.0.0")
	routeCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	output, err := routeCmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	return parseWindowsRoutePrint(output)
}
