package core

import (
  "fmt"
  "github.com/ethereum/go-ethereum/common"
  "github.com/ethereum/go-ethereum/common/hexutil"
  "github.com/ethereum/go-ethereum/core/types"
  "github.com/ethereum/go-ethereum/event"
  "github.com/ethereum/go-ethereum/accounts/abi"
  lru "github.com/hashicorp/golang-lru"
)

var (
  revertCache *lru.Cache
  reorgFeed event.Feed
)

func CacheRevertReason(h, blockHash common.Hash, reason []byte) {
  if revertCache == nil { revertCache, _ = lru.New(10000) }
  if reason != nil {
    key := [64]byte{}
    copy(key[:32], blockHash[:])
    copy(key[32:], h[:])
    if reasonString, err := abi.UnpackRevert(reason); err == nil {
      revertCache.Add(key, reasonString)
    } else {
      revertCache.Add(key, fmt.Sprintf("%#x", reason))
    }
  }
}

func GetRevertReason(h, blockHash common.Hash) (string, bool) {
  if revertCache == nil { revertCache, _ = lru.New(10000) }
  key := [64]byte{}
  copy(key[:32], blockHash[:])
  copy(key[32:], h[:])
  if v, ok := revertCache.Get(key); ok {
    return v.(string), true
  }
  return "", false
}
type Reorg struct {
	Common  common.Hash    `json:"common"`
	Number  hexutil.Uint64 `json:"number"`
	Removed []common.Hash  `json:"removed"`
	Added   []common.Hash  `json:"added"`
}

func sendReorg(commonAncestor *types.Block, removed, added types.Blocks) {
	reorg := &Reorg{
		Common: commonAncestor.Hash(),
		Number: hexutil.Uint64(commonAncestor.NumberU64()),
		Removed: make([]common.Hash, len(removed)),
		Added: make([]common.Hash, len(added)),
	}
	for i, block := range removed {
		reorg.Removed[i] = block.Hash()
	}
	for i, block := range added {
		reorg.Added[i] = block.Hash()
	}
	reorgFeed.Send(reorg)
}

func SubscribeReorgs(ch chan<- *Reorg) event.Subscription {
	return reorgFeed.Subscribe(ch)
}
