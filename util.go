package main

import (
  "strconv"
  "crypto/sha256"
  "encoding/hex"
  _"fmt"
  _"math"
  "github.com/obscuren/sha3"
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

func Sha3Bin(data []byte) []byte {
  d := sha3.NewKeccak224()
  d.Reset()
  d.Write(data)

  return d.Sum(nil)
}

// Helper function for comparing slices
func CompareIntSlice(a, b []int) bool {
  if len(a) != len(b) {
    return false
  }
  for i, v := range a {
    if v != b[i] {
      return false
    }
  }
  return true
}

// Returns the amount of nibbles that match each other from 0 ...
func MatchingNibbleLength(a, b []int) int {
  i := 0
  for CompareIntSlice(a[:i+1], b[:i+1]) && i < len(b) {
    i+=1
  }

  //fmt.Println(a, b, i-1)

  return i
}

func Hex(d []byte) string {
  return hex.EncodeToString(d)
}
