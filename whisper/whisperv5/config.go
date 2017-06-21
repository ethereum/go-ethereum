// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package whisperv5

import (
	"gopkg.in/urfave/cli.v1"
)

type Config struct {
	MaxMessageSize     uint32  `toml:",omitempty"`
	MinimumAcceptedPOW float64 `toml:",omitempty"`
}

var DefaultConfig = Config{
	MaxMessageSize:     DefaultMaxMessageSize,
	MinimumAcceptedPOW: DefaultMinimumPoW,
}

var (
	WhisperEnabledFlag = cli.BoolFlag{
		Name:  "shh",
		Usage: "Enable Whisper",
	}
	MaxMessageSizeFlag = cli.IntFlag{
		Name:  "shh.maxmessagesize",
		Usage: "Max message size accepted",
		Value: int(DefaultMaxMessageSize),
	}
	MinPOWFlag = cli.Float64Flag{
		Name:  "shh.pow",
		Usage: "Minimum POW accepted",
		Value: DefaultMinimumPoW,
	}

	Flags = []cli.Flag{WhisperEnabledFlag, MaxMessageSizeFlag, MinPOWFlag}
)
