// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

//go:build linux

package nat

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"net"
	"os"
	"strconv"
	"strings"
)

func defaultGatewayIPs() []net.IP {
	data, err := os.ReadFile("/proc/net/route")
	if err != nil {
		return nil
	}
	return parseLinuxRouteTable(data)
}

func parseLinuxRouteTable(data []byte) []net.IP {
	var gws []net.IP

	scanner := bufio.NewScanner(bytes.NewReader(data))
	if !scanner.Scan() {
		return nil
	}
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 || fields[1] != "00000000" {
			continue
		}
		flags, err := strconv.ParseUint(fields[3], 16, 16)
		if err != nil || flags&0x2 == 0 {
			continue
		}
		gw, err := parseLinuxRouteHexIPv4(fields[2])
		if err != nil || gw == nil {
			continue
		}
		gws = append(gws, gw)
	}
	return gws
}

func parseLinuxRouteHexIPv4(s string) (net.IP, error) {
	n, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return nil, err
	}
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(n))
	return net.IPv4(buf[0], buf[1], buf[2], buf[3]).To4(), nil
}
