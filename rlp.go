package main

import (
  _"fmt"
  "bytes"
  "math"
)

func BinaryLength(n uint64) uint64 {
  if n == 0 { return 0 }

  return 1 + BinaryLength(n / 256)
}

func ToBinarySlice(n uint64, length uint64) []uint64 {
  if length == 0 {
    length = BinaryLength(n)
  }

  if n == 0 { return make([]uint64, 1) }

  slice := ToBinarySlice(n / 256, 0)
  slice = append(slice, n % 256)

  return slice
}

func ToBin(n uint64, length uint64) string {
  var buf bytes.Buffer
  for _, val := range ToBinarySlice(n, length) {
    buf.WriteString(string(val))
  }

  return buf.String()
}

func FromBin(data []byte) uint64 {
  if len(data) == 0 { return 0 }

  return FromBin(data[:len(data)-1]) * 256 + uint64(data[len(data)-1])
}

func Decode(data []byte, pos int) {
  char := int(data[pos])
  switch {
  case char < 24:
  }
}

func Encode(object interface{}) []byte {
  var buff bytes.Buffer

  switch t := object.(type) {
  case uint32, uint64:
    var num uint64
    if _num, ok := t.(uint64); ok {
      num = _num
    } else if _num, ok := t.(uint32); ok {
      num = uint64(_num)
    }

    if num >= 0 && num < 24 {
      buff.WriteString(string(num))
    } else if num <= uint64(math.Pow(2, 256)) {
      b := ToBin(num, 0)
      buff.WriteString(string(len(b) + 23) + b)
    } else {
      b := ToBin(num, 0)
      b2 := ToBin(uint64(len(b)), 0)
      buff.WriteString(string(len(b2) + 55) + b2 + b)
    }

  case string:
    if len(t) < 56 {
      buff.WriteString(string(len(t) + 64) + t)
    } else {
      b2 := ToBin(uint64(len(t)), 0)
      buff.WriteString(string(len(b2) + 119) + b2 + t)
    }

  case []byte:
    // Cast the byte slice to a string
    buff.Write(Encode(string(t)))

  case []interface{}, []string:
    // Inline function for writing the slice header
    WriteSliceHeader := func(length int) {
      if length < 56 {
        buff.WriteString(string(length + 128))
      } else {
        b2 := ToBin(uint64(length), 0)
        buff.WriteString(string(len(b2) + 183) + b2)
      }
    }

    // FIXME How can I do this "better"?
    if interSlice, ok := t.([]interface{}); ok {
      WriteSliceHeader(len(interSlice))
      for _, val := range interSlice {
        buff.Write(Encode(val))
      }
    } else if stringSlice, ok := t.([]string); ok {
      WriteSliceHeader(len(stringSlice))
      for _, val := range stringSlice {
        buff.Write(Encode(val))
      }
    }
  }

  return buff.Bytes()
}
