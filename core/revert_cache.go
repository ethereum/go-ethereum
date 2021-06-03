package core

import (
  "github.com/ethereum/go-ethereum/common"
  "github.com/ethereum/go-ethereum/accounts/abi"
  lru "github.com/hashicorp/golang-lru"
)

var (
  revertCache *lru.Cache
)

func CacheRevertReason(h common.Hash, reason []byte) {
  if revertCache == nil { revertCache, _ = lru.New(10000) }
  if reason != nil {
    if reasonString, err := abi.UnpackRevert(reason); err == nil {
      revertCache.Add(h, reasonString)
    }
  }
}

func GetRevertReason(h common.Hash) (string, bool) {
  if revertCache == nil { revertCache, _ = lru.New(10000) }
  if v, ok := revertCache.Get(h); ok {
    return v.(string), true
  }
  return "", false
}
