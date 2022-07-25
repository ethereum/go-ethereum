package heimdallgrpc

import (
	"log"

	proto "github.com/maticnetwork/polyproto/heimdall"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	stateFetchLimit = 50
)

type HeimdallGRPCClient struct {
	conn    *grpc.ClientConn
	client  proto.HeimdallClient
	closeCh chan struct{}
}

func NewHeimdallGRPCClient(address string) *HeimdallGRPCClient {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Heimdall gRPC server: %v", err)
	}

	return &HeimdallGRPCClient{
		conn:    conn,
		client:  proto.NewHeimdallClient(conn),
		closeCh: make(chan struct{}),
	}
}

func (h *HeimdallGRPCClient) Close() {
	close(h.closeCh)
	h.conn.Close()
}
