package main

import (
  "math"
  "bytes"
)

func ToBinary(x int, bytes int) string {
  if bytes == 0 {
    return ""
  } else {
    return ToBinary(int(x / 256), bytes - 1) + string(x % 256)
  }
}

func NumToVarInt(x int) string {
  if x < 253 {
    return string(x)
  } else if x < int(math.Pow(2,16)) {
    return string(253) + ToBinary(x, 2)
  } else if x < int(math.Pow(2,32)) {
    return string(253) + ToBinary(x, 4)
  } else {
    return string(253) + ToBinary(x, 8)
  }
}

func RlpEncode(object interface{}) string {
  if str, ok := object.(string); ok {
    return "\x00" + NumToVarInt(len(str)) + str
  } else if slice, ok := object.([]interface{}); ok {
    var buffer bytes.Buffer
    for _, val := range slice {
      if v, ok := val.(string); ok {
        buffer.WriteString(RlpEncode(v))
      } else {
        buffer.WriteString(RlpEncode(val))
      }
    }

    return "\x01" + RlpEncode(len(buffer.String())) + buffer.String()
  } else if slice, ok := object.([]string); ok {

    // FIXME this isn't dry. Fix this
    var buffer bytes.Buffer
    for _, val := range slice {
      buffer.WriteString(RlpEncode(val))
    }
    return "\x01" + RlpEncode(len(buffer.String())) + buffer.String()
  }

  return ""
}

type RlpSerializer interface {
  MarshalRls() []byte
  UnmarshalRls([]byte)
}

