package kcpxfer

import (
	"crypto/sha256"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/xtaci/kcp-go"
)

const (
	ecParityShards = 3
	ecDataShards   = 10
	minPacketSize  = len(ID{})
)

// Protocol messages.
type (
	startRequest struct {
		Size uint64
		Hash [32]byte
		// Key [16]byte

		xferID   ID
		fromNode enode.ID
		fromAddr *net.UDPAddr
		accept   chan *xferState
		isClient bool
	}

	startResponse struct {
		Accept bool
		// Key [16]byte
	}
)

// ID is a transfer identifier. IDs are assigned based on the hash of the
// transferred item and the node it is being sent to.
type ID [16]byte

func computeID(contentHash [32]byte, fromNode enode.ID) (id ID) {
	h := sha256.New()
	h.Write(contentHash[:])
	h.Write(fromNode[:])
	copy(id[:], h.Sum(nil))
	return id
}

type Server struct {
	disc      *discover.UDPv5
	conn      *net.UDPConn
	packet    <-chan discover.ReadPacket
	start     chan *startRequest
	serveFunc func(ConnRequest) error
}

type xferState struct {
	id      ID
	conn    *kcpConn
	session *kcp.UDPSession
}

func NewServer(disc *discover.UDPv5, conn *net.UDPConn, pch <-chan discover.ReadPacket) {
	s := &Server{
		disc:   disc,
		conn:   conn,
		packet: pch,
		start:  make(chan *startRequest),
	}
	go s.loop()
	s.disc.RegisterTalkHandler("wrm", s.handleTalk)
}

// Transfer creates an outgoing transfer to the given node.
func (s *Server) Dial(n *enode.Node, contentHash [32]byte, size int64) (net.Conn, error) {
	var addr net.UDPAddr
	if n.IP() == nil && n.UDP() == 0 {
		return nil, fmt.Errorf("destination node has no UDP endpoint")
	}
	addr.IP = n.IP()
	addr.Port = n.UDP()
	req := startRequest{
		Hash:     contentHash,
		Size:     uint64(size),
		xferID:   computeID(contentHash, n.ID()),
		fromNode: n.ID(),
		fromAddr: &addr,
		accept:   make(chan *xferState),
	}
	s.start <- &req
	xfer := <-req.accept
	if xfer == nil {
		return nil, fmt.Errorf("failed to start session")
	}
	return xfer.session, nil
}

func (s *Server) handleTalk(node enode.ID, addr *net.UDPAddr, data []byte) []byte {
	var req startRequest
	err := rlp.DecodeBytes(data, &req)
	if err != nil {
		log.Error("Invalid xfer start request", "id", node, "addr", addr, "err", err)
		return []byte{}
	}
	req.xferID = computeID(req.Hash, node)
	req.fromNode = node
	req.fromAddr = addr
	req.isClient = true

	s.start <- &req
	xfer := <-req.accept

	var resp []byte
	if xfer != nil {
		resp, _ = rlp.EncodeToBytes(&startResponse{Accept: true})
	} else {
		resp, _ = rlp.EncodeToBytes(&startResponse{Accept: false})
	}
	return resp
}

func (s *Server) loop() {
	xfers := make(map[ID]*xferState)

	for {
		select {
		case pkt := <-s.packet:
			if len(pkt.Data) > minPacketSize {
				var id ID
				copy(id[:], pkt.Data)
				xfer := xfers[id]
				if xfer != nil {
					xfer.conn.enqueue(pkt.Data[len(id):])
				}
			}

		case req := <-s.start:
			xfer := s.newState(req)
			if req.isClient {
				err := s.serve(req, xfer)
				if err != nil {
					xfer.close()
					req.accept <- nil
					continue
				}
			}
			if xfer != nil {
				xfers[req.xferID] = xfer
			}
			req.accept <- xfer
		}
	}
}

func (s *Server) serve(req *startRequest) *xferState {
	s.serveFunc()
}

// newState creates a new transfer state.
func (s *Server) newState(req *startRequest) *xferState {
	conn := newKCPConn(req.fromAddr, s.conn)
	session, err := kcp.NewConn3(0, req.fromAddr, nil, ecDataShards, ecParityShards, conn)
	if err != nil {
		log.Error("Could not establish kcp session", "err", err)
		return nil
	}
	defer session.Close()

	return &xferState{
		id:      req.xferID,
		conn:    conn,
		session: session,
	}
}

// kcpConn implements net.PacketConn for use by KCP.
type kcpConn struct {
	id  ID
	out net.PacketConn

	mu      sync.Mutex
	flag    *sync.Cond
	inqueue [][]byte
	remote  *net.UDPAddr
}

func newKCPConn(remote *net.UDPAddr, id ID, out net.PacketConn) *kcpConn {
	o := &kcpConn{out: out, remote: remote}
	o.flag = sync.NewCond(&o.mu)
	return o
}

// enqueue adds a packet to the queue.
func (o *kcpConn) enqueue(p []byte) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.inqueue = append(o.inqueue, p)
	o.flag.Broadcast()
}

// ReadFrom delivers a single packet from o.inqueue into the buffer p.
// If a packet does not fit into the buffer, the remaining bytes of the packet
// are discarded.
func (o *kcpConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	o.mu.Lock()
	for len(o.inqueue) == 0 {
		o.flag.Wait()
	}
	defer o.mu.Unlock()

	// Move packet data into p.
	n = copy(p, o.inqueue[0])

	// Delete the packet from inqueue.
	copy(o.inqueue, o.inqueue[1:])
	o.inqueue = o.inqueue[:len(o.inqueue)-1]

	// log.Info("KCP read", "buf", len(p), "n", n, "remaining-in-q", len(o.inqueue))
	// kcpStatsDump(kcp.DefaultSnmp)
	return n, o.remote, nil
}

// WriteTo just writes to the underlying connection.
func (o *kcpConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	n, err = o.out.WriteTo(p, addr)
	// log.Info("KCP write", "buf", len(p), "n", n, "err", err)
	return n, err
}

func (o *kcpConn) LocalAddr() net.Addr                { panic("not implemented") }
func (o *kcpConn) Close() error                       { return nil }
func (o *kcpConn) SetDeadline(t time.Time) error      { return nil }
func (o *kcpConn) SetReadDeadline(t time.Time) error  { return nil }
func (o *kcpConn) SetWriteDeadline(t time.Time) error { return nil }
