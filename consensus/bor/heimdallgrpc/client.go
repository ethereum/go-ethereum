package heimdallgrpc

import (
	"github.com/ethereum/go-ethereum/log"
	proto "github.com/maticnetwork/polyproto/heimdall"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	stateFetchLimit = 50
)

type HeimdallGRPCClient struct {
	conn   *grpc.ClientConn
	client proto.HeimdallClient
}

func NewHeimdallGRPCClient(address string) *HeimdallGRPCClient {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error("Failed to connect to Heimdall gRPC server: %v", err)
		panic(err)
	}

	return &HeimdallGRPCClient{
		conn:   conn,
		client: proto.NewHeimdallClient(conn),
	}
}

func (h *HeimdallGRPCClient) Close() {
	log.Debug("Shutdown detected, Closing Heimdall gRPC client")
	h.conn.Close()
}
