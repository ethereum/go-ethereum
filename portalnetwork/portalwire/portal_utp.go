package portalwire

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"sync"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/discover/v5wire"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/netutil"
	"github.com/optimism-java/utp-go"
	"github.com/optimism-java/utp-go/libutp"
	"go.uber.org/zap"
)

type PortalUtp struct {
	ctx          context.Context
	log          log.Logger
	discV5       *discover.UDPv5
	conn         discover.UDPConn
	ListenAddr   string
	listener     *utp.Listener
	utpSm        *utp.SocketManager
	packetRouter *utp.PacketRouter
	lAddr        *utp.Addr

	startOnce sync.Once
}

func NewPortalUtp(ctx context.Context, config *PortalProtocolConfig, discV5 *discover.UDPv5, conn discover.UDPConn) *PortalUtp {
	return &PortalUtp{
		ctx:        ctx,
		log:        log.New("protocol", "utp", "local", conn.LocalAddr().String()),
		discV5:     discV5,
		conn:       conn,
		ListenAddr: config.ListenAddr,
	}
}

func (p *PortalUtp) Start() error {
	var err error
	go p.startOnce.Do(func() {
		var logger *zap.Logger
		if p.log.Enabled(p.ctx, log.LevelDebug) || p.log.Enabled(p.ctx, log.LevelTrace) {
			logger, err = zap.NewDevelopmentConfig().Build()
		} else {
			logger, err = zap.NewProductionConfig().Build()
		}
		if err != nil {
			return
		}

		laddr := p.getLocalAddr()
		p.packetRouter = utp.NewPacketRouter(p.packetRouterFunc)
		p.utpSm, err = utp.NewSocketManagerWithOptions(
			"utp",
			laddr,
			utp.WithContext(p.ctx),
			utp.WithLogger(logger.Named(p.ListenAddr)),
			utp.WithPacketRouter(p.packetRouter),
			utp.WithMaxPacketSize(1145))
		if err != nil {
			return
		}
		p.listener, err = utp.ListenUTPOptions("utp", (*utp.Addr)(laddr), utp.WithSocketManager(p.utpSm))
		if err != nil {
			return
		}
		p.lAddr = p.listener.Addr().(*utp.Addr)

		// register discv5 listener
		p.discV5.RegisterTalkHandler(string(Utp), p.handleUtpTalkRequest)
	})

	return err
}

func (p *PortalUtp) Stop() {
	err := p.listener.Close()
	if err != nil {
		p.log.Error("close utp listener has error", "error", err)
	}
	p.discV5.Close()
}

func (p *PortalUtp) DialWithCid(ctx context.Context, dest *enode.Node, connId uint16) (net.Conn, error) {
	raddr := &utp.Addr{IP: dest.IP(), Port: dest.UDP()}
	p.log.Debug("will connect to: ", "nodeId", dest.ID().String(), "connId", connId)
	conn, err := utp.DialUTPOptions("utp", p.lAddr, raddr, utp.WithContext(ctx), utp.WithSocketManager(p.utpSm), utp.WithConnId(connId))
	return conn, err
}

func (p *PortalUtp) Dial(ctx context.Context, dest *enode.Node) (net.Conn, error) {
	raddr := &utp.Addr{IP: dest.IP(), Port: dest.UDP()}
	p.log.Info("will connect to: ", "addr", raddr.String())
	conn, err := utp.DialUTPOptions("utp", p.lAddr, raddr, utp.WithContext(ctx), utp.WithSocketManager(p.utpSm))
	return conn, err
}

func (p *PortalUtp) AcceptWithCid(ctx context.Context, nodeId enode.ID, cid *libutp.ConnId) (*utp.Conn, error) {
	p.log.Debug("will accept from: ", "nodeId", nodeId.String(), "sendId", cid.SendId(), "recvId", cid.RecvId())
	return p.listener.AcceptUTPContext(ctx, nodeId, cid)
}

func (p *PortalUtp) Accept(ctx context.Context) (*utp.Conn, error) {
	return p.listener.AcceptUTPContext(ctx, enode.ID{}, nil)
}

func (p *PortalUtp) getLocalAddr() *net.UDPAddr {
	laddr := p.conn.LocalAddr().(*net.UDPAddr)
	p.log.Debug("UDP listener up", "addr", laddr)
	return laddr
}

func (p *PortalUtp) packetRouterFunc(buf []byte, id enode.ID, addr *net.UDPAddr) (int, error) {
	p.log.Info("will send to target data", "nodeId", id.String(), "ip", addr.IP.To4().String(), "port", addr.Port, "bufLength", len(buf))

	if n, ok := p.discV5.GetCachedNode(addr.String()); ok {
		//_, err := p.DiscV5.TalkRequestToID(id, addr, string(portalwire.UTPNetwork), buf)
		req := &v5wire.TalkRequest{Protocol: string(Utp), Message: buf}
		p.discV5.SendFromAnotherThreadWithNode(n, netip.AddrPortFrom(netutil.IPToAddr(addr.IP), uint16(addr.Port)), req)

		return len(buf), nil
	} else {
		p.log.Warn("not found target node info", "ip", addr.IP.To4().String(), "port", addr.Port, "bufLength", len(buf))
		return 0, fmt.Errorf("not found target node id")
	}
}

func (p *PortalUtp) handleUtpTalkRequest(id enode.ID, addr *net.UDPAddr, msg []byte) []byte {
	p.log.Trace("receive utp data", "nodeId", id.String(), "addr", addr, "msg-length", len(msg))
	p.packetRouter.ReceiveMessage(msg, &utp.NodeInfo{Id: id, Addr: addr})
	return []byte("")
}
