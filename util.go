package main

import (
  "strconv"
  "crypto/sha256"
  "encoding/hex"
)

func Uitoa(i uint32) string {
  return strconv.FormatUint(uint64(i), 10)
}

func Sha256Hex(data []byte) string {
  hash := sha256.Sum256(data)

  return hex.EncodeToString(hash[:])
}

func Sha256Bin(data []byte) []byte {
  hash := sha256.Sum256(data)

  return hash[:]
}
