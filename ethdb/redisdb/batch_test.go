package redisdb

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

type mockClient struct {
	executions      []string
	existsResult    *redis.IntCmd
	keysResult      *redis.StringSliceCmd
	getResult       *redis.StringCmd
	mgetResult      *redis.SliceCmd
	msetResult      *redis.StatusCmd
	delResult       *redis.IntCmd
	setResult       *redis.StatusCmd
	configGetResult *redis.MapStringStringCmd
	scanResult      *redis.ScanCmd
	closeResult     error
}

var _ simpleClient = (*mockClient)(nil)

func (m *mockClient) reset() {
	m.executions = nil
	m.existsResult = new(redis.IntCmd)
	m.getResult = new(redis.StringCmd)
	m.msetResult = new(redis.StatusCmd)
	m.delResult = new(redis.IntCmd)
	m.setResult = new(redis.StatusCmd)
	m.configGetResult = new(redis.MapStringStringCmd)
	m.scanResult = new(redis.ScanCmd)
	m.closeResult = nil
}

func (m *mockClient) Exists(ctx context.Context, keys ...string) *redis.IntCmd {
	m.executions = append(m.executions, "exists")
	return m.existsResult
}

func (m *mockClient) Keys(ctx context.Context, pattern string) *redis.StringSliceCmd {
	m.executions = append(m.executions, "keys")
	return m.keysResult
}

func (m *mockClient) Get(ctx context.Context, key string) *redis.StringCmd {
	m.executions = append(m.executions, "get")
	return m.getResult
}

func (m *mockClient) MGet(ctx context.Context, keys ...string) *redis.SliceCmd {
	m.executions = append(m.executions, "mget")
	return m.mgetResult
}

func (m *mockClient) MSet(ctx context.Context, pairs ...interface{}) *redis.StatusCmd {
	m.executions = append(m.executions, "mset")
	return m.msetResult
}

func (m *mockClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	m.executions = append(m.executions, "del")
	return m.delResult
}

func (m *mockClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	m.executions = append(m.executions, "set")
	return m.setResult
}

func (m *mockClient) ConfigGet(ctx context.Context, parameter string) *redis.MapStringStringCmd {
	m.executions = append(m.executions, "configGet")
	return m.configGetResult
}

func (m *mockClient) Scan(ctx context.Context, cursor uint64, match string, count int64) *redis.ScanCmd {
	m.executions = append(m.executions, "scan")
	return m.scanResult
}

func (m *mockClient) Close() error {
	m.executions = append(m.executions, "close")
	return m.closeResult
}

func TestBatch_Write(t *testing.T) {
	mock := &mockClient{}
	mock.reset()

	db := &Database{client: mock}
	var b *redisBatch = newBatch(db, 1000).(*redisBatch)
	assert.Len(t, mock.executions, 0)
	assert.Len(t, b.operations, 0)

	// Test put without key
	err := b.Put([]byte{}, []byte("value1"))
	assert.NoError(t, err)
	assert.Len(t, mock.executions, 0)
	assert.Len(t, b.operations, 0)

	// Test delete without key
	err = b.Delete([]byte{})
	assert.NoError(t, err)
	assert.Len(t, mock.executions, 0)
	assert.Len(t, b.operations, 0)

	// Insert some data
	err = b.Put([]byte("key1"), []byte("value1"))
	assert.NoError(t, err)
	assert.Len(t, mock.executions, 0)
	assert.Len(t, b.operations, 1)
	assert.False(t, b.operations[0].delete)
	assert.Len(t, b.operations[0].values, 2)
	assert.Equal(t, "key1", b.operations[0].values[0])
	assert.Equal(t, "value1", b.operations[0].values[1])

	err = b.Put([]byte("key2"), []byte("value2"))
	assert.NoError(t, err)
	assert.Len(t, mock.executions, 0)
	assert.Len(t, b.operations, 1)
	assert.False(t, b.operations[0].delete)
	assert.Len(t, b.operations[0].values, 4)
	assert.Equal(t, "key2", b.operations[0].values[2])
	assert.Equal(t, "value2", b.operations[0].values[3])

	err = b.Delete([]byte("key3"))
	assert.NoError(t, err)
	assert.Len(t, mock.executions, 0)
	assert.Len(t, b.operations, 2)
	assert.False(t, b.operations[0].delete)
	assert.True(t, b.operations[1].delete)
	assert.Len(t, b.operations[1].values, 1)
	assert.Equal(t, "key3", b.operations[1].values[0])

	err = b.Delete([]byte("key4"))
	assert.NoError(t, err)
	assert.Len(t, mock.executions, 0)
	assert.Len(t, b.operations, 2)
	assert.False(t, b.operations[0].delete)
	assert.True(t, b.operations[1].delete)
	assert.Len(t, b.operations[1].values, 2)
	assert.Equal(t, "key3", b.operations[1].values[0])
	assert.Equal(t, "key4", b.operations[1].values[1])

	err = b.Put([]byte("key5"), []byte("value5"))
	assert.NoError(t, err)
	assert.Len(t, mock.executions, 0)
	assert.Len(t, b.operations, 3)
	assert.False(t, b.operations[0].delete)
	assert.True(t, b.operations[1].delete)
	assert.False(t, b.operations[2].delete)
	assert.Len(t, b.operations[2].values, 2)
	assert.Equal(t, "key5", b.operations[2].values[0])
	assert.Equal(t, "value5", b.operations[2].values[1])

	// Write the batch
	assert.Len(t, mock.executions, 0)
	err = b.Write()
	assert.NoError(t, err)
	assert.Len(t, mock.executions, 3)
	assert.EqualValues(t, []string{"mset", "del", "mset"}, mock.executions)

	// Write the batch again
	err = b.Write()
	assert.NoError(t, err)
	assert.Len(t, mock.executions, 6)
	assert.EqualValues(t, []string{"mset", "del", "mset", "mset", "del", "mset"}, mock.executions)

	// Check the results
	mock.msetResult.SetErr(errors.New("mset error"))
	err = b.Write()
	assert.Error(t, err)
	assert.Len(t, mock.executions, 7)
	assert.EqualValues(t, []string{"mset", "del", "mset", "mset", "del", "mset", "mset"}, mock.executions)
	mock.msetResult.SetErr(nil)

	mock.delResult.SetErr(errors.New("del error"))
	err = b.Write()
	assert.Error(t, err)
	assert.Len(t, mock.executions, 9)
	assert.EqualValues(t, []string{"mset", "del", "mset", "mset", "del", "mset", "mset", "mset", "del"}, mock.executions)
	mock.delResult.SetErr(nil)

}
