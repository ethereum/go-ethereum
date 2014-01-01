package main

import (
  "fmt"
  "testing"
  "encoding/json"
)

type TestSource struct {
  Inputs map[string]string
  Expectation string
}

func NewTestSource(source string) *TestSource {
  s := &TestSource{}
  err := json.Unmarshal([]byte(source), s)
  if err != nil {
    fmt.Println(err)
  }

  return s
}

type TestRunner struct {
  source *TestSource
}

func NewTestRunner(t *testing.T) *TestRunner {
  return &TestRunner{}
}

func (runner *TestRunner) RunFromString(input string, Cb func(*TestSource)) {
  source := NewTestSource(input)
  Cb(source)
}
