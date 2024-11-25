package portalwire

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
	"sync/atomic"
	"time"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/discover/v5wire"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/ethereum/go-ethereum/p2p/netutil"
	"github.com/ethereum/go-ethereum/portalnetwork/storage"
	"github.com/ethereum/go-ethereum/rlp"
	ssz "github.com/ferranbt/fastssz"
	"github.com/holiman/uint256"
	"github.com/optimism-java/utp-go"
	"github.com/optimism-java/utp-go/libutp"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/tetratelabs/wabin/leb128"
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
	ListenAddr      string
	NetRestrict     *netutil.Netlist
	NodeRadius      *uint256.Int
	RadiusCacheSize int
	NodeDBPath      string
	NAT             nat.Interface
	clock           mclock.Clock
}

func DefaultPortalProtocolConfig() *PortalProtocolConfig {
	return &PortalProtocolConfig{
		BootstrapNodes:  make([]*enode.Node, 0),
		ListenAddr:      ":9009",
		NetRestrict:     nil,
		RadiusCacheSize: 32 * 1024 * 1024,
		NodeDBPath:      "",
		clock:           mclock.System{},
	}
}

type PortalProtocol struct {
	table *discover.Table

	protocolId   string
	protocolName string

	DiscV5         *discover.UDPv5
	localNode      *enode.LocalNode
	Log            log.Logger
	PrivateKey     *ecdsa.PrivateKey
	NetRestrict    *netutil.Netlist
	BootstrapNodes []*enode.Node
	conn           discover.UDPConn

	Utp       *PortalUtp
	connIdGen libutp.ConnIdGenerator

	validSchemes   enr.IdentityScheme
	radiusCache    *fastcache.Cache
	closeCtx       context.Context
	cancelCloseCtx context.CancelFunc
	storage        storage.ContentStorage
	toContentId    func(contentKey []byte) []byte

	contentQueue chan *ContentElement
	offerQueue   chan *OfferRequestWithNode

	portMappingRegister chan *portMapping
	clock               mclock.Clock
	NAT                 nat.Interface

	portalMetrics *portalMetrics
}

func defaultContentIdFunc(contentKey []byte) []byte {
	digest := sha256.Sum256(contentKey)
	return digest[:]
}

func NewPortalProtocol(config *PortalProtocolConfig, protocolId ProtocolId, privateKey *ecdsa.PrivateKey, conn discover.UDPConn, localNode *enode.LocalNode, discV5 *discover.UDPv5, utp *PortalUtp, storage storage.ContentStorage, contentQueue chan *ContentElement, opts ...PortalProtocolOption) (*PortalProtocol, error) {
	closeCtx, cancelCloseCtx := context.WithCancel(context.Background())

	protocol := &PortalProtocol{
		protocolId:     string(protocolId),
		protocolName:   protocolId.Name(),
		Log:            log.New("protocol", protocolId.Name()),
		PrivateKey:     privateKey,
		NetRestrict:    config.NetRestrict,
		BootstrapNodes: config.BootstrapNodes,
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
		Utp:            utp,
		NAT:            config.NAT,
		clock:          config.clock,
		connIdGen:      libutp.NewConnIdGenerator(),
	}

	for _, opt := range opts {
		opt(protocol)
	}

	if metrics.Enabled {
		protocol.portalMetrics = newPortalMetrics(protocolId.Name())
	}

	return protocol, nil
}

func (p *PortalProtocol) Start() error {
	p.setupPortMapping()

	err := p.setupDiscV5AndTable()
	if err != nil {
		return err
	}

	p.DiscV5.RegisterTalkHandler(p.protocolId, p.handleTalkRequest)
	if p.Utp != nil {
		err = p.Utp.Start()
	}
	if err != nil {
		return err
	}

	go p.table.Loop()

	for i := 0; i < concurrentOffers; i++ {
		go p.offerWorker()
	}

	// wait for both initialization processes to complete
	p.DiscV5.Table().WaitInit()
	p.table.WaitInit()
	return nil
}

func (p *PortalProtocol) Stop() {
	p.cancelCloseCtx()
	p.table.Close()
	p.DiscV5.Close()
	if p.Utp != nil {
		p.Utp.Stop()
	}
}
func (p *PortalProtocol) RoutingTableInfo() [][]string {
	return p.table.NodeIds()
}

func (p *PortalProtocol) AddEnr(n *enode.Node) {
	added := p.table.AddInboundNode(n)
	if !added {
		p.Log.Warn("add node failed", "id", n.ID(), "ip", n.IPAddr())
		return
	}
	id := n.ID().String()
	p.radiusCache.Set([]byte(id), MaxDistance)
}

func (p *PortalProtocol) Radius() *uint256.Int {
	return p.storage.Radius()
}

func (p *PortalProtocol) setupUDPListening() error {
	laddr := p.conn.LocalAddr().(*net.UDPAddr)
	p.localNode.SetFallbackUDP(laddr.Port)
	p.Log.Debug("UDP listener up", "addr", laddr)
	// TODO: NAT
	if !laddr.IP.IsLoopback() && !laddr.IP.IsPrivate() {
		p.portMappingRegister <- &portMapping{
			protocol: "UDP",
			name:     "ethereum portal peer discovery",
			port:     laddr.Port,
		}
	}
	return nil
}

func (p *PortalProtocol) setupDiscV5AndTable() error {
	err := p.setupUDPListening()
	if err != nil {
		return err
	}

	cfg := discover.Config{
		PrivateKey:  p.PrivateKey,
		NetRestrict: p.NetRestrict,
		Bootnodes:   p.BootstrapNodes,
		Log:         p.Log,
	}

	p.table, err = discover.NewTable(p, p.localNode.Database(), cfg)
	if err != nil {
		return err
	}

	return nil
}

func (p *PortalProtocol) Ping(node *enode.Node) (uint64, error) {
	pong, err := p.pingInner(node)
	if err != nil {
		return 0, err
	}

	return pong.EnrSeq, nil
}

func (p *PortalProtocol) pingInner(node *enode.Node) (*Pong, error) {
	enrSeq := p.Self().Seq()
	radiusBytes, err := p.Radius().MarshalSSZ()
	if err != nil {
		return nil, err
	}
	customPayload := &PingPongCustomData{
		Radius: radiusBytes,
	}

	customPayloadBytes, err := customPayload.MarshalSSZ()
	if err != nil {
		return nil, err
	}

	pingRequest := &Ping{
		EnrSeq:        enrSeq,
		CustomPayload: customPayloadBytes,
	}

	p.Log.Trace(">> PING/"+p.protocolName, "protocol", p.protocolName, "ip", p.Self().IP().String(), "source", p.Self().ID(), "target", node.ID(), "ping", pingRequest)
	if metrics.Enabled {
		p.portalMetrics.messagesSentPing.Mark(1)
	}
	pingRequestBytes, err := pingRequest.MarshalSSZ()
	if err != nil {
		return nil, err
	}

	talkRequestBytes := make([]byte, 0, len(pingRequestBytes)+1)
	talkRequestBytes = append(talkRequestBytes, PING)
	talkRequestBytes = append(talkRequestBytes, pingRequestBytes...)

	talkResp, err := p.DiscV5.TalkRequest(node, p.protocolId, talkRequestBytes)

	if err != nil {
		return nil, err
	}

	p.Log.Trace("<< PONG/"+p.protocolName, "source", p.Self().ID(), "target", node.ID(), "res", talkResp)
	if metrics.Enabled {
		p.portalMetrics.messagesReceivedPong.Mark(1)
	}

	return p.processPong(node, talkResp)
}

func (p *PortalProtocol) findNodes(node *enode.Node, distances []uint) ([]*enode.Node, error) {
	if p.localNode.ID().String() == node.ID().String() {
		return make([]*enode.Node, 0), nil
	}

	distancesBytes := make([][2]byte, len(distances))
	for i, distance := range distances {
		copy(distancesBytes[i][:], ssz.MarshalUint16(make([]byte, 0), uint16(distance)))
	}

	findNodes := &FindNodes{
		Distances: distancesBytes,
	}

	p.Log.Trace(">> FIND_NODES/"+p.protocolName, "id", node.ID(), "findNodes", findNodes)
	if metrics.Enabled {
		p.portalMetrics.messagesSentFindNodes.Mark(1)
	}
	findNodesBytes, err := findNodes.MarshalSSZ()
	if err != nil {
		p.Log.Error("failed to marshal find nodes request", "err", err)
		return nil, err
	}

	talkRequestBytes := make([]byte, 0, len(findNodesBytes)+1)
	talkRequestBytes = append(talkRequestBytes, FINDNODES)
	talkRequestBytes = append(talkRequestBytes, findNodesBytes...)

	talkResp, err := p.DiscV5.TalkRequest(node, p.protocolId, talkRequestBytes)
	if err != nil {
		p.Log.Error("failed to send find nodes request", "ip", node.IP().String(), "port", node.UDP(), "err", err)
		return nil, err
	}

	return p.processNodes(node, talkResp, distances)
}

func (p *PortalProtocol) findContent(node *enode.Node, contentKey []byte) (byte, interface{}, error) {
	findContent := &FindContent{
		ContentKey: contentKey,
	}

	p.Log.Trace(">> FIND_CONTENT/"+p.protocolName, "id", node.ID(), "findContent", findContent)
	if metrics.Enabled {
		p.portalMetrics.messagesSentFindContent.Mark(1)
	}
	findContentBytes, err := findContent.MarshalSSZ()
	if err != nil {
		p.Log.Error("failed to marshal find content request", "err", err)
		return 0xff, nil, err
	}

	talkRequestBytes := make([]byte, 0, len(findContentBytes)+1)
	talkRequestBytes = append(talkRequestBytes, FINDCONTENT)
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

	offer := &Offer{
		ContentKeys: contentKeys,
	}

	p.Log.Trace(">> OFFER/"+p.protocolName, "offer", offer)
	if metrics.Enabled {
		p.portalMetrics.messagesSentOffer.Mark(1)
	}
	offerBytes, err := offer.MarshalSSZ()
	if err != nil {
		p.Log.Error("failed to marshal offer request", "err", err)
		return nil, err
	}

	talkRequestBytes := make([]byte, 0, len(offerBytes)+1)
	talkRequestBytes = append(talkRequestBytes, OFFER)
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
	if resp[0] != ACCEPT {
		return nil, fmt.Errorf("invalid accept response")
	}

	p.Log.Info("will process Offer", "id", target.ID(), "ip", target.IP().To4().String(), "port", target.UDP())

	accept := &Accept{}
	err = accept.UnmarshalSSZ(resp[1:])
	if err != nil {
		return nil, err
	}

	p.Log.Trace("<< ACCEPT/"+p.protocolName, "id", target.ID(), "accept", accept)
	if metrics.Enabled {
		p.portalMetrics.messagesReceivedAccept.Mark(1)
	}
	isAdded := p.table.AddFoundNode(target, true)
	if isAdded {
		log.Debug("Node added to bucket", "protocol", p.protocolName, "node", target.IP(), "port", target.UDP())
	} else {
		log.Debug("Node added to replacements list", "protocol", p.protocolName, "node", target.IP(), "port", target.UDP())
	}
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
		defer func() {
			if conn == nil {
				return
			}
			err := conn.Close()
			if err != nil {
				p.Log.Error("failed to close connection", "err", err)
			}
		}()
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
				conn, err = p.Utp.DialWithCid(connctx, target, libutp.ReceConnId(connId).SendId())
				conncancel()
				if err != nil {
					if metrics.Enabled {
						p.portalMetrics.utpOutFailConn.Inc(1)
					}
					p.Log.Error("failed to dial utp connection", "err", err)
					return
				}

				err = conn.SetWriteDeadline(time.Now().Add(defaultUTPWriteTimeout))
				if err != nil {
					if metrics.Enabled {
						p.portalMetrics.utpOutFailDeadline.Inc(1)
					}
					p.Log.Error("failed to set write deadline", "err", err)
					return
				}

				var written int
				written, err = conn.Write(contentsPayload)
				if err != nil {
					if metrics.Enabled {
						p.portalMetrics.utpOutFailWrite.Inc(1)
					}
					p.Log.Error("failed to write to utp connection", "err", err)
					return
				}
				p.Log.Trace(">> CONTENT/"+p.protocolName, "id", target.ID(), "contents", contents, "size", written)
				if metrics.Enabled {
					p.portalMetrics.messagesSentContent.Mark(1)
					p.portalMetrics.utpOutSuccess.Inc(1)
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

	if resp[0] != CONTENT {
		return 0xff, nil, fmt.Errorf("invalid content response")
	}

	p.Log.Info("will process content", "id", target.ID(), "ip", target.IP().To4().String(), "port", target.UDP())

	switch resp[1] {
	case ContentRawSelector:
		content := &Content{}
		err := content.UnmarshalSSZ(resp[2:])
		if err != nil {
			return 0xff, nil, err
		}

		p.Log.Trace("<< CONTENT/"+p.protocolName, "id", target.ID(), "content", content)
		if metrics.Enabled {
			p.portalMetrics.messagesReceivedContent.Mark(1)
		}
		isAdded := p.table.AddFoundNode(target, true)
		if isAdded {
			log.Debug("Node added to bucket", "protocol", p.protocolName, "node", target.IP(), "port", target.UDP())
		} else {
			log.Debug("Node added to replacements list", "protocol", p.protocolName, "node", target.IP(), "port", target.UDP())
		}
		return resp[1], content.Content, nil
	case ContentConnIdSelector:
		connIdMsg := &ConnectionId{}
		err := connIdMsg.UnmarshalSSZ(resp[2:])
		if err != nil {
			return 0xff, nil, err
		}

		p.Log.Trace("<< CONTENT_CONNECTION_ID/"+p.protocolName, "id", target.ID(), "resp", common.Bytes2Hex(resp), "connIdMsg", connIdMsg)
		if metrics.Enabled {
			p.portalMetrics.messagesReceivedContent.Mark(1)
		}
		isAdded := p.table.AddFoundNode(target, true)
		if isAdded {
			log.Debug("Node added to bucket", "protocol", p.protocolName, "node", target.IP(), "port", target.UDP())
		} else {
			log.Debug("Node added to replacements list", "protocol", p.protocolName, "node", target.IP(), "port", target.UDP())
		}
		connctx, conncancel := context.WithTimeout(p.closeCtx, defaultUTPConnectTimeout)
		connId := binary.BigEndian.Uint16(connIdMsg.Id[:])
		conn, err := p.Utp.DialWithCid(connctx, target, libutp.ReceConnId(connId).SendId())
		defer func() {
			if conn == nil {
				if metrics.Enabled {
					p.portalMetrics.utpInFailConn.Inc(1)
				}
				return
			}
			err := conn.Close()
			if err != nil {
				p.Log.Error("failed to close connection", "err", err)
			}
		}()
		conncancel()
		if err != nil {
			return 0xff, nil, err
		}

		err = conn.SetReadDeadline(time.Now().Add(defaultUTPReadTimeout))
		if err != nil {
			if metrics.Enabled {
				p.portalMetrics.utpInFailDeadline.Inc(1)
			}
			return 0xff, nil, err
		}
		// Read ALL the data from the connection until EOF and return it
		data, err := io.ReadAll(conn)
		if err != nil {
			if metrics.Enabled {
				p.portalMetrics.utpInFailRead.Inc(1)
			}
			p.Log.Error("failed to read from utp connection", "err", err)
			return 0xff, nil, err
		}
		p.Log.Trace("<< CONTENT/"+p.protocolName, "id", target.ID(), "size", len(data), "data", data)
		if metrics.Enabled {
			p.portalMetrics.messagesReceivedContent.Mark(1)
			p.portalMetrics.utpInSuccess.Inc(1)
		}
		return resp[1], data, nil
	case ContentEnrsSelector:
		enrs := &Enrs{}
		err := enrs.UnmarshalSSZ(resp[2:])

		if err != nil {
			return 0xff, nil, err
		}

		p.Log.Trace("<< CONTENT_ENRS/"+p.protocolName, "id", target.ID(), "enrs", enrs)
		if metrics.Enabled {
			p.portalMetrics.messagesReceivedContent.Mark(1)
		}
		isAdded := p.table.AddFoundNode(target, true)
		if isAdded {
			log.Debug("Node added to bucket", "protocol", p.protocolName, "node", target.IP(), "port", target.UDP())
		} else {
			log.Debug("Node added to replacements list", "protocol", p.protocolName, "node", target.IP(), "port", target.UDP())
		}
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

	if resp[0] != NODES {
		return nil, fmt.Errorf("invalid nodes response")
	}

	nodesResp := &Nodes{}
	err := nodesResp.UnmarshalSSZ(resp[1:])
	if err != nil {
		return nil, err
	}

	isAdded := p.table.AddFoundNode(target, true)
	if isAdded {
		log.Debug("Node added to bucket", "protocol", p.protocolName, "node", target.IP(), "port", target.UDP())
	} else {
		log.Debug("Node added to replacements list", "protocol", p.protocolName, "node", target.IP(), "port", target.UDP())
	}
	nodes := p.filterNodes(target, nodesResp.Enrs, distances)

	return nodes, nil
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

	p.Log.Trace("<< NODES/"+p.protocolName, "id", target.ID(), "total", len(enrs), "verified", verified, "nodes", nodes)
	if metrics.Enabled {
		p.portalMetrics.messagesReceivedNodes.Mark(1)
	}
	return nodes
}

func (p *PortalProtocol) processPong(target *enode.Node, resp []byte) (*Pong, error) {
	if len(resp) == 0 {
		return nil, ErrEmptyResp
	}
	if resp[0] != PONG {
		return nil, fmt.Errorf("invalid pong response")
	}
	pong := &Pong{}
	err := pong.UnmarshalSSZ(resp[1:])
	if err != nil {
		return nil, err
	}

	p.Log.Trace("<< PONG_RESPONSE/"+p.protocolName, "id", target.ID(), "pong", pong)
	if metrics.Enabled {
		p.portalMetrics.messagesReceivedPong.Mark(1)
	}

	customPayload := &PingPongCustomData{}
	err = customPayload.UnmarshalSSZ(pong.CustomPayload)
	if err != nil {
		return nil, err
	}

	p.Log.Trace("<< PONG_RESPONSE/"+p.protocolName, "id", target.ID(), "pong", pong, "customPayload", customPayload)
	if metrics.Enabled {
		p.portalMetrics.messagesReceivedPong.Mark(1)
	}
	isAdded := p.table.AddFoundNode(target, true)
	if isAdded {
		log.Debug("Node added to bucket", "protocol", p.protocolName, "node", target.IP(), "port", target.UDP())
	} else {
		log.Debug("Node added to replacements list", "protocol", p.protocolName, "node", target.IP(), "port", target.UDP())
	}

	p.radiusCache.Set([]byte(target.ID().String()), customPayload.Radius)
	return pong, nil
}

func (p *PortalProtocol) handleTalkRequest(id enode.ID, addr *net.UDPAddr, msg []byte) []byte {
	if n := p.DiscV5.GetNode(id); n != nil {
		p.table.AddInboundNode(n)
	}

	msgCode := msg[0]

	switch msgCode {
	case PING:
		pingRequest := &Ping{}
		err := pingRequest.UnmarshalSSZ(msg[1:])
		if err != nil {
			p.Log.Error("failed to unmarshal ping request", "err", err)
			return nil
		}

		p.Log.Trace("<< PING/"+p.protocolName, "protocol", p.protocolName, "source", id, "pingRequest", pingRequest)
		if metrics.Enabled {
			p.portalMetrics.messagesReceivedPing.Mark(1)
		}
		resp, err := p.handlePing(id, pingRequest)
		if err != nil {
			p.Log.Error("failed to handle ping request", "err", err)
			return nil
		}

		return resp
	case FINDNODES:
		findNodesRequest := &FindNodes{}
		err := findNodesRequest.UnmarshalSSZ(msg[1:])
		if err != nil {
			p.Log.Error("failed to unmarshal find nodes request", "err", err)
			return nil
		}

		p.Log.Trace("<< FIND_NODES/"+p.protocolName, "protocol", p.protocolName, "source", id, "findNodesRequest", findNodesRequest)
		if metrics.Enabled {
			p.portalMetrics.messagesReceivedFindNodes.Mark(1)
		}
		resp, err := p.handleFindNodes(addr, findNodesRequest)
		if err != nil {
			p.Log.Error("failed to handle find nodes request", "err", err)
			return nil
		}

		return resp
	case FINDCONTENT:
		findContentRequest := &FindContent{}
		err := findContentRequest.UnmarshalSSZ(msg[1:])
		if err != nil {
			p.Log.Error("failed to unmarshal find content request", "err", err)
			return nil
		}

		p.Log.Trace("<< FIND_CONTENT/"+p.protocolName, "protocol", p.protocolName, "source", id, "findContentRequest", findContentRequest)
		if metrics.Enabled {
			p.portalMetrics.messagesReceivedFindContent.Mark(1)
		}
		resp, err := p.handleFindContent(id, addr, findContentRequest)
		if err != nil {
			p.Log.Error("failed to handle find content request", "err", err)
			return nil
		}

		return resp
	case OFFER:
		offerRequest := &Offer{}
		err := offerRequest.UnmarshalSSZ(msg[1:])
		if err != nil {
			p.Log.Error("failed to unmarshal offer request", "err", err)
			return nil
		}

		p.Log.Trace("<< OFFER/"+p.protocolName, "protocol", p.protocolName, "source", id, "offerRequest", offerRequest)
		if metrics.Enabled {
			p.portalMetrics.messagesReceivedOffer.Mark(1)
		}
		resp, err := p.handleOffer(id, addr, offerRequest)
		if err != nil {
			p.Log.Error("failed to handle offer request", "err", err)
			return nil
		}

		return resp
	}

	return nil
}

func (p *PortalProtocol) handlePing(id enode.ID, ping *Ping) ([]byte, error) {
	pingCustomPayload := &PingPongCustomData{}
	err := pingCustomPayload.UnmarshalSSZ(ping.CustomPayload)
	if err != nil {
		return nil, err
	}

	p.radiusCache.Set([]byte(id.String()), pingCustomPayload.Radius)

	enrSeq := p.Self().Seq()
	radiusBytes, err := p.Radius().MarshalSSZ()
	if err != nil {
		return nil, err
	}
	pongCustomPayload := &PingPongCustomData{
		Radius: radiusBytes,
	}

	pongCustomPayloadBytes, err := pongCustomPayload.MarshalSSZ()
	if err != nil {
		return nil, err
	}

	pong := &Pong{
		EnrSeq:        enrSeq,
		CustomPayload: pongCustomPayloadBytes,
	}

	p.Log.Trace(">> PONG/"+p.protocolName, "protocol", p.protocolName, "source", id, "pong", pong)
	if metrics.Enabled {
		p.portalMetrics.messagesSentPong.Mark(1)
	}
	pongBytes, err := pong.MarshalSSZ()

	if err != nil {
		return nil, err
	}

	talkRespBytes := make([]byte, 0, len(pongBytes)+1)
	talkRespBytes = append(talkRespBytes, PONG)
	talkRespBytes = append(talkRespBytes, pongBytes...)

	return talkRespBytes, nil
}

func (p *PortalProtocol) handleFindNodes(fromAddr *net.UDPAddr, request *FindNodes) ([]byte, error) {
	distances := make([]uint, len(request.Distances))
	for i, distance := range request.Distances {
		distances[i] = uint(ssz.UnmarshallUint16(distance[:]))
	}

	nodes := p.collectTableNodes(fromAddr.IP, distances, portalFindnodesResultLimit)

	nodesOverhead := 1 + 1 + 4 // msg id + total + container offset
	maxPayloadSize := v5wire.MaxPacketSize - talkRespOverhead - nodesOverhead
	enrOverhead := 4 //per added ENR, 4 bytes offset overhead

	enrs := p.truncateNodes(nodes, maxPayloadSize, enrOverhead)

	nodesMsg := &Nodes{
		Total: 1,
		Enrs:  enrs,
	}

	p.Log.Trace(">> NODES/"+p.protocolName, "protocol", p.protocolName, "source", fromAddr, "nodes", nodesMsg)
	if metrics.Enabled {
		p.portalMetrics.messagesSentNodes.Mark(1)
	}
	nodesMsgBytes, err := nodesMsg.MarshalSSZ()
	if err != nil {
		return nil, err
	}

	talkRespBytes := make([]byte, 0, len(nodesMsgBytes)+1)
	talkRespBytes = append(talkRespBytes, NODES)
	talkRespBytes = append(talkRespBytes, nodesMsgBytes...)

	return talkRespBytes, nil
}

func (p *PortalProtocol) handleFindContent(id enode.ID, addr *net.UDPAddr, request *FindContent) ([]byte, error) {
	contentOverhead := 1 + 1 // msg id + SSZ Union selector
	maxPayloadSize := v5wire.MaxPacketSize - talkRespOverhead - contentOverhead
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

		enrsMsg := &Enrs{
			Enrs: enrs,
		}

		p.Log.Trace(">> CONTENT_ENRS/"+p.protocolName, "protocol", p.protocolName, "source", addr, "enrs", enrsMsg)
		if metrics.Enabled {
			p.portalMetrics.messagesSentContent.Mark(1)
		}
		var enrsMsgBytes []byte
		enrsMsgBytes, err = enrsMsg.MarshalSSZ()
		if err != nil {
			return nil, err
		}

		contentMsgBytes := make([]byte, 0, len(enrsMsgBytes)+1)
		contentMsgBytes = append(contentMsgBytes, ContentEnrsSelector)
		contentMsgBytes = append(contentMsgBytes, enrsMsgBytes...)

		talkRespBytes := make([]byte, 0, len(contentMsgBytes)+1)
		talkRespBytes = append(talkRespBytes, CONTENT)
		talkRespBytes = append(talkRespBytes, contentMsgBytes...)

		return talkRespBytes, nil
	} else if len(content) <= maxPayloadSize {
		rawContentMsg := &Content{
			Content: content,
		}

		p.Log.Trace(">> CONTENT_RAW/"+p.protocolName, "protocol", p.protocolName, "source", addr, "content", rawContentMsg)
		if metrics.Enabled {
			p.portalMetrics.messagesSentContent.Mark(1)
		}

		var rawContentMsgBytes []byte
		rawContentMsgBytes, err = rawContentMsg.MarshalSSZ()
		if err != nil {
			return nil, err
		}

		contentMsgBytes := make([]byte, 0, len(rawContentMsgBytes)+1)
		contentMsgBytes = append(contentMsgBytes, ContentRawSelector)
		contentMsgBytes = append(contentMsgBytes, rawContentMsgBytes...)

		talkRespBytes := make([]byte, 0, len(contentMsgBytes)+1)
		talkRespBytes = append(talkRespBytes, CONTENT)
		talkRespBytes = append(talkRespBytes, contentMsgBytes...)

		return talkRespBytes, nil
	} else {
		connectionId := p.connIdGen.GenCid(id, false)

		go func(bctx context.Context, connId *libutp.ConnId) {
			var conn *utp.Conn
			var connectCtx context.Context
			var cancel context.CancelFunc
			defer func() {
				p.connIdGen.Remove(connectionId)
				if conn == nil {
					return
				}
				err := conn.Close()
				if err != nil {
					p.Log.Error("failed to close connection", "err", err)
				}
			}()
			for {
				select {
				case <-bctx.Done():
					return
				default:
					p.Log.Debug("will accept find content conn from: ", "nodeId", id.String(), "source", addr, "connId", connId)
					connectCtx, cancel = context.WithTimeout(bctx, defaultUTPConnectTimeout)
					conn, err = p.Utp.AcceptWithCid(connectCtx, id, connectionId)
					cancel()
					if err != nil {
						if metrics.Enabled {
							p.portalMetrics.utpOutFailConn.Inc(1)
						}
						p.Log.Error("failed to accept utp connection for handle find content", "connId", connectionId.SendId(), "err", err)
						return
					}

					err = conn.SetWriteDeadline(time.Now().Add(defaultUTPWriteTimeout))
					if err != nil {
						if metrics.Enabled {
							p.portalMetrics.utpOutFailDeadline.Inc(1)
						}
						p.Log.Error("failed to set write deadline", "err", err)
						return
					}

					var n int
					n, err = conn.Write(content)
					if err != nil {
						if metrics.Enabled {
							p.portalMetrics.utpOutFailWrite.Inc(1)
						}
						p.Log.Error("failed to write content to utp connection", "err", err)
						return
					}

					if metrics.Enabled {
						p.portalMetrics.utpOutSuccess.Inc(1)
					}
					p.Log.Trace("wrote content size to utp connection", "n", n)
					return
				}
			}
		}(p.closeCtx, connectionId)

		idBuffer := make([]byte, 2)
		binary.BigEndian.PutUint16(idBuffer, connectionId.SendId())
		connIdMsg := &ConnectionId{
			Id: idBuffer,
		}

		p.Log.Trace(">> CONTENT_CONNECTION_ID/"+p.protocolName, "protocol", p.protocolName, "source", addr, "connId", connIdMsg)
		if metrics.Enabled {
			p.portalMetrics.messagesSentContent.Mark(1)
		}
		var connIdMsgBytes []byte
		connIdMsgBytes, err = connIdMsg.MarshalSSZ()
		if err != nil {
			return nil, err
		}

		contentMsgBytes := make([]byte, 0, len(connIdMsgBytes)+1)
		contentMsgBytes = append(contentMsgBytes, ContentConnIdSelector)
		contentMsgBytes = append(contentMsgBytes, connIdMsgBytes...)

		talkRespBytes := make([]byte, 0, len(contentMsgBytes)+1)
		talkRespBytes = append(talkRespBytes, CONTENT)
		talkRespBytes = append(talkRespBytes, contentMsgBytes...)

		return talkRespBytes, nil
	}
}

func (p *PortalProtocol) handleOffer(id enode.ID, addr *net.UDPAddr, request *Offer) ([]byte, error) {
	var err error
	contentKeyBitlist := bitfield.NewBitlist(uint64(len(request.ContentKeys)))
	if len(p.contentQueue) >= cap(p.contentQueue) {
		acceptMsg := &Accept{
			ConnectionId: []byte{0, 0},
			ContentKeys:  contentKeyBitlist,
		}

		p.Log.Trace(">> ACCEPT/"+p.protocolName, "protocol", p.protocolName, "source", addr, "accept", acceptMsg)
		if metrics.Enabled {
			p.portalMetrics.messagesSentAccept.Mark(1)
		}
		var acceptMsgBytes []byte
		acceptMsgBytes, err = acceptMsg.MarshalSSZ()
		if err != nil {
			return nil, err
		}

		talkRespBytes := make([]byte, 0, len(acceptMsgBytes)+1)
		talkRespBytes = append(talkRespBytes, ACCEPT)
		talkRespBytes = append(talkRespBytes, acceptMsgBytes...)

		return talkRespBytes, nil
	}

	contentKeys := make([][]byte, 0)
	for i, contentKey := range request.ContentKeys {
		contentId := p.toContentId(contentKey)
		if contentId != nil {
			if inRange(p.Self().ID(), p.Radius(), contentId) {
				if _, err = p.storage.Get(contentKey, contentId); err != nil {
					contentKeyBitlist.SetBitAt(uint64(i), true)
					contentKeys = append(contentKeys, contentKey)
				}
			}
		} else {
			return nil, ErrNilContentKey
		}
	}

	idBuffer := make([]byte, 2)
	if contentKeyBitlist.Count() != 0 {
		connectionId := p.connIdGen.GenCid(id, false)

		go func(bctx context.Context, connId *libutp.ConnId) {
			var conn *utp.Conn
			var connectCtx context.Context
			var cancel context.CancelFunc
			defer func() {
				p.connIdGen.Remove(connectionId)
				if conn == nil {
					return
				}
				err := conn.Close()
				if err != nil {
					p.Log.Error("failed to close connection", "err", err)
				}
			}()
			for {
				select {
				case <-bctx.Done():
					return
				default:
					p.Log.Debug("will accept offer conn from: ", "source", addr, "connId", connId)
					connectCtx, cancel = context.WithTimeout(bctx, defaultUTPConnectTimeout)
					conn, err = p.Utp.AcceptWithCid(connectCtx, id, connectionId)
					cancel()
					if err != nil {
						if metrics.Enabled {
							p.portalMetrics.utpInFailConn.Inc(1)
						}
						p.Log.Error("failed to accept utp connection for handle offer", "connId", connectionId.SendId(), "err", err)
						return
					}

					err = conn.SetReadDeadline(time.Now().Add(defaultUTPReadTimeout))
					if err != nil {
						if metrics.Enabled {
							p.portalMetrics.utpInFailDeadline.Inc(1)
						}
						p.Log.Error("failed to set read deadline", "err", err)
						return
					}
					// Read ALL the data from the connection until EOF and return it
					var data []byte
					data, err = io.ReadAll(conn)
					if err != nil {
						if metrics.Enabled {
							p.portalMetrics.utpInFailRead.Inc(1)
						}
						p.Log.Error("failed to read from utp connection", "err", err)
						return
					}
					p.Log.Trace("<< OFFER_CONTENT/"+p.protocolName, "id", id, "size", len(data), "data", data)
					if metrics.Enabled {
						p.portalMetrics.messagesReceivedContent.Mark(1)
					}

					err = p.handleOfferedContents(id, contentKeys, data)
					if err != nil {
						p.Log.Error("failed to handle offered Contents", "err", err)
						return
					}

					if metrics.Enabled {
						p.portalMetrics.utpInSuccess.Inc(1)
					}
					return
				}
			}
		}(p.closeCtx, connectionId)

		binary.BigEndian.PutUint16(idBuffer, connectionId.SendId())
	} else {
		binary.BigEndian.PutUint16(idBuffer, uint16(0))
	}

	acceptMsg := &Accept{
		ConnectionId: idBuffer,
		ContentKeys:  []byte(contentKeyBitlist),
	}

	p.Log.Trace(">> ACCEPT/"+p.protocolName, "protocol", p.protocolName, "source", addr, "accept", acceptMsg)
	if metrics.Enabled {
		p.portalMetrics.messagesSentAccept.Mark(1)
	}
	var acceptMsgBytes []byte
	acceptMsgBytes, err = acceptMsg.MarshalSSZ()
	if err != nil {
		return nil, err
	}

	talkRespBytes := make([]byte, 0, len(acceptMsgBytes)+1)
	talkRespBytes = append(talkRespBytes, ACCEPT)
	talkRespBytes = append(talkRespBytes, acceptMsgBytes...)

	return talkRespBytes, nil
}

func (p *PortalProtocol) handleOfferedContents(id enode.ID, keys [][]byte, payload []byte) error {
	contents, err := decodeContents(payload)
	if err != nil {
		if metrics.Enabled {
			p.portalMetrics.contentDecodedFalse.Inc(1)
		}
		return err
	}

	keyLen := len(keys)
	contentLen := len(contents)
	if keyLen != contentLen {
		if metrics.Enabled {
			p.portalMetrics.contentDecodedFalse.Inc(1)
		}
		return fmt.Errorf("content keys len %d doesn't match content values len %d", keyLen, contentLen)
	}

	contentElement := &ContentElement{
		Node:        id,
		ContentKeys: keys,
		Contents:    contents,
	}

	p.contentQueue <- contentElement

	if metrics.Enabled {
		p.portalMetrics.contentDecodedTrue.Inc(1)
	}
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
		return nil, discover.ErrLowPort
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

// LookupRandom looks up a random target.
// This is needed to satisfy the transport interface.
func (p *PortalProtocol) LookupRandom() []*enode.Node {
	return p.newRandomLookup(p.closeCtx).Run()
}

// LookupSelf looks up our own node ID.
// This is needed to satisfy the transport interface.
func (p *PortalProtocol) LookupSelf() []*enode.Node {
	return p.newLookup(p.closeCtx, p.Self().ID()).Run()
}

func (p *PortalProtocol) newRandomLookup(ctx context.Context) *discover.Lookup {
	var target enode.ID
	_, _ = crand.Read(target[:])
	return p.newLookup(ctx, target)
}

func (p *PortalProtocol) newLookup(ctx context.Context, target enode.ID) *discover.Lookup {
	return discover.NewLookup(ctx, p.table, target, func(n *enode.Node) ([]*enode.Node, error) {
		return p.lookupWorker(n, target)
	})
}

// lookupWorker performs FINDNODE calls against a single node during lookup.
func (p *PortalProtocol) lookupWorker(destNode *enode.Node, target enode.ID) ([]*enode.Node, error) {
	var (
		dists = discover.LookupDistances(target, destNode.ID())
		nodes = discover.NodesByDistance{Target: target}
		err   error
	)
	var r []*enode.Node

	r, err = p.findNodes(destNode, dists)
	if errors.Is(err, discover.ErrClosed) {
		return nil, err
	}
	for _, n := range r {
		if n.ID() != p.Self().ID() {
			isAdded := p.table.AddFoundNode(n, false)
			if isAdded {
				log.Debug("Node added to bucket", "protocol", p.protocolName, "node", n.IP(), "port", n.UDP())
			} else {
				log.Debug("Node added to replacements list", "protocol", p.protocolName, "node", n.IP(), "port", n.UDP())
			}
			nodes.Push(n, portalFindnodesResultLimit)
		}
	}
	return nodes.Entries, err
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
	allNodes := p.table.NodeList()
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
	return p.newLookup(p.closeCtx, target).Run()
}

// Resolve searches for a specific Node with the given ID and tries to get the most recent
// version of the Node record for it. It returns n if the Node could not be resolved.
func (p *PortalProtocol) Resolve(n *enode.Node) *enode.Node {
	if intable := p.table.GetNode(n.ID()); intable != nil && intable.Seq() > n.Seq() {
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
		p.Log.Debug("Resolve Self Id", "id", id.String())
		return p.Self()
	}

	n := p.table.GetNode(id)
	if n != nil {
		p.Log.Debug("found Id in table and will request enr from the node", "id", id.String())
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

		checkLive := !p.table.Config().NoFindnodeLivenessCheck
		for _, n := range p.table.AppendBucketNodes(dist, bn[:0], checkLive) {
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

	resChan := make(chan *traceContentInfoResp, discover.Alpha)
	hasResult := int32(0)

	result := ContentInfoResp{}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		for res := range resChan {
			if res.Flag != ContentEnrsSelector {
				result.Content = res.Content.([]byte)
				result.UtpTransfer = res.UtpTransfer
			}
		}
	}()

	discover.NewLookup(lookupContext, p.table, enode.ID(contentId), func(n *enode.Node) ([]*enode.Node, error) {
		return p.contentLookupWorker(n, contentKey, resChan, cancel, &hasResult)
	}).Run()
	close(resChan)

	wg.Wait()
	if hasResult == 1 {
		return result.Content, result.UtpTransfer, nil
	}
	defer cancel()
	return nil, false, ContentNotFound
}

func (p *PortalProtocol) TraceContentLookup(contentKey, contentId []byte) (*TraceContentResult, error) {
	lookupContext, cancel := context.WithCancel(context.Background())
	// resp channel
	resChan := make(chan *traceContentInfoResp, discover.Alpha)

	hasResult := int32(0)

	traceContentRes := &TraceContentResult{}

	selfHexId := "0x" + p.Self().ID().String()

	trace := &Trace{
		Origin:      selfHexId,
		TargetId:    hexutil.Encode(contentId),
		StartedAtMs: int(time.Now().UnixMilli()),
		Responses:   make(map[string]RespByNode),
		Metadata:    make(map[string]*NodeMetadata),
		Cancelled:   make([]string, 0),
	}

	nodes := p.table.FindnodeByID(enode.ID(contentId), discover.BucketSize, false)

	localResponse := make([]string, 0, len(nodes.Entries))
	for _, node := range nodes.Entries {
		id := "0x" + node.ID().String()
		localResponse = append(localResponse, id)
	}
	trace.Responses[selfHexId] = RespByNode{
		DurationMs:    0,
		RespondedWith: localResponse,
	}

	dis := p.Distance(p.Self().ID(), enode.ID(contentId))

	trace.Metadata[selfHexId] = &NodeMetadata{
		Enr:      p.Self().String(),
		Distance: hexutil.Encode(dis[:]),
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		for res := range resChan {
			node := res.Node
			hexId := "0x" + node.ID().String()
			dis := p.Distance(node.ID(), enode.ID(contentId))
			p.Log.Debug("reveice res", "id", hexId, "flag", res.Flag)
			trace.Metadata[hexId] = &NodeMetadata{
				Enr:      node.String(),
				Distance: hexutil.Encode(dis[:]),
			}
			// no content return
			if traceContentRes.Content == "" {
				if res.Flag == ContentRawSelector || res.Flag == ContentConnIdSelector {
					trace.ReceivedFrom = hexId
					content := res.Content.([]byte)
					traceContentRes.Content = hexutil.Encode(content)
					traceContentRes.UtpTransfer = res.UtpTransfer
					trace.Responses[hexId] = RespByNode{}
				} else {
					nodes := res.Content.([]*enode.Node)
					respByNode := RespByNode{
						RespondedWith: make([]string, 0, len(nodes)),
					}
					for _, node := range nodes {
						idInner := "0x" + node.ID().String()
						respByNode.RespondedWith = append(respByNode.RespondedWith, idInner)
						if _, ok := trace.Metadata[idInner]; !ok {
							dis := p.Distance(node.ID(), enode.ID(contentId))
							trace.Metadata[idInner] = &NodeMetadata{
								Enr:      node.String(),
								Distance: hexutil.Encode(dis[:]),
							}
						}
						trace.Responses[hexId] = respByNode
					}
				}
			} else {
				trace.Cancelled = append(trace.Cancelled, hexId)
			}
		}
	}()

	lookup := discover.NewLookup(lookupContext, p.table, enode.ID(contentId), func(n *enode.Node) ([]*enode.Node, error) {
		return p.contentLookupWorker(n, contentKey, resChan, cancel, &hasResult)
	})
	lookup.Run()
	close(resChan)

	wg.Wait()
	if hasResult == 0 {
		cancel()
	}
	traceContentRes.Trace = *trace

	return traceContentRes, nil
}

func (p *PortalProtocol) contentLookupWorker(n *enode.Node, contentKey []byte, resChan chan<- *traceContentInfoResp, cancel context.CancelFunc, done *int32) ([]*enode.Node, error) {
	wrapedNode := make([]*enode.Node, 0)
	flag, content, err := p.findContent(n, contentKey)
	if err != nil {
		return nil, err
	}
	p.Log.Debug("traceContentLookupWorker reveice response", "ip", n.IP().String(), "flag", flag)

	switch flag {
	case ContentRawSelector, ContentConnIdSelector:
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
		if flag == ContentConnIdSelector {
			res.UtpTransfer = true
		}
		if atomic.CompareAndSwapInt32(done, 0, 1) {
			p.Log.Debug("contentLookupWorker find content", "ip", n.IP().String(), "port", n.UDP())
			resChan <- res
			cancel()
		}
		return wrapedNode, err
	case ContentEnrsSelector:
		nodes, ok := content.([]*enode.Node)
		if !ok {
			return wrapedNode, fmt.Errorf("failed to assert to enrs content, value is: %v", content)
		}
		resChan <- &traceContentInfoResp{
			Node:        n,
			Flag:        flag,
			Content:     content,
			UtpTransfer: false,
		}
		return nodes, nil
	}
	return wrapedNode, nil
}

func (p *PortalProtocol) ToContentId(contentKey []byte) []byte {
	return p.toContentId(contentKey)
}

func (p *PortalProtocol) InRange(contentId []byte) bool {
	return inRange(p.Self().ID(), p.Radius(), contentId)
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

	contentList := make([]*ContentEntry, 0, ContentKeysLimit)
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
	p.Log.Debug("closest local nodes", "count", len(closestLocalNodes))

	gossipNodes := make([]*enode.Node, 0)
	for _, n := range closestLocalNodes {
		radius, found := p.radiusCache.HasGet(nil, []byte(n.ID().String()))
		if found {
			p.Log.Debug("found closest local nodes", "nodeId", n.ID(), "addr", n.IPAddr().String())
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
