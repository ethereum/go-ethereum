// Copyright 2021 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enr"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

type influx struct {
	_url     string
	org      string
	database string
	token    string
}

func NewInflux(_url, database, org, token string) (*influx, error) {
	return &influx{
		_url:     _url,
		org:      org,
		database: database,
		token:    token,
	}, nil
}

func (i influx) connect() influxdb2.Client {
	return influxdb2.NewClient(i._url, i.token)
}

func (i influx) updateNodes(nodes []crawledNode) error {
	now := time.Now()
	cl := i.connect()
	defer cl.Close()
	writer := cl.WriteAPI(i.org, i.database)
	for _, node := range nodes {
		n := node.node
		info := &clientInfo{}
		if node.info != nil {
			info = node.info
		}
		connType := ""
		var portUDP enr.UDP
		if n.N.Load(&portUDP) == nil {
			connType = "UDP"
		}
		var portTCP enr.TCP
		if n.N.Load(&portTCP) == nil {
			connType = "TCP"
		}
		var caps string
		for _, c := range info.Capabilities {
			caps = fmt.Sprintf("%v, %v", caps, c.String())
		}
		var pk string
		if n.N.Pubkey() != nil {
			pk = fmt.Sprintf("X: %v, Y: %v", n.N.Pubkey().X.String(), n.N.Pubkey().Y.String())
		}
		fid := fmt.Sprintf("Hash: %v, NExt %v", info.ForkID.Hash, info.ForkID.Next)
		point := influxdb2.NewPoint(
			fmt.Sprintf("nodes.%v", n.N.ID()),
			nil,
			map[string]interface{}{
				"ClientType":      info.ClientType,
				"ID":              n.N.ID().GoString(),
				"PK":              pk,
				"SoftwareVersion": info.SoftwareVersion,
				"Capabilities":    caps,
				"NetworkID":       info.NetworkID,
				"ForkID":          fid,
				"Blockheight":     info.Blockheight,
				"TotalDifficulty": info.TotalDifficulty.String(),
				"HeadHash":        info.HeadHash.String(),
				"IP":              n.N.IP().String(),
				"FirstSeen":       n.FirstResponse.String(),
				"LastSeen":        n.LastResponse.String(),
				"Seq":             n.Seq,
				"Score":           n.Score,
				"ConnType":        connType,
			},
			now,
		)
		writer.WritePoint(point)
	}

	writer.Flush()
	select {
	case err := <-writer.Errors():
		return err
	default:
		return nil
	}
}
