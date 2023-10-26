package discover

import (
	"context"
	"crypto/ecdsa"
	crand "crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sort"
	"time"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover/portalwire"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/p2p/netutil"
	"github.com/ethereum/go-ethereum/rlp"
	ssz "github.com/ferranbt/fastssz"
	"github.com/holiman/uint256"
	"github.com/optimism-java/utp-go"
)

const (
	// This is the fairness knob for the discovery mixer. When looking for peers, we'll
	// wait this long for a single source of candidates before moving on and trying other
	// sources.
	discmixTimeout = 5 * time.Second

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

	defaultUTPAcceptTimeout = 15 * time.Second

	defaultUTPWriteTimeout = 60 * time.Second
)

type PortalProtocolConfig struct {
	BootstrapNodes []*enode.Node

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
		ListenAddr:      ":9000",
		NetRestrict:     nil,
		NodeRadius:      nodeRadius,
		RadiusCacheSize: 32 * 1024 * 1024,
		NodeDBPath:      "",
	}
}

type PortalProtocol struct {
	table *Table

	protocolId string

	nodeRadius     *uint256.Int
	DiscV5         *UDPv5
	utp            *utp.Listener
	ListenAddr     string
	localNode      *enode.LocalNode
	log            log.Logger
	discmix        *enode.FairMix
	PrivateKey     *ecdsa.PrivateKey
	NetRestrict    *netutil.Netlist
	BootstrapNodes []*enode.Node

	validSchemes   enr.IdentityScheme
	radiusCache    *fastcache.Cache
	closeCtx       context.Context
	cancelCloseCtx context.CancelFunc
	storage        Storage
}

func NewPortalProtocol(config *PortalProtocolConfig, protocolId string, privateKey *ecdsa.PrivateKey) (*PortalProtocol, error) {
	nodeDB, err := enode.OpenDB(config.NodeDBPath)
	if err != nil {
		return nil, err
	}

	localNode := enode.NewLocalNode(nodeDB, privateKey)
	localNode.SetFallbackIP(net.IP{127, 0, 0, 1})
	closeCtx, cancelCloseCtx := context.WithCancel(context.Background())

	protocol := &PortalProtocol{
		protocolId:     protocolId,
		ListenAddr:     config.ListenAddr,
		log:            log.New("protocol", protocolId),
		PrivateKey:     privateKey,
		NetRestrict:    config.NetRestrict,
		BootstrapNodes: config.BootstrapNodes,
		nodeRadius:     config.NodeRadius,
		radiusCache:    fastcache.New(config.RadiusCacheSize),
		closeCtx:       closeCtx,
		cancelCloseCtx: cancelCloseCtx,
		localNode:      localNode,
		validSchemes:   enode.ValidSchemes,
	}

	return protocol, nil
}

func (p *PortalProtocol) Start() error {
	err := p.setupDiscV5AndTable()
	if err != nil {
		return err
	}

	p.DiscV5.RegisterTalkHandler(p.protocolId, p.handleTalkRequest)

	go p.table.loop()
	return nil
}

func (p *PortalProtocol) setupUDPListening() (*net.UDPConn, error) {
	listenAddr := p.ListenAddr

	addr, err := net.ResolveUDPAddr("udp", listenAddr)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}
	laddr := conn.LocalAddr().(*net.UDPAddr)
	p.localNode.SetFallbackUDP(laddr.Port)
	p.log.Debug("UDP listener up", "addr", laddr)
	// TODO: NAT
	//if !laddr.IP.IsLoopback() && !laddr.IP.IsPrivate() {
	//	srv.portMappingRegister <- &portMapping{
	//		protocol: "UDP",
	//		name:     "ethereum peer discovery",
	//		port:     laddr.Port,
	//	}
	//}

	p.utp, err = utp.ListenUTP("udp", (*utp.Addr)(laddr))
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (p *PortalProtocol) setupDiscV5AndTable() error {
	p.discmix = enode.NewFairMix(discmixTimeout)

	conn, err := p.setupUDPListening()
	if err != nil {
		return err
	}

	cfg := Config{
		PrivateKey:  p.PrivateKey,
		NetRestrict: p.NetRestrict,
		Bootnodes:   p.BootstrapNodes,
		Log:         p.log,
	}
	p.DiscV5, err = ListenV5(conn, p.localNode, cfg)
	if err != nil {
		return err
	}

	p.table, err = newMeteredTable(p, p.localNode.Database(), cfg)
	if err != nil {
		return err
	}

	return nil
}

func (p *PortalProtocol) findNodes(node *enode.Node, distances []uint) ([]*enode.Node, error) {
	distancesBytes := make([][2]byte, len(distances))
	for i, distance := range distances {
		copy(distancesBytes[i][:], ssz.MarshalUint16(make([]byte, 0), uint16(distance)))
	}

	findNodes := &portalwire.FindNodes{
		Distances: distancesBytes,
	}

	p.log.Trace("Sending find nodes request", "id", node.ID(), "findNodes", findNodes)
	findNodesBytes, err := findNodes.MarshalSSZ()
	if err != nil {
		p.log.Error("failed to marshal find nodes request", "err", err)
		return nil, err
	}

	talkRequestBytes := make([]byte, 0, len(findNodesBytes)+1)
	talkRequestBytes = append(talkRequestBytes, portalwire.FINDNODES)
	talkRequestBytes = append(talkRequestBytes, findNodesBytes...)

	talkResp, err := p.DiscV5.TalkRequest(node, p.protocolId, talkRequestBytes)
	if err != nil {
		p.log.Error("failed to send find nodes request", "err", err)
		return nil, err
	}

	return p.processNodes(node, talkResp, distances)
}

func (p *PortalProtocol) processNodes(target *enode.Node, resp []byte, distances []uint) ([]*enode.Node, error) {
	var (
		nodes    []*enode.Node
		seen     = make(map[enode.ID]struct{})
		err      error
		verified = 0
	)

	if resp[0] != portalwire.NODES {
		return nil, fmt.Errorf("invalid nodes response")
	}

	nodesResp := &portalwire.Nodes{}
	err = nodesResp.UnmarshalSSZ(resp[1:])
	if err != nil {
		return nil, err
	}

	p.table.addVerifiedNode(wrapNode(target))
	var n *enode.Node
	for _, b := range nodesResp.Enrs {
		record := &enr.Record{}
		err = rlp.DecodeBytes(b, record)
		if err != nil {
			p.log.Debug("Invalid record in nodes response", "id", target.ID(), "err", err)
			continue
		}
		n, err = p.verifyResponseNode(target, record, distances, seen)
		if err != nil {
			p.log.Debug("Invalid record in nodes response", "id", target.ID(), "err", err)
			continue
		}
		verified++
		nodes = append(nodes, n)
	}

	p.log.Trace("Received nodes response", "id", target.ID(), "total", nodesResp.Total, "verified", verified, "nodes", nodes)
	return nodes, nil
}

func (p *PortalProtocol) processPong(target *enode.Node, resp []byte) (uint64, error) {
	if resp[0] != portalwire.PONG {
		return 0, fmt.Errorf("invalid pong response")
	}
	pong := &portalwire.Pong{}
	err := pong.UnmarshalSSZ(resp[1:])
	if err != nil {
		p.replaceNode(target)
		return 0, err
	}

	p.log.Trace("Received pong response", "id", target.ID(), "pong", pong)

	customPayload := &portalwire.PingPongCustomData{}
	err = customPayload.UnmarshalSSZ(pong.CustomPayload)
	if err != nil {
		p.replaceNode(target)
		return 0, err
	}

	p.log.Trace("Received pong response", "id", target.ID(), "pong", pong, "customPayload", customPayload)

	p.radiusCache.Set([]byte(target.ID().String()), customPayload.Radius)
	p.table.addVerifiedNode(wrapNode(target))
	return pong.EnrSeq, nil
}

func (p *PortalProtocol) handleTalkRequest(id enode.ID, addr *net.UDPAddr, msg []byte) []byte {
	if node := p.DiscV5.getNode(id); node != nil {
		p.table.addSeenNode(wrapNode(node))
	}

	msgCode := msg[0]

	switch msgCode {
	case portalwire.PING:
		pingRequest := &portalwire.Ping{}
		err := pingRequest.UnmarshalSSZ(msg[1:])
		if err != nil {
			p.log.Error("failed to unmarshal ping request", "err", err)
			return nil
		}

		p.log.Trace("received ping request", "protocol", p.protocolId, "source", id, "pingRequest", pingRequest)
		resp, err := p.handlePing(id, pingRequest)
		if err != nil {
			p.log.Error("failed to handle ping request", "err", err)
			return nil
		}

		return resp
	case portalwire.FINDNODES:
		findNodesRequest := &portalwire.FindNodes{}
		err := findNodesRequest.UnmarshalSSZ(msg[1:])
		if err != nil {
			p.log.Error("failed to unmarshal find nodes request", "err", err)
			return nil
		}

		p.log.Trace("received find nodes request", "protocol", p.protocolId, "source", id, "findNodesRequest", findNodesRequest)
		resp, err := p.handleFindNodes(addr, findNodesRequest)
		if err != nil {
			p.log.Error("failed to handle find nodes request", "err", err)
			return nil
		}

		return resp
	case portalwire.FINDCONTENT:
		findContentRequest := &portalwire.FindContent{}
		err := findContentRequest.UnmarshalSSZ(msg[1:])
		if err != nil {
			p.log.Error("failed to unmarshal find content request", "err", err)
			return nil
		}

		p.log.Trace("received find content request", "protocol", p.protocolId, "source", id, "findContentRequest", findContentRequest)
		resp, err := p.handleFindContent(id, addr, findContentRequest)
		if err != nil {
			p.log.Error("failed to handle find content request", "err", err)
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

	enrSeq := p.DiscV5.LocalNode().Seq()
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

	p.log.Trace("Sending pong response", "protocol", p.protocolId, "source", id, "pong", pong)
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

	nodes := p.DiscV5.collectTableNodes(fromAddr.IP, distances, portalFindnodesResultLimit)

	nodesOverhead := 1 + 1 + 4 // msg id + total + container offset
	maxPayloadSize := maxPacketSize - talkRespOverhead - nodesOverhead
	enrOverhead := 4 //per added ENR, 4 bytes offset overhead

	enrs := p.truncateNodes(nodes, maxPayloadSize, enrOverhead)

	nodesMsg := &portalwire.Nodes{
		Total: 1,
		Enrs:  enrs,
	}

	p.log.Trace("Sending nodes response", "protocol", p.protocolId, "source", fromAddr, "nodes", nodesMsg)
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

	contentId := p.storage.ContentId(request.ContentKey)
	if contentId == nil {
		return nil, fmt.Errorf("content not found")
	}

	var content []byte
	content, err = p.storage.Get(request.ContentKey, contentId)
	if err != nil {
		return nil, err
	}

	if content == nil {
		closestNodes := p.findNodesCloseToContent(contentId)
		for i, n := range closestNodes {
			if n.ID() == id {
				closestNodes = append(closestNodes[:i], closestNodes[i+1:]...)
				break
			}
		}

		enrs := p.truncateNodes(closestNodes, maxPayloadSize, enrOverhead)

		enrsMsg := &portalwire.Enrs{
			Enrs: enrs,
		}

		p.log.Trace("Sending enrs content response", "protocol", p.protocolId, "source", addr, "enrs", enrsMsg)
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
		contentMsgBytes := make([]byte, 0, len(content)+1)
		contentMsgBytes = append(contentMsgBytes, portalwire.ContentRawSelector)
		contentMsgBytes = append(contentMsgBytes, content...)

		talkRespBytes := make([]byte, 0, len(contentMsgBytes)+1)
		talkRespBytes = append(talkRespBytes, portalwire.CONTENT)
		talkRespBytes = append(talkRespBytes, contentMsgBytes...)

		return talkRespBytes, nil
	} else {
		connIdGen := utp.NewConnIdGenerator()
		connId := connIdGen.GenCid(id, false)
		connIdSend := connId.SendId()

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), defaultUTPAcceptTimeout)
			var conn *utp.Conn
			conn, err = p.utp.AcceptUTPContext(ctx, connIdSend)
			if err != nil {
				p.log.Error("failed to accept utp connection", "err", err)
				cancel()
				return
			}
			cancel()

			wctx, wcancel := context.WithTimeout(context.Background(), defaultUTPWriteTimeout)
			var n int
			n, err = conn.WriteContext(wctx, content)
			if err != nil {
				p.log.Error("failed to write content to utp connection", "err", err)
				wcancel()
				return
			}
			wcancel()
			p.log.Trace("wrote content size to utp connection", "n", n)
		}()

		idBuffer := make([]byte, 2)
		binary.BigEndian.PutUint16(idBuffer, uint16(connIdSend))
		connIdMsg := &portalwire.ConnectionId{
			Id: idBuffer,
		}

		p.log.Trace("Sending connection id content response", "protocol", p.protocolId, "source", addr, "connId", connIdMsg)
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

func (p *PortalProtocol) Self() *enode.Node {
	return p.DiscV5.LocalNode().Node()
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
		if !containsUint(uint(nd), distances) {
			return nil, errors.New("does not match any requested distance")
		}
	}
	if _, ok := seen[n.ID()]; ok {
		return nil, fmt.Errorf("duplicate record")
	}
	seen[n.ID()] = struct{}{}
	return n, nil
}

func (p *PortalProtocol) ping(node *enode.Node) (uint64, error) {
	enrSeq := p.DiscV5.LocalNode().Seq()
	radiusBytes, err := p.nodeRadius.MarshalSSZ()
	if err != nil {
		return 0, err
	}
	customPayload := &portalwire.PingPongCustomData{
		Radius: radiusBytes,
	}

	customPayloadBytes, err := customPayload.MarshalSSZ()
	if err != nil {
		return 0, err
	}

	pingRequest := &portalwire.Ping{
		EnrSeq:        enrSeq,
		CustomPayload: customPayloadBytes,
	}

	p.log.Trace("Sending ping request", "protocol", p.protocolId, "source", p.Self().ID(), "target", node.ID(), "ping", pingRequest)
	pingRequestBytes, err := pingRequest.MarshalSSZ()
	if err != nil {
		return 0, err
	}

	talkRequestBytes := make([]byte, 0, len(pingRequestBytes)+1)
	talkRequestBytes = append(talkRequestBytes, portalwire.PING)
	talkRequestBytes = append(talkRequestBytes, pingRequestBytes...)

	talkResp, err := p.DiscV5.TalkRequest(node, p.protocolId, talkRequestBytes)

	if err != nil {
		p.replaceNode(node)
	}
	return p.processPong(node, talkResp)
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
	crand.Read(target[:])
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
			nodes.push(wrapNode(n), portalFindnodesResultLimit)
		}
	}
	return nodes.entries, err
}

func (p *PortalProtocol) truncateNodes(nodes []*enode.Node, maxSize int, enrOverhead int) [][]byte {
	res := make([][]byte, 0)
	totalSize := 0
	for _, n := range nodes {
		enrBytes, err := rlp.EncodeToBytes(n.Record())
		if err != nil {
			p.log.Error("failed to encode n", "err", err)
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

func (p *PortalProtocol) findNodesCloseToContent(contentId []byte) []*enode.Node {
	allNodes := p.table.Nodes()
	sort.Slice(allNodes, func(i, j int) bool {
		return enode.LogDist(allNodes[i].ID(), enode.ID(contentId)) < enode.LogDist(allNodes[j].ID(), enode.ID(contentId))
	})

	if len(allNodes) > portalFindnodesResultLimit {
		allNodes = allNodes[:portalFindnodesResultLimit]
	} else {
		allNodes = allNodes[:]
	}

	return allNodes
}
