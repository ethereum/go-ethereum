package loggy

import (
	"encoding/json"
	"io/ioutil"

	"github.com/ethereum/go-ethereum/log"
)

type LoggyConfig struct {
	FlagLoggy      bool
	FlagConn       bool
	FlagConnWarn   bool
	FlagPerigee    bool
	FlagForward    bool
	FlagBroadcast  bool
	FlagObserve    bool
	FlagAllTx      bool
	EPOCH_DURATION int64
	LOGS_BASEPATH  string
}

func NewLoggyConfig(path string) (*LoggyConfig, error) {
	cfg := &LoggyConfig{}

	if path == "" {
		cfg = &LoggyConfig{
			FlagLoggy:      false,
			FlagConn:       false,
			FlagConnWarn:   false,
			FlagPerigee:    true,
			FlagForward:    true,
			FlagBroadcast:  false,
			FlagObserve:    false,
			FlagAllTx:      false,
			EPOCH_DURATION: 14400,
			LOGS_BASEPATH:  "/data/logs/geth",
		}
	} else {
		file, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal([]byte(file), cfg)
		if err != nil {
			return nil, err
		}
	}

	out, err := json.Marshal(cfg)
	if err != nil {
		panic(err)
	}

	log.Info("Loggy config: " + string(out))
	return cfg, nil
}
