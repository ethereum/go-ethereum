package chat

import (
	"time"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/swarm/pss"
	"github.com/ethereum/go-ethereum/swarm/network"
)

const (
	ESendFail = iota
)

var (
	chatConnString = map[int]string{
		ESendFail: "Send error",
	}
)

type ChatMsg struct {
	Serial uint64
	Content []byte
	Source string
}

type chatPing struct {
	Created time.Time
	Pong bool
}

type chatAck struct {
	Seen time.Time
	Serial uint64
}

var ChatProtocol = &protocols.Spec{
	Name: "pssChat",
	Version: 1,
	MaxMsgSize: 1024,
	Messages: []interface{}{
		ChatMsg{}, chatPing{}, chatAck{},
	},
}

var ChatTopic = pss.NewTopic(ChatProtocol.Name, int(ChatProtocol.Version))

type ChatConn struct {
	Addr []byte
	E int
}

func (c* ChatConn) Error() string {
	return chatConnString[c.E]
}

type ChatCtrl struct {
	Peer *protocols.Peer
	OutC chan interface{}
	ConnC chan ChatConn
	inC chan *ChatMsg
	oAddr []byte
	pingTX int
	pingRX int
	pingLast int
}

func (self *ChatCtrl) chatHandler(msg interface{}) error {
	Chatmsg, ok := msg.(*ChatMsg)
	if ok {
		if self.inC != nil {
			self.inC <- Chatmsg
		}
	}
	return nil
}

func New(inC chan *ChatMsg, connC chan ChatConn, injectfunc func(*ChatCtrl)) *p2p.Protocol {
//func New(inC chan *ChatMsg, outC chan interface{}, connC chan ChatConn) *p2p.Protocol {
	chatctrl := &ChatCtrl{
		inC: inC,
		ConnC: connC,
	}
	return &p2p.Protocol{
		Name:    ChatProtocol.Name,
		Version: ChatProtocol.Version,
		Length:  3,
		Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
			peerid := p.ID()
			pp := protocols.NewPeer(p, rw, ChatProtocol)
			chatctrl.Peer = pp
			chatctrl.oAddr = network.ToOverlayAddr(peerid[:])
			injectfunc(chatctrl)
			pp.Run(chatctrl.chatHandler)
			return nil
		},
	}
}
