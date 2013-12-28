package main

import (
  "testing"
  "fmt"
)

func TestEncode(t *testing.T) {
  strRes := "Cdog"
  bytes := Encode("dog")
  str := string(bytes)
  if str != strRes {
    t.Error(fmt.Sprintf("Expected %q, got %q", strRes, str))
  }
  dec,_ := Decode(bytes, 0)
  fmt.Printf("raw: %v encoded: %q == %v\n", dec, str, "dog")

  sliceRes := "\x83CdogCgodCcat"
  strs := []string{"dog", "god", "cat"}
  bytes = Encode(strs)
  slice := string(bytes)
  if slice != sliceRes {
    t.Error(fmt.Sprintf("Expected %q, got %q", sliceRes, slice))
  }

  dec,_ = Decode(bytes, 0)
  fmt.Printf("raw: %v encoded: %q == %v\n", dec, slice, strs)
}

func TestMultiEncode(t *testing.T) {
  inter := []interface{}{
    []interface{}{
      "1","2","3",
    },
    []string{
      "string",
      "string2",
      "\x86A0J1234567890A\x00B20A0\x82F395843F657986",
      "\x86A0J1234567890A\x00B20A0\x8cF395843F657986I335612448F524099H16716881A0H13114947G2039362G1507139H16719697G1048387E65360",
    },
    "test",
  }

  bytes := Encode(inter)
  fmt.Printf("%q\n", bytes)

  dec, _ := Decode(bytes, 0)
  fmt.Println(dec)
}

func BenchmarkEncodeDecode(b *testing.B) {
  for i := 0; i < b.N; i++ {
    bytes := Encode([]string{"dog", "god", "cat"})
    Decode(bytes, 0)
  }
}
