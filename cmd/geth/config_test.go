package main

import (
	"testing"
)

func TestTomlHTTPRpcTimeout(t *testing.T) {
	var cfg gethConfig
	err := loadConfig("testdata/config.toml", &cfg)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Eth.HTTPRpcTimeout.Seconds() != 20 {
		t.Errorf("HTTPRpcTimeout should be 20 seconds")
	}
}
