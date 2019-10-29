// Copyright 2019 The go-ethereum Authors
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
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

const jsonIndent = "    "

// nodeSet is the nodes.json file format. It holds a set of node records
// as a JSON object.
type nodeSet map[enode.ID]nodeJSON

type nodeJSON struct {
	Seq uint64      `json:"seq"`
	N   *enode.Node `json:"record"`

	// The score tracks how many liveness checks were performed. It is incremented by one
	// every time the node passes a check, and halved every time it doesn't.
	Score int `json:"score,omitempty"`
	// These two track the time of last successful contact.
	FirstResponse time.Time `json:"firstResponse,omitempty"`
	LastResponse  time.Time `json:"lastResponse,omitempty"`
	// This one tracks the time of our last attempt to contact the node.
	LastCheck time.Time `json:"lastCheck,omitempty"`
}

func loadNodesJSON(file string) nodeSet {
	var nodes nodeSet
	if err := common.LoadJSON(file, &nodes); err != nil {
		exit(err)
	}
	return nodes
}

func writeNodesJSON(file string, nodes nodeSet) {
	nodesJSON, err := json.MarshalIndent(nodes, "", jsonIndent)
	if err != nil {
		exit(err)
	}
	if file == "-" {
		os.Stdout.Write(nodesJSON)
		return
	}
	if err := ioutil.WriteFile(file, nodesJSON, 0644); err != nil {
		exit(err)
	}
}

func (ns nodeSet) nodes() []*enode.Node {
	result := make([]*enode.Node, 0, len(ns))
	for _, n := range ns {
		result = append(result, n.N)
	}
	// Sort by ID.
	sort.Slice(result, func(i, j int) bool {
		return bytes.Compare(result[i].ID().Bytes(), result[j].ID().Bytes()) < 0
	})
	return result
}

func (ns nodeSet) add(nodes ...*enode.Node) {
	for _, n := range nodes {
		ns[n.ID()] = nodeJSON{Seq: n.Seq(), N: n}
	}
}

func (ns nodeSet) verify() error {
	for id, n := range ns {
		if n.N.ID() != id {
			return fmt.Errorf("invalid node %v: ID does not match ID %v in record", id, n.N.ID())
		}
		if n.N.Seq() != n.Seq {
			return fmt.Errorf("invalid node %v: 'seq' does not match seq %d from record", id, n.N.Seq())
		}
	}
	return nil
}
