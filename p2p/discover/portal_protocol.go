package discover

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/big"
	"math/rand"
	"net"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover/portalwire"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/p2p/netutil"
	"github.com/ethereum/go-ethereum/portalnetwork/storage"
	"github.com/ethereum/go-ethereum/rlp"
	ssz "github.com/ferranbt/fastssz"
	"github.com/holiman/uint256"
	"github.com/optimism-java/utp-go"
	"github.com/optimism-java/utp-go/libutp"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/tetratelabs/wabin/leb128"
	"go.uber.org/zap"
)

const (

	// TalkResp message is a response message so the session is established and a
	// regular discv5 packet is assumed for size calculation.
	// Regular message = IV + header + message
	// talkResp message = rlp: [request-id, response]
	talkRespOverhead = 16 + // IV size
		55 + // header size
		1 + // talkResp msg id
		3 + // rlp encoding outer list, max length will be encoded in 2 bytes
		9 + // request id (max = 8) + 1 byte from rlp encoding byte string
		3 + // rlp encoding response byte string, max length in 2 bytes
		16 // HMAC

	portalFindnodesResultLimit = 32

	defaultUTPConnectTimeout = 15 * time.Second

	defaultUTPWriteTimeout = 60 * time.Second

	defaultUTPReadTimeout = 60 * time.Second

	// These are the concurrent offers per Portal wire protocol that is running.
	// Using the `offerQueue` allows for limiting the amount of offers send and
	// thus how many streams can be started.
	// TODO:
	// More thought needs to go into this as it is currently on a per network
	// basis. Keep it simple like that? Or limit it better at the stream transport
	// level? In the latter case, this might still need to be checked/blocked at
	// the very start of sending the offer, because blocking/waiting too long
	// between the received accept message and actually starting the stream and
	// sending data could give issues due to timeouts on the other side.
	// And then there are still limits to be applied also for FindContent and the
	// incoming directions.
	concurrentOffers = 50
)

const (
	TransientOfferRequestKind byte = 0x01
	PersistOfferRequestKind   byte = 0x02
)

type ClientTag string

func (c ClientTag) ENRKey() string { return "c" }

const Tag ClientTag = "shisui"

var ErrNilContentKey = errors.New("content key cannot be nil")

var ContentNotFound = storage.ErrContentNotFound

var ErrEmptyResp = errors.New("empty resp")

var MaxDistance = hexutil.MustDecode("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")

type ContentElement struct {
	Node        enode.ID
	ContentKeys [][]byte
	Contents    [][]byte
}

type ContentEntry struct {
	ContentKey []byte
	Content    []byte
}

type TransientOfferRequest struct {
	Contents []*ContentEntry
}

type PersistOfferRequest struct {
	ContentKeys [][]byte
}

type OfferRequest struct {
	Kind    byte
	Request interface{}
}

type OfferRequestWithNode struct {
	Request *OfferRequest
	Node    *enode.Node
}

type ContentInfoResp struct {
	Content     []byte
	UtpTransfer bool
}

type traceContentInfoResp struct {
	Node        *enode.Node
	Flag        byte
	Content     any
	UtpTransfer bool
}

type PortalProtocolOption func(p *PortalProtocol)

type PortalProtocolConfig struct {
	BootstrapNodes  []*enode.Node
	NodeIP          net.IP
	ListenAddr      string
	NetRestrict     *netutil.Netlist
	NodeRadius      *uint256.Int
	RadiusCacheSize int
	NodeDBPath      string
}

func DefaultPortalProtocolConfig() *PortalProtocolConfig {
	nodeRadius, _ := uint256.FromHex("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	return &PortalProtocolConfig{
		BootstrapNodes:  make([]*enode.Node, 0),
		ListenAddr:      ":9009",
		NetRestrict:     nil,
		NodeRadius:      nodeRadius,
		RadiusCacheSize: 32 * 1024 * 1024,
		NodeDBPath:      "",
	}
}

type PortalProtocol struct {
	table         *Table
	cachedIdsLock sync.Mutex
	cachedIds     map[string]enode.ID

	protocolId   string
	protocolName string

	nodeRadius     *uint256.Int
	DiscV5         *UDPv5
	utp            *utp.Listener
	utpSm          *utp.SocketManager
	packetRouter   *utp.PacketRouter
	connIdGen      libutp.ConnIdGenerator
	ListenAddr     string
	localNode      *enode.LocalNode
	Log            log.Logger
	PrivateKey     *ecdsa.PrivateKey
	NetRestrict    *netutil.Netlist
	BootstrapNodes []*enode.Node
	conn           UDPConn

	validSchemes   enr.IdentityScheme
	radiusCache    *fastcache.Cache
	closeCtx       context.Context
	cancelCloseCtx context.CancelFunc
	storage        storage.ContentStorage
	toContentId    func(contentKey []byte) []byte

	contentQueue chan *ContentElement
	offerQueue   chan *OfferRequestWithNode
}

func defaultContentIdFunc(contentKey []byte) []byte {
	digest := sha256.Sum256(contentKey)
	return digest[:]
}

func NewPortalProtocol(config *PortalProtocolConfig, protocolId string, privateKey *ecdsa.PrivateKey, conn UDPConn, localNode *enode.LocalNode, discV5 *UDPv5, storage storage.ContentStorage, contentQueue chan *ContentElement, opts ...PortalProtocolOption) (*PortalProtocol, error) {
	closeCtx, cancelCloseCtx := context.WithCancel(context.Background())

	protocolName := portalwire.NetworkNameMap[protocolId]
	protocol := &PortalProtocol{
		cachedIds:      make(map[string]enode.ID),
		protocolId:     protocolId,
		protocolName:   protocolName,
		ListenAddr:     config.ListenAddr,
		Log:            log.New("protocol", protocolName),
		PrivateKey:     privateKey,
		NetRestrict:    config.NetRestrict,
		BootstrapNodes: config.BootstrapNodes,
		nodeRadius:     config.NodeRadius,
		radiusCache:    fastcache.New(config.RadiusCacheSize),
		closeCtx:       closeCtx,
		cancelCloseCtx: cancelCloseCtx,
		localNode:      localNode,
		validSchemes:   enode.ValidSchemes,
		storage:        storage,
		toContentId:    defaultContentIdFunc,
		contentQueue:   contentQueue,
		offerQueue:     make(chan *OfferRequestWithNode, concurrentOffers),
		conn:           conn,
		DiscV5:         discV5,
	}

	for _, opt := range opts {
		opt(protocol)
	}

	return protocol, nil
}

func (p *PortalProtocol) Start() error {
	err := p.setupDiscV5AndTable()
	if err != nil {
		return err
	}

	p.DiscV5.RegisterTalkHandler(p.protocolId, p.handleTalkRequest)
	p.DiscV5.RegisterTalkHandler(string(portalwire.UTPNetwork), p.handleUtpTalkRequest)

	go p.table.loop()

	for i := 0; i < concurrentOffers; i++ {
		go p.offerWorker()
	}

	// wait for the routing table to be initialized
	<-p.table.initDone
	return nil
}

func (p *PortalProtocol) Stop() {
	p.cancelCloseCtx()
	p.table.close()
	p.DiscV5.Close()
	err := p.utp.Close()
	if err != nil {
		p.Log.Error("failed to close utp listener", "err", err)
	}
}
func (p *PortalProtocol) RoutingTableInfo() [][]string {
	p.table.mutex.Lock()
	defer p.table.mutex.Unlock()
	nodes := make([][]string, 0)
	for _, b := range &p.table.buckets {
		bucketNodes := make([]string, 0)
		for _, n := range b.entries {
			bucketNodes = append(bucketNodes, unwrapNode(n).ID().String())
		}
		nodes = append(nodes, bucketNodes)
	}
	p.Log.Trace("routingTableInfo resp:", "nodes", nodes)
	return nodes
}

func (p *PortalProtocol) AddEnr(n *enode.Node) {
	p.setJustSeen(n)
	id := n.ID().String()
	p.radiusCache.Set([]byte(id), MaxDistance)
}

func (p *PortalProtocol) setupUDPListening() error {
	laddr := p.conn.LocalAddr().(*net.UDPAddr)
	p.localNode.SetFallbackUDP(laddr.Port)
	p.Log.Debug("UDP listener up", "addr", laddr)
	// TODO: NAT
	//if !laddr.IP.IsLoopback() && !laddr.IP.IsPrivate() {
	//	srv.portMappingRegister <- &portMapping{
	//		protocol: "UDP",
	//		name:     "ethereum peer discovery",
	//		port:     laddr.Port,
	//	}
	//}

	var err error
	p.packetRouter = utp.NewPacketRouter(
		func(buf []byte, addr *net.UDPAddr) (int, error) {
			p.Log.Info("will send to target data", "ip", addr.IP.To4().String(), "port", addr.Port, "bufLength", len(buf))

			p.cachedIdsLock.Lock()
			defer p.cachedIdsLock.Unlock()
			if id, ok := p.cachedIds[addr.String()]; ok {
				_, err := p.DiscV5.TalkRequestToID(id, addr, string(portalwire.UTPNetwork), buf)
				return len(buf), err
			} else {
				p.Log.Warn("not found target node info", "ip", addr.IP.To4().String(), "port", addr.Port, "bufLength", len(buf))
				return 0, fmt.Errorf("not found target node id")
			}
		})

	ctx := context.Background()
	var logger *zap.Logger

	if p.Log.Enabled(ctx, log.LevelDebug) || p.Log.Enabled(ctx, log.LevelTrace) {
		logger, err = zap.NewDevelopmentConfig().Build()
	} else {
		logger, err = zap.NewProductionConfig().Build()
	}

	if err != nil {
		return err
	}
	p.utpSm, err = utp.NewSocketManagerWithOptions("utp", laddr, utp.WithLogger(logger.Named(p.ListenAddr)), utp.WithPacketRouter(p.packetRouter), utp.WithMaxPacketSize(1145))
	if err != nil {
		return err
	}
	p.utp, err = utp.ListenUTPOptions("utp", (*utp.Addr)(laddr), utp.WithSocketManager(p.utpSm))

	p.connIdGen = utp.NewConnIdGenerator()
	if err != nil {
		return err
	}
	return nil
}

func (p *PortalProtocol) setupDiscV5AndTable() error {
	err := p.setupUDPListening()
	if err != nil {
		return err
	}

	cfg := Config{
		PrivateKey:  p.PrivateKey,
		NetRestrict: p.NetRestrict,
		Bootnodes:   p.BootstrapNodes,
		Log:         p.Log,
	}

	p.table, err = newMeteredTable(p, p.localNode.Database(), cfg)
	if err != nil {
		return err
	}

	return nil
}

func (p *PortalProtocol) putCacheNodeId(node *enode.Node) {
	p.cachedIdsLock.Lock()
	defer p.cachedIdsLock.Unlock()
	addr := &net.UDPAddr{IP: node.IP(), Port: node.UDP()}
	if _, ok := p.cachedIds[addr.String()]; !ok {
		p.cachedIds[addr.String()] = node.ID()
	}
}

func (p *PortalProtocol) putCacheId(id enode.ID, addr *net.UDPAddr) {
	p.cachedIdsLock.Lock()
	defer p.cachedIdsLock.Unlock()
	if _, ok := p.cachedIds[addr.String()]; !ok {
		p.cachedIds[addr.String()] = id
	}
}

func (p *PortalProtocol) ping(node *enode.Node) (uint64, error) {
	pong, err := p.pingInner(node)
	if err != nil {
		return 0, err
	}

	return pong.EnrSeq, nil
}

func (p *PortalProtocol) pingInner(node *enode.Node) (*portalwire.Pong, error) {
	enrSeq := p.Self().Seq()
	radiusBytes, err := p.nodeRadius.MarshalSSZ()
	if err != nil {
		return nil, err
	}
	customPayload := &portalwire.PingPongCustomData{
		Radius: radiusBytes,
	}

	customPayloadBytes, err := customPayload.MarshalSSZ()
	if err != nil {
		return nil, err
	}

	pingRequest := &portalwire.Ping{
		EnrSeq:        enrSeq,
		CustomPayload: customPayloadBytes,
	}

	p.Log.Trace("Sending ping request", "protocol", p.protocolName, "ip", p.Self().IP().String(), "source", p.Self().ID(), "target", node.ID(), "ping", pingRequest)
	pingRequestBytes, err := pingRequest.MarshalSSZ()
	if err != nil {
		return nil, err
	}

	talkRequestBytes := make([]byte, 0, len(pingRequestBytes)+1)
	talkRequestBytes = append(talkRequestBytes, portalwire.PING)
	talkRequestBytes = append(talkRequestBytes, pingRequestBytes...)

	talkResp, err := p.DiscV5.TalkRequest(node, p.protocolId, talkRequestBytes)

	if err != nil {
		p.Log.Error("ping error:", "ip", node.IP().String(), "port", node.UDP(), "err", err)
		p.replaceNode(node)
		return nil, err
	}

	p.Log.Trace("Received ping response", "source", p.Self().ID(), "target", node.ID(), "res", talkResp)

	return p.processPong(node, talkResp)
}

func (p *PortalProtocol) findNodes(node *enode.Node, distances []uint) ([]*enode.Node, error) {
	distancesBytes := make([][2]byte, len(distances))
	for i, distance := range distances {
		copy(distancesBytes[i][:], ssz.MarshalUint16(make([]byte, 0), uint16(distance)))
	}

	findNodes := &portalwire.FindNodes{
		Distances: distancesBytes,
	}

	p.Log.Trace("Sending find nodes request", "id", node.ID(), "findNodes", findNodes)
	findNodesBytes, err := findNodes.MarshalSSZ()
	if err != nil {
		p.Log.Error("failed to marshal find nodes request", "err", err)
		return nil, err
	}

	talkRequestBytes := make([]byte, 0, len(findNodesBytes)+1)
	talkRequestBytes = append(talkRequestBytes, portalwire.FINDNODES)
	talkRequestBytes = append(talkRequestBytes, findNodesBytes...)

	talkResp, err := p.DiscV5.TalkRequest(node, p.protocolId, talkRequestBytes)
	if err != nil {
		p.Log.Error("failed to send find nodes request", "ip", node.IP().String(), "port", node.UDP(), "err", err)
		return nil, err
	}

	return p.processNodes(node, talkResp, distances)
}

func (p *PortalProtocol) findContent(node *enode.Node, contentKey []byte) (byte, interface{}, error) {
	findContent := &portalwire.FindContent{
		ContentKey: contentKey,
	}

	p.Log.Trace("Sending find content request", "id", node.ID(), "findContent", findContent)
	findContentBytes, err := findContent.MarshalSSZ()
	if err != nil {
		p.Log.Error("failed to marshal find content request", "err", err)
		return 0xff, nil, err
	}

	talkRequestBytes := make([]byte, 0, len(findContentBytes)+1)
	talkRequestBytes = append(talkRequestBytes, portalwire.FINDCONTENT)
	talkRequestBytes = append(talkRequestBytes, findContentBytes...)

	talkResp, err := p.DiscV5.TalkRequest(node, p.protocolId, talkRequestBytes)
	if err != nil {
		p.Log.Error("failed to send find content request", "ip", node.IP().String(), "port", node.UDP(), "err", err)
		return 0xff, nil, err
	}

	return p.processContent(node, talkResp)
}

func (p *PortalProtocol) offer(node *enode.Node, offerRequest *OfferRequest) ([]byte, error) {
	contentKeys := getContentKeys(offerRequest)

	offer := &portalwire.Offer{
		ContentKeys: contentKeys,
	}

	p.Log.Trace("Sending offer request", "offer", offer)
	offerBytes, err := offer.MarshalSSZ()
	if err != nil {
		p.Log.Error("failed to marshal offer request", "err", err)
		return nil, err
	}

	talkRequestBytes := make([]byte, 0, len(offerBytes)+1)
	talkRequestBytes = append(talkRequestBytes, portalwire.OFFER)
	talkRequestBytes = append(talkRequestBytes, offerBytes...)

	talkResp, err := p.DiscV5.TalkRequest(node, p.protocolId, talkRequestBytes)
	if err != nil {
		p.Log.Error("failed to send offer request", "err", err)
		return nil, err
	}

	return p.processOffer(node, talkResp, offerRequest)
}

func (p *PortalProtocol) processOffer(target *enode.Node, resp []byte, request *OfferRequest) ([]byte, error) {
	var err error
	if len(resp) == 0 {
		return nil, ErrEmptyResp
	}
	if resp[0] != portalwire.ACCEPT {
		return nil, fmt.Errorf("invalid accept response")
	}

	p.Log.Info("will process Offer", "id", target.ID(), "ip", target.IP().To4().String(), "port", target.UDP())
	p.putCacheNodeId(target)

	accept := &portalwire.Accept{}
	err = accept.UnmarshalSSZ(resp[1:])
	if err != nil {
		return nil, err
	}

	p.Log.Trace("Received accept response", "id", target.ID(), "accept", accept)
	p.setJustSeen(target)

	var contentKeyLen int
	if request.Kind == TransientOfferRequestKind {
		contentKeyLen = len(request.Request.(*TransientOfferRequest).Contents)
	} else {
		contentKeyLen = len(request.Request.(*PersistOfferRequest).ContentKeys)
	}

	contentKeyBitlist := bitfield.Bitlist(accept.ContentKeys)
	if contentKeyBitlist.Len() != uint64(contentKeyLen) {
		return nil, fmt.Errorf("accepted content key bitlist has invalid size, expected %d, got %d", contentKeyLen, contentKeyBitlist.Len())
	}

	if contentKeyBitlist.Count() == 0 {
		return nil, nil
	}

	connId := binary.BigEndian.Uint16(accept.ConnectionId[:])
	go func(ctx context.Context) {
		var conn net.Conn
		for {
			select {
			case <-ctx.Done():
				return
			default:
				contents := make([][]byte, 0, contentKeyBitlist.Count())
				var content []byte
				if request.Kind == TransientOfferRequestKind {
					for _, index := range contentKeyBitlist.BitIndices() {
						content = request.Request.(*TransientOfferRequest).Contents[index].Content
						contents = append(contents, content)
					}
				} else {
					for _, index := range contentKeyBitlist.BitIndices() {
						contentKey := request.Request.(*PersistOfferRequest).ContentKeys[index]
						contentId := p.toContentId(contentKey)
						if contentId != nil {
							content, err = p.storage.Get(contentKey, contentId)
							if err != nil {
								p.Log.Error("failed to get content from storage", "err", err)
								contents = append(contents, []byte{})
							} else {
								contents = append(contents, content)
							}
						} else {
							contents = append(contents, []byte{})
						}
					}
				}

				var contentsPayload []byte
				contentsPayload, err = encodeContents(contents)
				if err != nil {
					p.Log.Error("failed to encode contents", "err", err)
					return
				}

				connctx, conncancel := context.WithTimeout(ctx, defaultUTPConnectTimeout)
				laddr := p.utp.Addr().(*utp.Addr)
				raddr := &utp.Addr{IP: target.IP(), Port: target.UDP()}
				p.Log.Info("will connect to: ", "addr", raddr.String(), "connId", connId)
				conn, err = utp.DialUTPOptions("utp", laddr, raddr, utp.WithContext(connctx), utp.WithSocketManager(p.utpSm), utp.WithConnId(uint32(connId)))
				if err != nil {
					conncancel()
					p.Log.Error("failed to dial utp connection", "err", err)
					return
				}
				conncancel()

				err = conn.SetWriteDeadline(time.Now().Add(defaultUTPWriteTimeout))
				if err != nil {
					p.Log.Error("failed to set write deadline", "err", err)
					err = conn.Close()
					if err != nil {
						p.Log.Error("failed to close utp connection", "err", err)
						return
					}

					return
				}

				var written int
				written, err = conn.Write(contentsPayload)
				if err != nil {
					p.Log.Error("failed to write to utp connection", "err", err)
					err = conn.Close()
					if err != nil {
						p.Log.Error("failed to close utp connection", "err", err)
						return
					}
					return
				}
				p.Log.Trace("Sent content response", "id", target.ID(), "contents", contents, "size", written)
				err = conn.Close()
				if err != nil {
					p.Log.Error("failed to close utp connection", "err", err)
					return
				}
				return
			}
		}
	}(p.closeCtx)

	return accept.ContentKeys, nil
}

func (p *PortalProtocol) processContent(target *enode.Node, resp []byte) (byte, interface{}, error) {
	if len(resp) == 0 {
		return 0x00, nil, ErrEmptyResp
	}

	if resp[0] != portalwire.CONTENT {
		return 0xff, nil, fmt.Errorf("invalid content response")
	}

	p.Log.Info("will process content", "id", target.ID(), "ip", target.IP().To4().String(), "port", target.UDP())
	p.putCacheNodeId(target)

	switch resp[1] {
	case portalwire.ContentRawSelector:
		content := &portalwire.Content{}
		err := content.UnmarshalSSZ(resp[2:])
		if err != nil {
			return 0xff, nil, err
		}

		p.Log.Trace("Received content response", "id", target.ID(), "content", content)
		p.setJustSeen(target)
		return resp[1], content.Content, nil
	case portalwire.ContentConnIdSelector:
		connIdMsg := &portalwire.ConnectionId{}
		err := connIdMsg.UnmarshalSSZ(resp[2:])
		if err != nil {
			return 0xff, nil, err
		}

		p.Log.Trace("Received returned content response", "id", target.ID(), "connIdMsg", connIdMsg)
		p.setJustSeen(target)
		connctx, conncancel := context.WithTimeout(p.closeCtx, defaultUTPConnectTimeout)
		laddr := p.utp.Addr().(*utp.Addr)
		raddr := &utp.Addr{IP: target.IP(), Port: target.UDP()}
		connId := binary.BigEndian.Uint16(connIdMsg.Id[:])
		p.Log.Info("will connect to: ", "addr", raddr.String(), "connId", connId)
		conn, err := utp.DialUTPOptions("utp", laddr, raddr, utp.WithContext(connctx), utp.WithSocketManager(p.utpSm), utp.WithConnId(uint32(connId)))
		if err != nil {
			conncancel()
			return 0xff, nil, err
		}
		conncancel()

		err = conn.SetReadDeadline(time.Now().Add(defaultUTPReadTimeout))
		if err != nil {
			return 0xff, nil, err
		}
		// Read ALL the data from the connection until EOF and return it
		data, err := io.ReadAll(conn)
		if err != nil {
			p.Log.Error("failed to read from utp connection", "err", err)
			return 0xff, nil, err
		}
		p.Log.Trace("Received content response", "id", target.ID(), "size", len(data), "data", data)
		return resp[1], data, nil
	case portalwire.ContentEnrsSelector:
		enrs := &portalwire.Enrs{}
		err := enrs.UnmarshalSSZ(resp[2:])

		if err != nil {
			return 0xff, nil, err
		}

		p.Log.Trace("Received content response", "id", target.ID(), "enrs", enrs)
		p.setJustSeen(target)
		nodes := p.filterNodes(target, enrs.Enrs, nil)
		return resp[1], nodes, nil
	default:
		return 0xff, nil, fmt.Errorf("invalid content response")
	}
}

func (p *PortalProtocol) processNodes(target *enode.Node, resp []byte, distances []uint) ([]*enode.Node, error) {
	if len(resp) == 0 {
		return nil, ErrEmptyResp
	}

	if resp[0] != portalwire.NODES {
		return nil, fmt.Errorf("invalid nodes response")
	}

	nodesResp := &portalwire.Nodes{}
	err := nodesResp.UnmarshalSSZ(resp[1:])
	if err != nil {
		return nil, err
	}

	p.setJustSeen(target)
	nodes := p.filterNodes(target, nodesResp.Enrs, distances)

	return nodes, nil
}

func (p *PortalProtocol) setJustSeen(target *enode.Node) {
	wn := wrapNode(target)
	wn.livenessChecks++
	p.table.addVerifiedNode(wn)
}

func (p *PortalProtocol) filterNodes(target *enode.Node, enrs [][]byte, distances []uint) []*enode.Node {
	var (
		nodes    []*enode.Node
		seen     = make(map[enode.ID]struct{})
		err      error
		verified = 0
		n        *enode.Node
	)

	for _, b := range enrs {
		record := &enr.Record{}
		err = rlp.DecodeBytes(b, record)
		if err != nil {
			p.Log.Error("Invalid record in nodes response", "id", target.ID(), "err", err)
			continue
		}
		n, err = p.verifyResponseNode(target, record, distances, seen)
		if err != nil {
			p.Log.Error("Invalid record in nodes response", "id", target.ID(), "err", err)
			continue
		}
		verified++
		nodes = append(nodes, n)
	}

	p.Log.Trace("Received nodes response", "id", target.ID(), "total", len(enrs), "verified", verified, "nodes", nodes)
	return nodes
}

func (p *PortalProtocol) processPong(target *enode.Node, resp []byte) (*portalwire.Pong, error) {
	if len(resp) == 0 {
		return nil, ErrEmptyResp
	}
	if resp[0] != portalwire.PONG {
		return nil, fmt.Errorf("invalid pong response")
	}
	pong := &portalwire.Pong{}
	err := pong.UnmarshalSSZ(resp[1:])
	if err != nil {
		p.replaceNode(target)
		return nil, err
	}

	p.Log.Trace("Received pong response", "id", target.ID(), "pong", pong)

	customPayload := &portalwire.PingPongCustomData{}
	err = customPayload.UnmarshalSSZ(pong.CustomPayload)
	if err != nil {
		p.replaceNode(target)
		return nil, err
	}

	p.Log.Trace("Received pong response", "id", target.ID(), "pong", pong, "customPayload", customPayload)
	p.setJustSeen(target)

	p.radiusCache.Set([]byte(target.ID().String()), customPayload.Radius)
	return pong, nil
}

func (p *PortalProtocol) handleUtpTalkRequest(id enode.ID, addr *net.UDPAddr, msg []byte) []byte {
	if n := p.DiscV5.getNode(id); n != nil {
		p.table.addSeenNode(wrapNode(n))
	}

	p.putCacheId(id, addr)
	p.Log.Trace("receive utp data", "addr", addr, "msg-length", len(msg))
	p.packetRouter.ReceiveMessage(msg, addr)
	return []byte("")
}

func (p *PortalProtocol) handleTalkRequest(id enode.ID, addr *net.UDPAddr, msg []byte) []byte {
	if n := p.DiscV5.getNode(id); n != nil {
		p.table.addSeenNode(wrapNode(n))
	}
	p.putCacheId(id, addr)

	msgCode := msg[0]

	switch msgCode {
	case portalwire.PING:
		pingRequest := &portalwire.Ping{}
		err := pingRequest.UnmarshalSSZ(msg[1:])
		if err != nil {
			p.Log.Error("failed to unmarshal ping request", "err", err)
			return nil
		}

		p.Log.Trace("received ping request", "protocol", p.protocolName, "source", id, "pingRequest", pingRequest)
		resp, err := p.handlePing(id, pingRequest)
		if err != nil {
			p.Log.Error("failed to handle ping request", "err", err)
			return nil
		}

		return resp
	case portalwire.FINDNODES:
		findNodesRequest := &portalwire.FindNodes{}
		err := findNodesRequest.UnmarshalSSZ(msg[1:])
		if err != nil {
			p.Log.Error("failed to unmarshal find nodes request", "err", err)
			return nil
		}

		p.Log.Trace("received find nodes request", "protocol", p.protocolName, "source", id, "findNodesRequest", findNodesRequest)
		resp, err := p.handleFindNodes(addr, findNodesRequest)
		if err != nil {
			p.Log.Error("failed to handle find nodes request", "err", err)
			return nil
		}

		return resp
	case portalwire.FINDCONTENT:
		findContentRequest := &portalwire.FindContent{}
		err := findContentRequest.UnmarshalSSZ(msg[1:])
		if err != nil {
			p.Log.Error("failed to unmarshal find content request", "err", err)
			return nil
		}

		p.Log.Trace("received find content request", "protocol", p.protocolName, "source", id, "findContentRequest", findContentRequest)
		resp, err := p.handleFindContent(id, addr, findContentRequest)
		if err != nil {
			p.Log.Error("failed to handle find content request", "err", err)
			return nil
		}

		return resp
	case portalwire.OFFER:
		offerRequest := &portalwire.Offer{}
		err := offerRequest.UnmarshalSSZ(msg[1:])
		if err != nil {
			p.Log.Error("failed to unmarshal offer request", "err", err)
			return nil
		}

		p.Log.Trace("received offer request", "protocol", p.protocolName, "source", id, "offerRequest", offerRequest)
		resp, err := p.handleOffer(id, addr, offerRequest)
		if err != nil {
			p.Log.Error("failed to handle offer request", "err", err)
			return nil
		}

		return resp
	}

	return nil
}

func (p *PortalProtocol) handlePing(id enode.ID, ping *portalwire.Ping) ([]byte, error) {
	pingCustomPayload := &portalwire.PingPongCustomData{}
	err := pingCustomPayload.UnmarshalSSZ(ping.CustomPayload)
	if err != nil {
		return nil, err
	}

	p.radiusCache.Set([]byte(id.String()), pingCustomPayload.Radius)

	enrSeq := p.Self().Seq()
	radiusBytes, err := p.nodeRadius.MarshalSSZ()
	if err != nil {
		return nil, err
	}
	pongCustomPayload := &portalwire.PingPongCustomData{
		Radius: radiusBytes,
	}

	pongCustomPayloadBytes, err := pongCustomPayload.MarshalSSZ()
	if err != nil {
		return nil, err
	}

	pong := &portalwire.Pong{
		EnrSeq:        enrSeq,
		CustomPayload: pongCustomPayloadBytes,
	}

	p.Log.Trace("Sending pong response", "protocol", p.protocolName, "source", id, "pong", pong)
	pongBytes, err := pong.MarshalSSZ()

	if err != nil {
		return nil, err
	}

	talkRespBytes := make([]byte, 0, len(pongBytes)+1)
	talkRespBytes = append(talkRespBytes, portalwire.PONG)
	talkRespBytes = append(talkRespBytes, pongBytes...)

	return talkRespBytes, nil
}

func (p *PortalProtocol) handleFindNodes(fromAddr *net.UDPAddr, request *portalwire.FindNodes) ([]byte, error) {
	distances := make([]uint, len(request.Distances))
	for i, distance := range request.Distances {
		distances[i] = uint(ssz.UnmarshallUint16(distance[:]))
	}

	nodes := p.collectTableNodes(fromAddr.IP, distances, portalFindnodesResultLimit)

	nodesOverhead := 1 + 1 + 4 // msg id + total + container offset
	maxPayloadSize := maxPacketSize - talkRespOverhead - nodesOverhead
	enrOverhead := 4 //per added ENR, 4 bytes offset overhead

	enrs := p.truncateNodes(nodes, maxPayloadSize, enrOverhead)

	nodesMsg := &portalwire.Nodes{
		Total: 1,
		Enrs:  enrs,
	}

	p.Log.Trace("Sending nodes response", "protocol", p.protocolName, "source", fromAddr, "nodes", nodesMsg)
	nodesMsgBytes, err := nodesMsg.MarshalSSZ()
	if err != nil {
		return nil, err
	}

	talkRespBytes := make([]byte, 0, len(nodesMsgBytes)+1)
	talkRespBytes = append(talkRespBytes, portalwire.NODES)
	talkRespBytes = append(talkRespBytes, nodesMsgBytes...)

	return talkRespBytes, nil
}

func (p *PortalProtocol) handleFindContent(id enode.ID, addr *net.UDPAddr, request *portalwire.FindContent) ([]byte, error) {
	contentOverhead := 1 + 1 // msg id + SSZ Union selector
	maxPayloadSize := maxPacketSize - talkRespOverhead - contentOverhead
	enrOverhead := 4 //per added ENR, 4 bytes offset overhead
	var err error
	contentKey := request.ContentKey
	contentId := p.toContentId(contentKey)
	if contentId == nil {
		return nil, ErrNilContentKey
	}

	var content []byte
	content, err = p.storage.Get(contentKey, contentId)
	if err != nil && !errors.Is(err, ContentNotFound) {
		return nil, err
	}

	p.putCacheId(id, addr)

	if errors.Is(err, ContentNotFound) {
		closestNodes := p.findNodesCloseToContent(contentId, portalFindnodesResultLimit)
		for i, n := range closestNodes {
			if n.ID() == id {
				closestNodes = append(closestNodes[:i], closestNodes[i+1:]...)
				break
			}
		}

		enrs := p.truncateNodes(closestNodes, maxPayloadSize, enrOverhead)
		// TODO fix when no content and no enrs found
		if len(enrs) == 0 {
			enrs = nil
		}

		enrsMsg := &portalwire.Enrs{
			Enrs: enrs,
		}

		p.Log.Trace("Sending enrs content response", "protocol", p.protocolName, "source", addr, "enrs", enrsMsg)
		var enrsMsgBytes []byte
		enrsMsgBytes, err = enrsMsg.MarshalSSZ()
		if err != nil {
			return nil, err
		}

		contentMsgBytes := make([]byte, 0, len(enrsMsgBytes)+1)
		contentMsgBytes = append(contentMsgBytes, portalwire.ContentEnrsSelector)
		contentMsgBytes = append(contentMsgBytes, enrsMsgBytes...)

		talkRespBytes := make([]byte, 0, len(contentMsgBytes)+1)
		talkRespBytes = append(talkRespBytes, portalwire.CONTENT)
		talkRespBytes = append(talkRespBytes, contentMsgBytes...)

		return talkRespBytes, nil
	} else if len(content) <= maxPayloadSize {
		rawContentMsg := &portalwire.Content{
			Content: content,
		}

		p.Log.Trace("Sending raw content response", "protocol", p.protocolName, "source", addr, "content", rawContentMsg)

		var rawContentMsgBytes []byte
		rawContentMsgBytes, err = rawContentMsg.MarshalSSZ()
		if err != nil {
			return nil, err
		}

		contentMsgBytes := make([]byte, 0, len(rawContentMsgBytes)+1)
		contentMsgBytes = append(contentMsgBytes, portalwire.ContentRawSelector)
		contentMsgBytes = append(contentMsgBytes, rawContentMsgBytes...)

		talkRespBytes := make([]byte, 0, len(contentMsgBytes)+1)
		talkRespBytes = append(talkRespBytes, portalwire.CONTENT)
		talkRespBytes = append(talkRespBytes, contentMsgBytes...)

		return talkRespBytes, nil
	} else {
		connId := p.connIdGen.GenCid(id, false)
		connIdSend := connId.SendId()

		go func(bctx context.Context) {
			for {
				select {
				case <-bctx.Done():
					return
				default:
					ctx, cancel := context.WithTimeout(bctx, defaultUTPConnectTimeout)
					var conn *utp.Conn
					p.Log.Debug("will accept find content conn from: ", "source", addr, "connId", connId)
					conn, err = p.utp.AcceptUTPContext(ctx, connIdSend)
					if err != nil {
						p.Log.Error("failed to accept utp connection for handle find content", "connId", connIdSend, "err", err)
						cancel()
						return
					}
					cancel()

					err = conn.SetWriteDeadline(time.Now().Add(defaultUTPWriteTimeout))
					if err != nil {
						p.Log.Error("failed to set write deadline", "err", err)
						err = conn.Close()
						if err != nil {
							p.Log.Error("failed to close utp connection", "err", err)
							return
						}
						return
					}

					var n int
					n, err = conn.Write(content)
					if err != nil {
						p.Log.Error("failed to write content to utp connection", "err", err)
						err = conn.Close()
						if err != nil {
							p.Log.Error("failed to close utp connection", "err", err)
							return
						}
						return
					}

					err = conn.Close()
					if err != nil {
						p.Log.Error("failed to close utp connection", "err", err)
						return
					}

					p.Log.Trace("wrote content size to utp connection", "n", n)
					return
				}
			}
		}(p.closeCtx)

		idBuffer := make([]byte, 2)
		binary.BigEndian.PutUint16(idBuffer, uint16(connIdSend))
		connIdMsg := &portalwire.ConnectionId{
			Id: idBuffer,
		}

		p.Log.Trace("Sending connection id content response", "protocol", p.protocolName, "source", addr, "connId", connIdMsg)
		var connIdMsgBytes []byte
		connIdMsgBytes, err = connIdMsg.MarshalSSZ()
		if err != nil {
			return nil, err
		}

		contentMsgBytes := make([]byte, 0, len(connIdMsgBytes)+1)
		contentMsgBytes = append(contentMsgBytes, portalwire.ContentConnIdSelector)
		contentMsgBytes = append(contentMsgBytes, connIdMsgBytes...)

		talkRespBytes := make([]byte, 0, len(contentMsgBytes)+1)
		talkRespBytes = append(talkRespBytes, portalwire.CONTENT)
		talkRespBytes = append(talkRespBytes, contentMsgBytes...)

		return talkRespBytes, nil
	}
}

func (p *PortalProtocol) handleOffer(id enode.ID, addr *net.UDPAddr, request *portalwire.Offer) ([]byte, error) {
	var err error
	contentKeyBitlist := bitfield.NewBitlist(uint64(len(request.ContentKeys)))
	if len(p.contentQueue) >= cap(p.contentQueue) {
		acceptMsg := &portalwire.Accept{
			ConnectionId: []byte{0, 0},
			ContentKeys:  contentKeyBitlist,
		}

		p.Log.Trace("Sending accept response", "protocol", p.protocolName, "source", addr, "accept", acceptMsg)
		var acceptMsgBytes []byte
		acceptMsgBytes, err = acceptMsg.MarshalSSZ()
		if err != nil {
			return nil, err
		}

		talkRespBytes := make([]byte, 0, len(acceptMsgBytes)+1)
		talkRespBytes = append(talkRespBytes, portalwire.ACCEPT)
		talkRespBytes = append(talkRespBytes, acceptMsgBytes...)

		return talkRespBytes, nil
	}

	contentKeys := make([][]byte, 0)
	for i, contentKey := range request.ContentKeys {
		contentId := p.toContentId(contentKey)
		if contentId != nil {
			if inRange(p.Self().ID(), p.nodeRadius, contentId) {
				if _, err = p.storage.Get(contentKey, contentId); err != nil {
					contentKeyBitlist.SetBitAt(uint64(i), true)
					contentKeys = append(contentKeys, contentKey)
				}
			}
		} else {
			return nil, ErrNilContentKey
		}
	}

	p.putCacheId(id, addr)

	idBuffer := make([]byte, 2)
	if contentKeyBitlist.Count() != 0 {
		connId := p.connIdGen.GenCid(id, false)
		connIdSend := connId.SendId()

		go func(bctx context.Context) {
			for {
				select {
				case <-bctx.Done():
					return
				default:
					ctx, cancel := context.WithTimeout(bctx, defaultUTPConnectTimeout)
					var conn *utp.Conn
					p.Log.Debug("will accept offer conn from: ", "source", addr, "connId", connId)
					conn, err = p.utp.AcceptUTPContext(ctx, connIdSend)
					if err != nil {
						p.Log.Error("failed to accept utp connection for handle offer", "connId", connIdSend, "err", err)
						cancel()
						return
					}
					cancel()

					err = conn.SetReadDeadline(time.Now().Add(defaultUTPReadTimeout))
					if err != nil {
						p.Log.Error("failed to set read deadline", "err", err)
						return
					}
					// Read ALL the data from the connection until EOF and return it
					var data []byte
					data, err = io.ReadAll(conn)
					if err != nil {
						p.Log.Error("failed to read from utp connection", "err", err)
						return
					}
					p.Log.Trace("Received offer content response", "id", id, "size", len(data), "data", data)

					err = p.handleOfferedContents(id, contentKeys, data)
					if err != nil {
						p.Log.Error("failed to handle offered Contents", "err", err)
						return
					}

					return
				}
			}
		}(p.closeCtx)

		binary.BigEndian.PutUint16(idBuffer, uint16(connIdSend))
	} else {
		binary.BigEndian.PutUint16(idBuffer, uint16(0))
	}

	acceptMsg := &portalwire.Accept{
		ConnectionId: idBuffer,
		ContentKeys:  []byte(contentKeyBitlist),
	}

	p.Log.Trace("Sending accept response", "protocol", p.protocolName, "source", addr, "accept", acceptMsg)
	var acceptMsgBytes []byte
	acceptMsgBytes, err = acceptMsg.MarshalSSZ()
	if err != nil {
		return nil, err
	}

	talkRespBytes := make([]byte, 0, len(acceptMsgBytes)+1)
	talkRespBytes = append(talkRespBytes, portalwire.ACCEPT)
	talkRespBytes = append(talkRespBytes, acceptMsgBytes...)

	return talkRespBytes, nil
}

func (p *PortalProtocol) handleOfferedContents(id enode.ID, keys [][]byte, payload []byte) error {
	contents, err := decodeContents(payload)
	if err != nil {
		return err
	}

	keyLen := len(keys)
	contentLen := len(contents)
	if keyLen != contentLen {
		return fmt.Errorf("content keys len %d doesn't match content values len %d", keyLen, contentLen)
	}

	contentElement := &ContentElement{
		Node:        id,
		ContentKeys: keys,
		Contents:    contents,
	}

	p.contentQueue <- contentElement

	return nil
}

func (p *PortalProtocol) Self() *enode.Node {
	return p.localNode.Node()
}

func (p *PortalProtocol) RequestENR(n *enode.Node) (*enode.Node, error) {
	nodes, err := p.findNodes(n, []uint{0})
	if err != nil {
		return nil, err
	}
	if len(nodes) != 1 {
		return nil, fmt.Errorf("%d nodes in response for distance zero", len(nodes))
	}
	return nodes[0], nil
}

func (p *PortalProtocol) verifyResponseNode(sender *enode.Node, r *enr.Record, distances []uint, seen map[enode.ID]struct{}) (*enode.Node, error) {
	n, err := enode.New(p.validSchemes, r)
	if err != nil {
		return nil, err
	}
	if err = netutil.CheckRelayIP(sender.IP(), n.IP()); err != nil {
		return nil, err
	}
	if p.NetRestrict != nil && !p.NetRestrict.Contains(n.IP()) {
		return nil, errors.New("not contained in netrestrict list")
	}
	if n.UDP() <= 1024 {
		return nil, errLowPort
	}
	if distances != nil {
		nd := enode.LogDist(sender.ID(), n.ID())
		if !slices.Contains(distances, uint(nd)) {
			return nil, errors.New("does not match any requested distance")
		}
	}
	if _, ok := seen[n.ID()]; ok {
		return nil, fmt.Errorf("duplicate record")
	}
	seen[n.ID()] = struct{}{}
	return n, nil
}

func (p *PortalProtocol) replaceNode(node *enode.Node) {
	p.table.mutex.Lock()
	defer p.table.mutex.Unlock()
	b := p.table.bucket(node.ID())
	p.table.replace(b, wrapNode(node))
}

// lookupRandom looks up a random target.
// This is needed to satisfy the transport interface.
func (p *PortalProtocol) lookupRandom() []*enode.Node {
	return p.newRandomLookup(p.closeCtx).run()
}

// lookupSelf looks up our own node ID.
// This is needed to satisfy the transport interface.
func (p *PortalProtocol) lookupSelf() []*enode.Node {
	return p.newLookup(p.closeCtx, p.Self().ID()).run()
}

func (p *PortalProtocol) newRandomLookup(ctx context.Context) *lookup {
	var target enode.ID
	_, _ = crand.Read(target[:])
	return p.newLookup(ctx, target)
}

func (p *PortalProtocol) newLookup(ctx context.Context, target enode.ID) *lookup {
	return newLookup(ctx, p.table, target, func(n *node) ([]*node, error) {
		return p.lookupWorker(n, target)
	})
}

// lookupWorker performs FINDNODE calls against a single node during lookup.
func (p *PortalProtocol) lookupWorker(destNode *node, target enode.ID) ([]*node, error) {
	var (
		dists = lookupDistances(target, destNode.ID())
		nodes = nodesByDistance{target: target}
		err   error
	)
	var r []*enode.Node

	r, err = p.findNodes(unwrapNode(destNode), dists)
	if errors.Is(err, errClosed) {
		return nil, err
	}
	for _, n := range r {
		if n.ID() != p.Self().ID() {
			wn := wrapNode(n)
			p.table.addSeenNode(wn)
			nodes.push(wn, portalFindnodesResultLimit)
		}
	}
	return nodes.entries, err
}

func (p *PortalProtocol) offerWorker() {
	for {
		select {
		case <-p.closeCtx.Done():
			return
		case offerRequestWithNode := <-p.offerQueue:
			p.Log.Trace("offerWorker", "offerRequestWithNode", offerRequestWithNode)
			_, err := p.offer(offerRequestWithNode.Node, offerRequestWithNode.Request)
			if err != nil {
				p.Log.Error("failed to offer", "err", err)
			}
		}
	}
}

func (p *PortalProtocol) truncateNodes(nodes []*enode.Node, maxSize int, enrOverhead int) [][]byte {
	res := make([][]byte, 0)
	totalSize := 0
	for _, n := range nodes {
		enrBytes, err := rlp.EncodeToBytes(n.Record())
		if err != nil {
			p.Log.Error("failed to encode n", "err", err)
			continue
		}

		if totalSize+len(enrBytes)+enrOverhead > maxSize {
			break
		} else {
			res = append(res, enrBytes)
			totalSize += len(enrBytes)
		}
	}
	return res
}

func (p *PortalProtocol) findNodesCloseToContent(contentId []byte, limit int) []*enode.Node {
	allNodes := p.table.Nodes()
	sort.Slice(allNodes, func(i, j int) bool {
		return enode.LogDist(allNodes[i].ID(), enode.ID(contentId)) < enode.LogDist(allNodes[j].ID(), enode.ID(contentId))
	})

	if len(allNodes) > limit {
		allNodes = allNodes[:limit]
	} else {
		allNodes = allNodes[:]
	}

	return allNodes
}

// Lookup performs a recursive lookup for the given target.
// It returns the closest nodes to target.
func (p *PortalProtocol) Lookup(target enode.ID) []*enode.Node {
	return p.newLookup(p.closeCtx, target).run()
}

// Resolve searches for a specific Node with the given ID and tries to get the most recent
// version of the Node record for it. It returns n if the Node could not be resolved.
func (p *PortalProtocol) Resolve(n *enode.Node) *enode.Node {
	if intable := p.table.getNode(n.ID()); intable != nil && intable.Seq() > n.Seq() {
		n = intable
	}
	// Try asking directly. This works if the Node is still responding on the endpoint we have.
	if resp, err := p.RequestENR(n); err == nil {
		return resp
	}
	// Otherwise do a network lookup.
	result := p.Lookup(n.ID())
	for _, rn := range result {
		if rn.ID() == n.ID() && rn.Seq() > n.Seq() {
			return rn
		}
	}
	return n
}

// ResolveNodeId searches for a specific Node with the given ID.
// It returns nil if the nodeId could not be resolved.
func (p *PortalProtocol) ResolveNodeId(id enode.ID) *enode.Node {
	if id == p.Self().ID() {
		return p.Self()
	}

	n := p.table.getNode(id)
	if n != nil {
		// Try asking directly. This works if the Node is still responding on the endpoint we have.
		if resp, err := p.RequestENR(n); err == nil {
			return resp
		}
	}

	// Otherwise do a network lookup.
	result := p.Lookup(id)
	for _, rn := range result {
		if rn.ID() == id {
			if n != nil && rn.Seq() <= n.Seq() {
				return n
			} else {
				return rn
			}
		}
	}

	return n
}

func (p *PortalProtocol) collectTableNodes(rip net.IP, distances []uint, limit int) []*enode.Node {
	var bn []*enode.Node
	var nodes []*enode.Node
	var processed = make(map[uint]struct{})
	for _, dist := range distances {
		// Reject duplicate / invalid distances.
		_, seen := processed[dist]
		if seen || dist > 256 {
			continue
		}
		processed[dist] = struct{}{}

		for _, n := range p.table.appendLiveNodes(dist, bn[:0]) {
			// Apply some pre-checks to avoid sending invalid nodes.
			// Note liveness is checked by appendLiveNodes.
			if netutil.CheckRelayIP(rip, n.IP()) != nil {
				continue
			}
			nodes = append(nodes, n)
			if len(nodes) >= limit {
				return nodes
			}
		}
	}
	return nodes
}

func (p *PortalProtocol) ContentLookup(contentKey, contentId []byte) ([]byte, bool, error) {
	lookupContext, cancel := context.WithCancel(context.Background())
	defer cancel()
	resChan := make(chan *ContentInfoResp, 1)
	defer close(resChan)
	newLookup(lookupContext, p.table, enode.ID(contentId), func(n *node) ([]*node, error) {
		return p.contentLookupWorker(unwrapNode(n), contentKey, resChan)
	}).run()

	if len(resChan) > 0 {
		res := <-resChan
		return res.Content, res.UtpTransfer, nil
	}
	return nil, false, ContentNotFound
}

func (p *PortalProtocol) contentLookupWorker(n *enode.Node, contentKey []byte, resChan chan<- *ContentInfoResp) ([]*node, error) {
	wrapedNode := make([]*node, 0)
	flag, content, err := p.findContent(n, contentKey)
	if err != nil {
		p.Log.Error("contentLookupWorker failed", "ip", n.IP().String(), "err", err)
		return nil, err
	}
	p.Log.Debug("contentLookupWorker reveice response", "ip", n.IP().String(), "flag", flag)
	// has find content
	if len(resChan) > 0 {
		return []*node{}, nil
	}
	switch flag {
	case portalwire.ContentRawSelector, portalwire.ContentConnIdSelector:
		content, ok := content.([]byte)
		if !ok {
			return wrapedNode, fmt.Errorf("failed to assert to raw content, value is: %v", content)
		}
		res := &ContentInfoResp{
			Content: content,
		}
		if flag == portalwire.ContentConnIdSelector {
			res.UtpTransfer = true
		}
		resChan <- res
		return wrapedNode, err
	case portalwire.ContentEnrsSelector:
		nodes, ok := content.([]*enode.Node)
		if !ok {
			return wrapedNode, fmt.Errorf("failed to assert to enrs content, value is: %v", content)
		}
		return wrapNodes(nodes), nil
	}
	return wrapedNode, nil
}

func (p *PortalProtocol) TraceContentLookup(contentKey, contentId []byte) (*TraceContentResult, error) {
	lookupContext, cancel := context.WithCancel(context.Background())
	defer cancel()
	requestNodeChan := make(chan *enode.Node, 3)
	resChan := make(chan *traceContentInfoResp, 3)

	requestNode := make([]*enode.Node, 0)
	requestRes := make(map[string]*traceContentInfoResp)

	traceContentRes := &TraceContentResult{}

	selfHexId := "0x" + p.Self().ID().String()

	trace := &Trace{
		Origin:      selfHexId,
		TargetId:    hexutil.Encode(contentId),
		StartedAtMs: int(time.Now().UnixMilli()),
		Responses:   make(map[string][]string),
		Metadata:    make(map[string]*NodeMetadata),
		Cancelled:   make([]string, 0),
	}

	nodes := p.table.findnodeByID(enode.ID(contentId), bucketSize, false)

	localResponse := make([]string, 0, len(nodes.entries))
	for _, node := range nodes.entries {
		id := "0x" + node.ID().String()
		localResponse = append(localResponse, id)
	}
	trace.Responses[selfHexId] = localResponse

	dis := p.Distance(p.Self().ID(), enode.ID(contentId))

	trace.Metadata[selfHexId] = &NodeMetadata{
		Enr:      p.Self().String(),
		Distance: hexutil.Encode(dis[:]),
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for node := range requestNodeChan {
			requestNode = append(requestNode, node)
		}
	}()

	go func() {
		defer wg.Done()
		for res := range resChan {
			key := res.Node.ID().String()
			requestRes[key] = res
			if res.Flag == portalwire.ContentRawSelector || res.Flag == portalwire.ContentConnIdSelector {
				// get the content
				return
			}
		}
	}()

	newLookup(lookupContext, p.table, enode.ID(contentId), func(n *node) ([]*node, error) {
		node := unwrapNode(n)
		requestNodeChan <- node
		return p.traceContentLookupWorker(node, contentKey, resChan)
	}).run()

	close(requestNodeChan)
	close(resChan)

	wg.Wait()

	for _, node := range requestNode {
		id := node.ID().String()
		hexId := "0x" + id
		dis := p.Distance(node.ID(), enode.ID(contentId))
		trace.Metadata[hexId] = &NodeMetadata{
			Enr:      node.String(),
			Distance: hexutil.Encode(dis[:]),
		}
		if res, ok := requestRes[id]; ok {
			if res.Flag == portalwire.ContentRawSelector || res.Flag == portalwire.ContentConnIdSelector {
				trace.ReceivedFrom = hexId
				content := res.Content.([]byte)
				traceContentRes.Content = hexutil.Encode(content)
				traceContentRes.UtpTransfer = res.UtpTransfer
				trace.Responses[hexId] = make([]string, 0)
			} else {
				content := res.Content.([]*enode.Node)
				ids := make([]string, 0)
				for _, n := range content {
					hexId := "0x" + n.ID().String()
					ids = append(ids, hexId)
				}
				trace.Responses[hexId] = ids
			}
		} else {
			trace.Cancelled = append(trace.Cancelled, id)
		}
	}

	traceContentRes.Trace = *trace

	return traceContentRes, nil
}

func (p *PortalProtocol) traceContentLookupWorker(n *enode.Node, contentKey []byte, resChan chan<- *traceContentInfoResp) ([]*node, error) {
	wrapedNode := make([]*node, 0)
	flag, content, err := p.findContent(n, contentKey)
	if err != nil {
		return nil, err
	}
	p.Log.Debug("traceContentLookupWorker reveice response", "ip", n.IP().String(), "flag", flag)
	// has find content
	if len(resChan) > 0 {
		return []*node{}, nil
	}

	switch flag {
	case portalwire.ContentRawSelector, portalwire.ContentConnIdSelector:
		content, ok := content.([]byte)
		if !ok {
			return wrapedNode, fmt.Errorf("failed to assert to raw content, value is: %v", content)
		}
		res := &traceContentInfoResp{
			Node:        n,
			Flag:        flag,
			Content:     content,
			UtpTransfer: false,
		}
		if flag == portalwire.ContentConnIdSelector {
			res.UtpTransfer = true
		}
		resChan <- res
		return wrapedNode, err
	case portalwire.ContentEnrsSelector:
		nodes, ok := content.([]*enode.Node)
		if !ok {
			return wrapedNode, fmt.Errorf("failed to assert to enrs content, value is: %v", content)
		}
		resChan <- &traceContentInfoResp{Node: n,
			Flag:        flag,
			Content:     content,
			UtpTransfer: false}
		return wrapNodes(nodes), nil
	}
	return wrapedNode, nil
}

func (p *PortalProtocol) ToContentId(contentKey []byte) []byte {
	return p.toContentId(contentKey)
}

func (p *PortalProtocol) InRange(contentId []byte) bool {
	return inRange(p.Self().ID(), p.nodeRadius, contentId)
}

func (p *PortalProtocol) Get(contentKey []byte, contentId []byte) ([]byte, error) {
	content, err := p.storage.Get(contentKey, contentId)
	p.Log.Trace("get local storage", "contentId", hexutil.Encode(contentId), "content", hexutil.Encode(content), "err", err)
	return content, err
}

func (p *PortalProtocol) Put(contentKey []byte, contentId []byte, content []byte) error {
	err := p.storage.Put(contentKey, contentId, content)
	p.Log.Trace("put local storage", "contentId", hexutil.Encode(contentId), "content", hexutil.Encode(content), "err", err)
	return err
}

func (p *PortalProtocol) GetContent() chan *ContentElement {
	return p.contentQueue
}

func (p *PortalProtocol) Gossip(srcNodeId *enode.ID, contentKeys [][]byte, content [][]byte) (int, error) {
	if len(content) == 0 {
		return 0, errors.New("empty content")
	}

	contentList := make([]*ContentEntry, 0, portalwire.ContentKeysLimit)
	for i := 0; i < len(content); i++ {
		contentEntry := &ContentEntry{
			ContentKey: contentKeys[i],
			Content:    content[i],
		}
		contentList = append(contentList, contentEntry)
	}

	contentId := p.toContentId(contentKeys[0])
	if contentId == nil {
		return 0, ErrNilContentKey
	}

	maxClosestNodes := 4
	maxFartherNodes := 4
	closestLocalNodes := p.findNodesCloseToContent(contentId, 32)

	gossipNodes := make([]*enode.Node, 0)
	for _, n := range closestLocalNodes {
		radius, found := p.radiusCache.HasGet(nil, []byte(n.ID().String()))
		if found {
			nodeRadius := new(uint256.Int)
			err := nodeRadius.UnmarshalSSZ(radius)
			if err != nil {
				return 0, err
			}
			if inRange(n.ID(), nodeRadius, contentId) {
				if srcNodeId == nil {
					gossipNodes = append(gossipNodes, n)
				} else if n.ID() != *srcNodeId {
					gossipNodes = append(gossipNodes, n)
				}
			}
		}
	}

	if len(gossipNodes) == 0 {
		return 0, nil
	}

	var finalGossipNodes []*enode.Node
	if len(gossipNodes) > maxClosestNodes {
		fartherNodes := gossipNodes[maxClosestNodes:]
		rand.Shuffle(len(fartherNodes), func(i, j int) {
			fartherNodes[i], fartherNodes[j] = fartherNodes[j], fartherNodes[i]
		})
		finalGossipNodes = append(gossipNodes[:maxClosestNodes], fartherNodes[:min(maxFartherNodes, len(fartherNodes))]...)
	} else {
		finalGossipNodes = gossipNodes
	}

	for _, n := range finalGossipNodes {
		transientOfferRequest := &TransientOfferRequest{
			Contents: contentList,
		}

		offerRequest := &OfferRequest{
			Kind:    TransientOfferRequestKind,
			Request: transientOfferRequest,
		}

		offerRequestWithNode := &OfferRequestWithNode{
			Node:    n,
			Request: offerRequest,
		}
		p.offerQueue <- offerRequestWithNode
	}

	return len(finalGossipNodes), nil
}

func (p *PortalProtocol) Distance(a, b enode.ID) enode.ID {
	res := [32]byte{}
	for i := range a {
		res[i] = a[i] ^ b[i]
	}
	return res
}

func inRange(nodeId enode.ID, nodeRadius *uint256.Int, contentId []byte) bool {
	distance := enode.LogDist(nodeId, enode.ID(contentId))
	disBig := new(big.Int).SetInt64(int64(distance))
	return nodeRadius.CmpBig(disBig) > 0
}

func encodeContents(contents [][]byte) ([]byte, error) {
	contentsBytes := make([]byte, 0)
	for _, content := range contents {
		contentLen := len(content)
		contentLenBytes := leb128.EncodeUint32(uint32(contentLen))
		contentsBytes = append(contentsBytes, contentLenBytes...)
		contentsBytes = append(contentsBytes, content...)
	}

	return contentsBytes, nil
}

func decodeContents(payload []byte) ([][]byte, error) {
	contents := make([][]byte, 0)
	buffer := bytes.NewBuffer(payload)

	for {
		contentLen, contentLenLen, err := leb128.DecodeUint32(bytes.NewReader(buffer.Bytes()))
		if err != nil {
			if errors.Is(err, io.EOF) {
				return contents, nil
			}
			return nil, err
		}

		buffer.Next(int(contentLenLen))

		content := make([]byte, contentLen)
		_, err = buffer.Read(content)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return contents, nil
			}
			return nil, err
		}

		contents = append(contents, content)
	}
}

func getContentKeys(request *OfferRequest) [][]byte {
	if request.Kind == TransientOfferRequestKind {
		contentKeys := make([][]byte, 0)
		contents := request.Request.(*TransientOfferRequest).Contents
		for _, content := range contents {
			contentKeys = append(contentKeys, content.ContentKey)
		}

		return contentKeys
	} else {
		return request.Request.(*PersistOfferRequest).ContentKeys
	}
}

func RandomNodes(table *Table, maxCount int, pred func(node *enode.Node) bool) []*enode.Node {
	maxAmount := maxCount
	tableCount := table.len()
	if maxAmount > tableCount {
		maxAmount = tableCount
	}

	var result []*enode.Node
	var seen = make(map[enode.ID]struct{})

	for {
		if len(result) >= maxAmount {
			return result
		}

		b := table.buckets[rand.Intn(len(table.buckets))]
		if len(b.entries) != 0 {
			n := b.entries[rand.Intn(len(b.entries))]
			if _, ok := seen[n.ID()]; !ok {
				seen[n.ID()] = struct{}{}
				if pred == nil || pred(&n.Node) {
					result = append(result, &n.Node)
				}
			}
		}
	}
}
