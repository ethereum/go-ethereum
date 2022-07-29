package server

import (
	"fmt"
	"io/ioutil"

	"github.com/BurntSushi/toml"
)

func readLegacyConfig(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	tomlData := string(data)

	if err != nil {
		return nil, fmt.Errorf("failed to read toml config file: %v", err)
	}

	conf := *DefaultConfig()

	if _, err := toml.Decode(tomlData, &conf); err != nil {
		return nil, fmt.Errorf("failed to decode toml config file: %v", err)
	}

	if err := conf.fillBigInt(); err != nil {
		return nil, err
	}

	if err := conf.fillTimeDurations(); err != nil {
		return nil, err
	}

	return &conf, nil
}
