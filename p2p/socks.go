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

package p2p

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"golang.org/x/net/proxy"
)

// errDiscoveryNotProxied is returned when a SOCKS proxy is configured while node
// discovery is still enabled. Discovery runs over UDP and is not carried by the
// proxy, so leaving it enabled would announce the node's real endpoint to the
// network and defeat the purpose of the proxy.
var errDiscoveryNotProxied = errors.New("p2p: SocksProxy is set but node discovery is enabled: " +
	"discovery runs over UDP and is not tunneled through the proxy, which would reveal the real " +
	"address of this node. Disable discovery (geth --nodiscover) to dial through a proxy")

// socksDialer implements NodeDialer by tunneling outbound peer connections through
// a SOCKS5 proxy.
type socksDialer struct {
	d proxy.ContextDialer
}

// newSocksDialer creates a NodeDialer which routes outbound connections through the
// SOCKS5 proxy at rawurl.
//
// The URL must use the socks5 or socks5h scheme and may carry credentials for
// username/password authentication, for example:
//
//	socks5://127.0.0.1:9050
//	socks5://user:password@127.0.0.1:1080
//
// Both schemes behave identically here: peer addresses come from enode records and
// are always IP addresses, so there is no host name for the proxy to resolve and no
// name resolution can leak.
func newSocksDialer(rawurl string, forward *net.Dialer) (NodeDialer, error) {
	// Reject a bare host:port up front. url.Parse fails on it with a message about
	// path segments, which is not a useful thing to show the user.
	if !strings.Contains(rawurl, "://") {
		return nil, fmt.Errorf("invalid SOCKS proxy URL %q: missing scheme, want socks5://host:port", rawurl)
	}
	u, err := url.Parse(rawurl)
	if err != nil {
		return nil, fmt.Errorf("invalid SOCKS proxy URL %q: %v", rawurl, err)
	}
	switch u.Scheme {
	case "socks5", "socks5h":
	default:
		return nil, fmt.Errorf("invalid SOCKS proxy URL %q: unsupported scheme %q, want socks5 or socks5h", rawurl, u.Scheme)
	}
	if _, _, err := net.SplitHostPort(u.Host); err != nil {
		return nil, fmt.Errorf("invalid SOCKS proxy URL %q: address must be host:port", rawurl)
	}

	var auth *proxy.Auth
	if u.User != nil {
		password, _ := u.User.Password()
		auth = &proxy.Auth{User: u.User.Username(), Password: password}
	}
	d, err := proxy.SOCKS5("tcp", u.Host, auth, forward)
	if err != nil {
		return nil, fmt.Errorf("can't create SOCKS proxy dialer for %q: %v", rawurl, err)
	}
	// The dialer returned by proxy.SOCKS5 always implements ContextDialer, but the
	// declared return type does not say so.
	ctxDialer, ok := d.(proxy.ContextDialer)
	if !ok {
		return nil, fmt.Errorf("SOCKS proxy dialer for %q does not support contexts", rawurl)
	}
	return socksDialer{d: ctxDialer}, nil
}

func (s socksDialer) Dial(ctx context.Context, dest *enode.Node) (net.Conn, error) {
	addr, ok := dest.TCPEndpoint()
	if !ok {
		return nil, errNoPort
	}
	return s.d.DialContext(ctx, "tcp", addr.String())
}
