// Copyright 2016 The go-ethereum Authors
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

package params

import (
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

// DAODrainList is the list of accounts whose full balances will be moved into a
// refund contract at the beginning of the dao-fork block.
var DAODrainList []common.Address

func init() {
	// Parse the list of DAO accounts to drain
	var list []map[string]string
	if err := json.Unmarshal([]byte(daoDrainListJSON), &list); err != nil {
		panic(fmt.Errorf("Failed to parse DAO drain list: %v", err))
	}
	// Collect all the accounts that need draining
	for _, dao := range list {
		DAODrainList = append(DAODrainList, common.HexToAddress(dao["address"]))
		DAODrainList = append(DAODrainList, common.HexToAddress(dao["extraBalanceAccount"]))
	}
}

// daoDrainListJSON is the JSON encoded list of accounts whose full balances will
// be moved into a refund contract at the beginning of the dao-fork block.
const daoDrainListJSON = `
[
   {
      "address":"0x304a554a310c7e546dfe434669c62820b7d83490",
      "balance":"30328a3f333ac2fb5f509",
      "extraBalance":"9184e72a000",
      "extraBalanceAccount":"0x914d1b8b43e92723e64fd0a06f5bdb8dd9b10c79"
   },
   {
      "address":"0xfe24cdd8648121a43a7c86d289be4dd2951ed49f",
      "balance":"ea0b1bdc78f500a43",
      "extraBalance":"0",
      "extraBalanceAccount":"0x17802f43a0137c506ba92291391a8a8f207f487d"
   },
   {
      "address":"0xb136707642a4ea12fb4bae820f03d2562ebff487",
      "balance":"6050bdeb3354b5c98adc3",
      "extraBalance":"0",
      "extraBalanceAccount":"0xdbe9b615a3ae8709af8b93336ce9b477e4ac0940"
   },
   {
      "address":"0xf14c14075d6c4ed84b86798af0956deef67365b5",
      "balance":"1d77844e94c25ba2",
      "extraBalance":"0",
      "extraBalanceAccount":"0xca544e5c4687d109611d0f8f928b53a25af72448"
   },
   {
      "address":"0xaeeb8ff27288bdabc0fa5ebb731b6f409507516c",
      "balance":"2e93a72de4fc5ec0ed",
      "extraBalance":"0",
      "extraBalanceAccount":"0xcbb9d3703e651b0d496cdefb8b92c25aeb2171f7"
   },
   {
      "address":"0xaccc230e8a6e5be9160b8cdf2864dd2a001c28b6",
      "balance":"14d0944eb3be947a8",
      "extraBalance":"0",
      "extraBalanceAccount":"0x2b3455ec7fedf16e646268bf88846bd7a2319bb2"
   },
   {
      "address":"0x4613f3bca5c44ea06337a9e439fbc6d42e501d0a",
      "balance":"275eaa8345ced6523a8",
      "extraBalance":"0",
      "extraBalanceAccount":"0xd343b217de44030afaa275f54d31a9317c7f441e"
   },
   {
      "address":"0x84ef4b2357079cd7a7c69fd7a37cd0609a679106",
      "balance":"4accfbf922fd046baa05",
      "extraBalance":"0",
      "extraBalanceAccount":"0xda2fef9e4a3230988ff17df2165440f37e8b1708"
   },
   {
      "address":"0xf4c64518ea10f995918a454158c6b61407ea345c",
      "balance":"38d275b0ed7862ba4f13",
      "extraBalance":"0",
      "extraBalanceAccount":"0x7602b46df5390e432ef1c307d4f2c9ff6d65cc97"
   },
   {
      "address":"0xbb9bc244d798123fde783fcc1c72d3bb8c189413",
      "balance":"1",
      "extraBalance":"49097c66ae78c50e4d3c",
      "extraBalanceAccount":"0x807640a13483f8ac783c557fcdf27be11ea4ac7a"
   }
]
`
