// Package websocket implements a websocket based transport for go-libp2p.
package websocket

import (
	"fmt"
	"net/http"
	"net/url"

	ws "github.com/gorilla/websocket"
	tpt "github.com/libp2p/go-libp2p-transport"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"

	mafmt "github.com/whyrusleeping/mafmt"
)

// WsProtocol is the multiaddr protocol definition for this transport.
var WsProtocol = ma.Protocol{
	Code:  477,
	Name:  "ws",
	VCode: ma.CodeToVarint(477),
}

// WsFmt is multiaddr formatter for WsProtocol
var WsFmt = mafmt.And(mafmt.TCP, mafmt.Base(WsProtocol.Code))

// WsCodec is the multiaddr-net codec definition for the websocket transport
var WsCodec = &manet.NetCodec{
	NetAddrNetworks:  []string{"websocket"},
	ProtocolName:     "ws",
	ConvertMultiaddr: ConvertWebsocketMultiaddrToNetAddr,
	ParseNetAddr:     ParseWebsocketNetAddr,
}

// Default gorilla upgrader
var upgrader = ws.Upgrader{
	// Allow requests from *all* origins.
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func init() {
	err := ma.AddProtocol(WsProtocol)
	if err != nil {
		panic(fmt.Errorf("error registering websocket protocol: %s", err))
	}

	manet.RegisterNetCodec(WsCodec)
}

// WebsocketTransport is the actual go-libp2p transport
type WebsocketTransport struct{}

var _ tpt.Transport = (*WebsocketTransport)(nil)

func (t *WebsocketTransport) Matches(a ma.Multiaddr) bool {
	return WsFmt.Matches(a)
}

func (t *WebsocketTransport) Dialer(_ ma.Multiaddr, opts ...tpt.DialOpt) (tpt.Dialer, error) {
	return &dialer{}, nil
}

func (t *WebsocketTransport) Listen(a ma.Multiaddr) (tpt.Listener, error) {
	list, err := manet.Listen(a)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse("http://" + list.Addr().String())
	if err != nil {
		return nil, err
	}

	tlist := t.wrapListener(list, u)

	go http.Serve(list.NetListener(), tlist)

	return tlist, nil
}

func (t *WebsocketTransport) wrapListener(l manet.Listener, origin *url.URL) *listener {
	return &listener{
		Listener: l,
		incoming: make(chan *Conn),
		tpt:      t,
		origin:   origin,
	}
}
