package eth

import (
	"fmt"
	"net/url"

	"github.com/ethereum/go-ethereum/core"
)

func CreateReplicator(target string) (*core.ChainReplicator, error) {
	targetURL, err := url.Parse(target)
	if err != nil {
		return nil, err
	}

	switch targetURL.Scheme {
	case "redis", "rediss":
		return NewRedisQueueReplicator(targetURL)
	case "file":
		return NewRLPFileSetReplicator(targetURL)
	default:
		return nil, fmt.Errorf("unknown replication-target URI scheme '%s'", targetURL.Scheme)
	}
}
