package core

import (
  "github.com/ethereum/go-ethereum/common"
  "github.com/ethereum/go-ethereum/accounts/abi"
  lru "github.com/hashicorp/golang-lru"
)

var (
  revertCache *lru.Cache
)

func CacheRevertReason(h, blockHash common.Hash, reason []byte) {
  if revertCache == nil { revertCache, _ = lru.New(10000) }
  if reason != nil {
    key := [64]byte{}
    copy(key[:32], blockHash[:])
    copy(key[32:], h[:])
    if reasonString, err := abi.UnpackRevert(reason); err == nil {
      revertCache.Add(key, reasonString)
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
