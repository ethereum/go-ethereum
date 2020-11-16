// Copyright 2020 The go-ethereum Authors
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
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"gopkg.in/urfave/cli.v1"
	"io/ioutil"
	"net/http"
	"regexp"
	"runtime"
)

const versionFeed = "https://geth.ethereum.org/docs/vulnerabilities/vulnerabilities.json"

type vulnJson struct {
	Name        string
	Uid         string
	Summary     string
	Description string
	Links       []string
	Introduced  string
	Fixed       string
	Published   string
	Severity    string
	Check       string
}

func versionCheck(ctx *cli.Context) error {
	args := ctx.Args()
	version := fmt.Sprintf("Geth/v%v/%v-%v/%v",
		params.VersionWithCommit(gitCommit, gitDate),
		runtime.GOOS,
		runtime.GOARCH,
		runtime.Version())
	if len(args) > 0 {
		// Explicit version string given
		version = args[0]
	}
	log.Info("Checking vulnerabilities", "version", version)
	return checkCurrent(version)
}

func checkCurrent(current string) error {
	res, err := http.Get(versionFeed)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	var vulns []vulnJson
	if err = json.Unmarshal(body, &vulns); err != nil {
		return err
	}
	for _, vuln := range vulns {
		r, err := regexp.Compile(vuln.Check)
		if err != nil {
			return err
		}
		if r.MatchString(current) {
			fmt.Printf("## Vulnerable to %v (%v)\n\n", vuln.Uid, vuln.Name)
			fmt.Printf("Severity: %v\n", vuln.Severity)
			fmt.Printf("Summary : %v\n", vuln.Summary)
			fmt.Printf("Fixed in: %v\n", vuln.Fixed)
			if len(vuln.Links) > 0 {
				fmt.Printf("References:\n")
				for _, ref := range vuln.Links {
					fmt.Printf("\t- %v\n", ref)
				}
			}
			fmt.Println()
		}
	}
	return nil
}
