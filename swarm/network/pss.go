package network

import (
	"fmt"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

type pssPeer struct {
	Peer
}

type PssMsg struct {
	Recipient pssPeer
	Payload   []byte
}

func (pm *PssMsg) String() string {
	return fmt.Sprintf("PssMsg: Recipient: %v", pm.Recipient)
}

func PssMsgHandler(msg interface{}) error {
	glog.V(logger.Detail).Infof("Pss Handled!")
	return nil
}
