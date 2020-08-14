package p2p

import (
	"fmt"
	"github.com/ethereum/go-ethereum/metrics"
	"net"
	"time"

	b "github.com/ethereum/go-ethereum/rlpx" // TODO change name of import
)

// TODO rename maybe? Is this even necessary? Idk.
type Transport struct {
	rlpx *b.Rlpx
}

func newTransport(conn net.Conn) *Transport {
	return &Transport{
		rlpx: b.NewRLPX(conn),
	}
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


