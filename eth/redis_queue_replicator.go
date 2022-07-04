package eth

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/ethereum/go-ethereum/core"

	"github.com/go-redis/redis/v7"
	"github.com/golang/snappy"
)

type RedisQueueReplicator struct {
	rdb         *redis.Client
	qKey        string
	description string
	compBuf     []byte
}

const redisQueueReplicatorCompBufSize = 20 * 1024 * 1024

func NewRedisQueueReplicator(rdbURL *url.URL) (*core.ChainReplicator, error) {
	q := rdbURL.Query()
	topic := q.Get("topic")
	if len(topic) == 0 {
		return nil, errors.New("redis replication target requires 'topic' query-param")
	}
	q.Del("topic")
	rdbURL.RawQuery = q.Encode()

	rdbOpts, err := redis.ParseURL(rdbURL.String())
	if err != nil {
		return nil, err
	}

	backend := &RedisQueueReplicator{
		rdb:         redis.NewClient(rdbOpts),
		qKey:        topic,
		description: fmt.Sprintf("Redis(addr=%s,type=stream,key=%s)", rdbOpts.Addr, topic),
		compBuf:     make([]byte, 0, redisQueueReplicatorCompBufSize),
	}

	return core.NewChainReplicator(backend), nil
}

func (r *RedisQueueReplicator) String() string {
	return r.description
}

func (r *RedisQueueReplicator) Process(ctx context.Context, events []*core.BlockReplicationEvent) (err error) {
	pipe := r.rdb.WithContext(ctx).Pipeline()

	for _, event := range events {
		encodedData := snappy.Encode(nil, event.Data)
		pipe.XAdd(&redis.XAddArgs{
			Stream:       r.qKey,
			MaxLenApprox: 500000,
			Values: map[string]interface{}{
				"hash": event.Hash,
				"data": encodedData,
			},
		})
	}

	_, err = pipe.Exec()
	return
}
