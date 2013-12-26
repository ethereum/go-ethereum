package main

import (
  "testing"
  "fmt"
)

func TestRlpEncode(t *testing.T) {
  strRes := "\x00\x03dog"
  str := RlpEncode("dog")
  if str != strRes {
    t.Error(fmt.Sprintf("Expected %q, got %q", strRes, str))
  }

  sliceRes := "\x01\x00\x03dog\x00\x03god\x00\x03cat"
  slice := RlpEncode([]string{"dog", "god", "cat"})
  if slice != sliceRes {
    t.Error(fmt.Sprintf("Expected %q, got %q", sliceRes, slice))
  }
}
