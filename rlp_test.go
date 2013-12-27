package main

import (
  "testing"
  "fmt"
)

func TestEncode(t *testing.T) {
  strRes := "Cdog"
  str := string(Encode("dog"))
  if str != strRes {
    t.Error(fmt.Sprintf("Expected %q, got %q", strRes, str))
  }

  sliceRes := "\u0083CdogCgodCcat"
  slice := string(Encode([]string{"dog", "god", "cat"}))
  if slice != sliceRes {
    t.Error(fmt.Sprintf("Expected %q, got %q", sliceRes, slice))
  }
}
