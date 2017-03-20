package network

import (
	"fmt"
	"bytes"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

type Pss struct {
	Overlay
	LocalAddr	[]byte
	C	chan []byte
}

func NewPss(k Overlay, addr []byte) *Pss {
	return &Pss{
		Overlay: k,
		LocalAddr: addr,
		C: make(chan []byte),
	}
}

type PssMsg struct {
	To   []byte
	Data	[]byte
}

func (pm *PssMsg) String() string {
	return fmt.Sprintf("PssMsg: Recipient: %v", pm.To)
}

func (ps *Pss) HandlePssMsg(msg interface{}) error {
	pssmsg := msg.(*PssMsg)
	to := pssmsg.To
	if bytes.Equal(to, ps.LocalAddr) {
		glog.V(logger.Detail).Infof("Pss to us, yay!", to)
		ps.C <- pssmsg.Data
		return nil
	}

	ps.EachLivePeer(to, 255, func(p Peer, po int) bool {
		err := p.Send(pssmsg)
		if err != nil {
			return true
		}
		return false
	})
	
	return nil
}
