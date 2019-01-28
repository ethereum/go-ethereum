// VulcanizeDB
// Copyright © 2019 Vulcanize

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.

// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package shared

import (
	"errors"
	"strings"
)

// ChainType enum for specifying blockchain
type ChainType int

const (
	UnknownChain ChainType = iota
	Ethereum
	Bitcoin
	Omni
	EthereumClassic
)

func (c ChainType) String() string {
	switch c {
	case Ethereum:
		return "Ethereum"
	case Bitcoin:
		return "Bitcoin"
	case Omni:
		return "Omni"
	case EthereumClassic:
		return "EthereumClassic"
	default:
		return ""
	}
}

func (c ChainType) API() string {
	switch c {
	case Ethereum:
		return "eth"
	case Bitcoin:
		return "btc"
	case Omni:
		return "omni"
	case EthereumClassic:
		return "etc"
	default:
		return ""
	}
}

func NewChainType(name string) (ChainType, error) {
	switch strings.ToLower(name) {
	case "ethereum", "eth":
		return Ethereum, nil
	case "bitcoin", "btc", "xbt":
		return Bitcoin, nil
	case "omni":
		return Omni, nil
	case "classic", "etc":
		return EthereumClassic, nil
	default:
		return UnknownChain, errors.New("invalid name for chain")
	}
}
