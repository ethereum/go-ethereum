package p2p

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"sync"
	"time"
)

type Handlers map[string]func() Protocol

type proto struct {
	in              chan Msg
	maxcode, offset MsgCode
	messenger       *messenger
}

func (rw *proto) WriteMsg(msg Msg) error {
	if msg.Code >= rw.maxcode {
		return NewPeerError(InvalidMsgCode, "not handled")
	}
	return rw.messenger.writeMsg(msg)
}

func (rw *proto) ReadMsg() (Msg, error) {
	msg, ok := <-rw.in
	if !ok {
		return msg, io.EOF
	}
	return msg, nil
}

// eofSignal is used to 'lend' the network connection
// to a protocol. when the protocol's read loop has read the
// whole payload, the done channel is closed.
type eofSignal struct {
	wrapped io.Reader
	eof     chan struct{}
}

func (r *eofSignal) Read(buf []byte) (int, error) {
	n, err := r.wrapped.Read(buf)
	if err != nil {
		close(r.eof) // tell messenger that msg has been consumed
	}
	return n, err
}

// messenger represents a message-oriented peer connection.
// It keeps track of the set of protocols understood
// by the remote peer.
type messenger struct {
	peer     *Peer
	handlers Handlers

	// the mutex protects the connection
	// so only one protocol can write at a time.
	writeMu sync.Mutex
	conn    net.Conn
	bufconn *bufio.ReadWriter

	protocolLock sync.RWMutex
	protocols    map[string]*proto
	offsets      map[MsgCode]*proto
	protoWG      sync.WaitGroup

	err   chan error
	pulse chan bool
}

func newMessenger(peer *Peer, conn net.Conn, errchan chan error, handlers Handlers) *messenger {
	return &messenger{
		conn:      conn,
		bufconn:   bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn)),
		peer:      peer,
		handlers:  handlers,
		protocols: make(map[string]*proto),
		err:       errchan,
		pulse:     make(chan bool, 1),
	}
}

func (m *messenger) Start() {
	m.protocols[""] = m.startProto(0, "", &baseProtocol{})
	go m.readLoop()
}

func (m *messenger) Stop() {
	m.conn.Close()
	m.protoWG.Wait()
}

const (
	// maximum amount of time allowed for reading a message
	msgReadTimeout = 5 * time.Second

	// messages smaller than this many bytes will be read at
	// once before passing them to a protocol.
	wholePayloadSize = 64 * 1024
)

func (m *messenger) readLoop() {
	defer m.closeProtocols()
	for {
		m.conn.SetReadDeadline(time.Now().Add(msgReadTimeout))
		msg, err := readMsg(m.bufconn)
		if err != nil {
			m.err <- err
			return
		}
		// send ping to heartbeat channel signalling time of last message
		m.pulse <- true
		proto, err := m.getProto(msg.Code)
		if err != nil {
			m.err <- err
			return
		}
		msg.Code -= proto.offset
		if msg.Size <= wholePayloadSize {
			// optimization: msg is small enough, read all
			// of it and move on to the next message
			buf, err := ioutil.ReadAll(msg.Payload)
			if err != nil {
				m.err <- err
				return
			}
			msg.Payload = bytes.NewReader(buf)
			proto.in <- msg
		} else {
			pr := &eofSignal{msg.Payload, make(chan struct{})}
			msg.Payload = pr
			proto.in <- msg
			<-pr.eof
		}
	}
}

func (m *messenger) closeProtocols() {
	m.protocolLock.RLock()
	for _, p := range m.protocols {
		close(p.in)
	}
	m.protocolLock.RUnlock()
}

func (m *messenger) startProto(offset MsgCode, name string, impl Protocol) *proto {
	proto := &proto{
		in:        make(chan Msg),
		offset:    offset,
		maxcode:   impl.Offset(),
		messenger: m,
	}
	m.protoWG.Add(1)
	go func() {
		if err := impl.Start(m.peer, proto); err != nil && err != io.EOF {
			logger.Errorf("protocol %q error: %v\n", name, err)
			m.err <- err
		}
		m.protoWG.Done()
	}()
	return proto
}

// getProto finds the protocol responsible for handling
// the given message code.
func (m *messenger) getProto(code MsgCode) (*proto, error) {
	m.protocolLock.RLock()
	defer m.protocolLock.RUnlock()
	for _, proto := range m.protocols {
		if code >= proto.offset && code < proto.offset+proto.maxcode {
			return proto, nil
		}
	}
	return nil, NewPeerError(InvalidMsgCode, "%d", code)
}

// setProtocols starts all subprotocols shared with the
// remote peer. the protocols must be sorted alphabetically.
func (m *messenger) setRemoteProtocols(protocols []string) {
	m.protocolLock.Lock()
	defer m.protocolLock.Unlock()
	offset := baseProtocolOffset
	for _, name := range protocols {
		protocolFunc, ok := m.handlers[name]
		if !ok {
			continue // not handled
		}
		inst := protocolFunc()
		m.protocols[name] = m.startProto(offset, name, inst)
		offset += inst.Offset()
	}
}

// writeProtoMsg sends the given message on behalf of the given named protocol.
func (m *messenger) writeProtoMsg(protoName string, msg Msg) error {
	m.protocolLock.RLock()
	proto, ok := m.protocols[protoName]
	m.protocolLock.RUnlock()
	if !ok {
		return fmt.Errorf("protocol %s not handled by peer", protoName)
	}
	if msg.Code >= proto.maxcode {
		return NewPeerError(InvalidMsgCode, "code %x is out of range for protocol %q", msg.Code, protoName)
	}
	msg.Code += proto.offset
	return m.writeMsg(msg)
}

// writeMsg writes a message to the connection.
func (m *messenger) writeMsg(msg Msg) error {
	m.writeMu.Lock()
	defer m.writeMu.Unlock()
	if err := writeMsg(m.bufconn, msg); err != nil {
		return err
	}
	return m.bufconn.Flush()
}
