package heimdallgrpc

import (
	"time"

	"github.com/ethereum/go-ethereum/log"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	proto "github.com/maticnetwork/polyproto/heimdall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
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
	opts := []grpc_retry.CallOption{
		grpc_retry.WithMax(10000),
		grpc_retry.WithBackoff(grpc_retry.BackoffLinear(5 * time.Second)),
		grpc_retry.WithCodes(codes.Internal, codes.Unavailable, codes.Aborted, codes.NotFound),
	}

	conn, err := grpc.Dial(address,
		grpc.WithStreamInterceptor(grpc_retry.StreamClientInterceptor(opts...)),
		grpc.WithUnaryInterceptor(grpc_retry.UnaryClientInterceptor(opts...)),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Crit("Failed to connect to Heimdall gRPC", "error", err)
	}

	log.Info("Connected to Heimdall gRPC server", "address", address)

	return &HeimdallGRPCClient{
		conn:   conn,
		client: proto.NewHeimdallClient(conn),
	}
}

func (h *HeimdallGRPCClient) Close() {
	log.Debug("Shutdown detected, Closing Heimdall gRPC client")
	h.conn.Close()
}
