package p2p

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/metrics"
	"net"
	"time"

	b "github.com/ethereum/go-ethereum/rlpx" // TODO change name of import
)

type transport interface {
	// The two handshakes.
	doEncHandshake(prv *ecdsa.PrivateKey, dialDest *ecdsa.PublicKey) (*ecdsa.PublicKey, error)
	doProtoHandshake(our *protoHandshake) (*protoHandshake, error)
	// The MsgReadWriter can only be used after the encryption
	// handshake has completed. The code uses conn.id to track this
	// by setting it to a non-nil value after the encryption handshake.
	MsgReadWriter
	// transports must provide Close because we use MsgPipe in some of
	// the tests. Closing the actual network connection doesn't do
	// anything in those tests because MsgPipe doesn't use it.
	close(err error)
}

// TODO rename maybe?
// TODO unexport
type Transport struct {
	rlpx *b.Rlpx
}

func newTransport(conn net.Conn) transport {
	return &Transport{
		rlpx: b.NewRLPX(conn),
	}
}

func (t *Transport) ReadMsg() (msg Msg, err error) {
	t.rlpx.Conn.SetReadDeadline(time.Now().Add(frameReadTimeout))
	rawMsg, err := t.rlpx.Read()
	if err != nil {
		return msg, err
	}
	msg = Msg{
		Code: rawMsg.Code,
		Size: rawMsg.Size,
		Payload: rawMsg.Payload,
		ReceivedAt: rawMsg.ReceivedAt,
		meterSize: rawMsg.Size,
	}
	return msg, err
}

func (t *Transport) WriteMsg(msg Msg) error {
	// compress if snappy enabled
	if t.rlpx.RW.Snappy {
		var err error
		msg.Size, msg.Payload, err = t.rlpx.RW.Compress(msg.Size, msg.Payload)
		if err != nil {
			return err
		}
	}

	msg.meterSize = msg.Size
	if metrics.Enabled && msg.meterCap.Name != "" { // don't meter non-subprotocol messages
		m := fmt.Sprintf("%s/%s/%d/%#02x", egressMeterName, msg.meterCap.Name, msg.meterCap.Version, msg.meterCode)
		metrics.GetOrRegisterMeter(m, nil).Mark(int64(msg.meterSize))
		metrics.GetOrRegisterMeter(m+"/packets", nil).Mark(1)
	}
	// construct raw message for transport
	rawMsg := b.RawRLPXMessage{
		Code: msg.Code,
		Size: msg.Size,
		Payload: msg.Payload,
		// TODO receivedAt?
	}

	t.rlpx.Conn.SetWriteDeadline(time.Now().Add(frameWriteTimeout)) // TODO set timeouts on the conn?
	return t.rlpx.Write(rawMsg)
}

func (t *Transport) close(err error) {
	// Tell the remote end why we're disconnecting if possible.
	if t.rlpx.RW != nil {
		if r, ok := err.(DiscReason); ok && r != DiscNetworkError {
			// rlpx tries to send DiscReason to disconnected peer
			// if the connection is net.Pipe (in-memory simulation)
			// it hangs forever, since net.Pipe does not implement
			// a write deadline. Because of this only try to send
			// the disconnect reason message if there is no error.
			if err := t.rlpx.Conn.SetWriteDeadline(time.Now().Add(discWriteTimeout)); err == nil {
				SendItems(t, discMsg, r)
			}
		}
	}
	t.rlpx.Close()
}


