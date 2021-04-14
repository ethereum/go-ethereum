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
	"net/url"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/influxdata/influxdb/client"
)

type influx struct {
	client   *client.Client
	database string
}

func (i influx) connect(_url, database, username, password string) (err error) {
	u, err := url.Parse(_url)
	if err != nil {
		log.Warn("Unable to parse InfluxDB", "url", _url, "err", err)
		return
	}
	i.client, err = client.NewClient(client.Config{
		URL:      *u,
		Username: username,
		Password: password,
		Timeout:  10 * time.Second,
	})
	i.database = database
	return err
}

func (i influx) updateNodes(nodes []crawledNode) error {
	var pts []client.Point
	now := time.Now()
	for _, node := range nodes {
		n := node.node
		info := &clientInfo{}
		if node.info == nil {
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
		pts = append(pts, client.Point{
			Measurement: fmt.Sprintf("nodes.%v", n.N.ID()),
			Fields: map[string]interface{}{
				"ClientType":      info.ClientType,
				"ID":              n.N.ID(),
				"PK":              n.N.Pubkey(),
				"SoftwareVersion": info.SoftwareVersion,
				"Capabilities":    info.Capabilities,
				"NetworkID":       info.NetworkID,
				"ForkID":          info.ForkID,
				"Blockheight":     info.Blockheight,
				"TotalDifficulty": info.TotalDifficulty,
				"HeadHash":        info.HeadHash,
				"IP":              n.N.IP(),
				"FirstSeen":       n.FirstResponse,
				"LastSeen":        n.LastResponse,
				"Seq":             n.Seq,
				"Score":           n.Score,
				"ConnType":        connType,
			},
			Time: now,
		})
	}
	bps := client.BatchPoints{
		Points:   pts,
		Database: i.database,
	}
	log.Info("Writing results to influx")
	_, err := i.client.Write(bps)
	return err
}
