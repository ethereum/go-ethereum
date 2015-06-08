package comms

type EthereumClient interface {
	Close()
	Send(interface{}) error
	Recv() (interface{}, error)
}
