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

package discover

import (
	"net"
	"testing"

	"github.com/ethereum/go-ethereum/internal/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

// This test runs v4 and v5 on one shared socket, as a devp2p bootnode would,
// and checks that v5 traffic forwarded through the shared connection is neither
// counted as bad v4 traffic nor counted twice in the ingress meter.
func TestMetricsSharedSocket(t *testing.T) {
	metrics.Enable()

	key := newkey()
	db, _ := enode.OpenDB("")
	defer db.Close()
	ln := enode.NewLocalNode(db, key)
	socket, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IP{127, 0, 0, 1}})
	if err != nil {
		t.Fatal(err)
	}
	realaddr := socket.LocalAddr().(*net.UDPAddr)
	ln.SetStaticIP(realaddr.IP)
	ln.SetFallbackUDP(realaddr.Port)

	cfg := Config{PrivateKey: key, Log: testlog.Logger(t, log.LevelTrace)}
	unhandled := make(chan ReadPacket, 100)
	v4cfg := cfg
	v4cfg.Unhandled = unhandled
	v4, err := ListenV4(socket, ln, v4cfg)
	if err != nil {
		t.Fatal(err)
	}
	v5, err := ListenV5(&SharedUDPConn{UDPConn: socket, Unhandled: unhandled}, ln, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		v4.Close()
		v5.Close()
	}()
	remote := startLocalhostV5(t, Config{})
	defer remote.Close()

	var (
		v4bad   = v4BadPacketMeter.Snapshot().Count()
		v5bad   = v5BadPacketMeter.Snapshot().Count()
		ingress = ingressTrafficMeter.Snapshot().Count()
		egress  = egressTrafficMeter.Snapshot().Count()
		pings   = metrics.GetOrRegisterMeter(ingressMeterName+"/PING/v5", nil).Snapshot().Count()
	)
	if _, err := remote.Ping(ln.Node()); err != nil {
		t.Fatal(err)
	}
	if n := v4BadPacketMeter.Snapshot().Count() - v4bad; n != 0 {
		t.Errorf("v4 bad packet meter counted %d forwarded v5 packets", n)
	}
	if n := v5BadPacketMeter.Snapshot().Count() - v5bad; n != 0 {
		t.Errorf("v5 bad packet meter counted %d packets", n)
	}
	if n := metrics.GetOrRegisterMeter(ingressMeterName+"/PING/v5", nil).Snapshot().Count() - pings; n != 1 {
		t.Errorf("PING/v5 ingress meter counted %d, want 1", n)
	}
	// Every packet crossed localhost between two metered sockets, so bytes
	// received can not exceed bytes sent unless the shared read counted twice.
	in, out := ingressTrafficMeter.Snapshot().Count()-ingress, egressTrafficMeter.Snapshot().Count()-egress
	if in > out {
		t.Errorf("ingress %d bytes > egress %d bytes: shared socket reads counted twice", in, out)
	}
}
