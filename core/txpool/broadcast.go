package txpool

type Broadcast struct {
	allClient        map[*Client]bool
	broadcastMessage chan []byte
	registerClient   chan *Client
	unregisterClient chan *Client
}

func NewBroadcast() *Broadcast {
	return &Broadcast{
		allClient:        make(map[*Client]bool),
		broadcastMessage: make(chan []byte),
		registerClient:   make(chan *Client),
		unregisterClient: make(chan *Client),
	}
}
func (b *Broadcast) Run() {
	for {
		select {
		case newClient := <-b.registerClient:
			b.allClient[newClient] = true
		case clientData := <-b.unregisterClient:
			_, stateClient := b.allClient[clientData]
			if stateClient {
				delete(b.allClient, clientData)
				close(clientData.sendMessage)
			}
		case messageData := <-b.broadcastMessage:
			for clientData := range b.allClient {
				clientData.sendMessage <- messageData
			}
		}
	}
}
