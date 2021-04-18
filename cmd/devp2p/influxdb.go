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
	_url     *url.URL
	database string
	username string
	password string
}

func NewInflux(_url, database, username, password string) (*influx, error) {
	u, err := url.Parse(_url)
	if err != nil {
		log.Warn("Unable to parse InfluxDB", "url", _url, "err", err)
		return nil, err
	}
	return &influx{
		_url:     u,
		database: database,
		username: username,
		password: password,
	}, nil
}

func (i influx) connect() (*client.Client, error) {
	return client.NewClient(client.Config{
		URL:      *i._url,
		Username: i.username,
		Password: i.password,
		Timeout:  10 * time.Minute,
	})
}

func (i influx) updateNodes(nodes []crawledNode) error {
	var pts []client.Point
	now := time.Now()
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
		pts = append(pts, client.Point{
			Measurement: fmt.Sprintf("nodes.%v", n.N.ID()),
			Fields: map[string]interface{}{
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
			Time: now,
		})
	}
	bps := client.BatchPoints{
		Points:   pts,
		Database: i.database,
	}
	cl, err := i.connect()
	if err != nil {
		return err
	}
	log.Info("Writing results to influx")
	_, err = cl.Write(bps)
	return err
}
