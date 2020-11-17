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
	"io/ioutil"
	"net/http"
	"regexp"

	"github.com/ethereum/go-ethereum/log"
	"github.com/jedisct1/go-minisign"
	"gopkg.in/urfave/cli.v1"
)

// TODO(@holiman) replace this later on with an actual key (or keys)
const GethPubkey = "RWQkliYstQBOKOdtClfgC3IypIPX6TAmoEi7beZ4gyR3wsaezvqOMWsp"

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
	url := ctx.String(VersionCheckUrlFlag.Name)
	version := ctx.String(VersionCheckVersionFlag.Name)
	log.Info("Checking vulnerabilities", "version", version, "url", url)
	return checkCurrent(url, version)
}

func checkCurrent(url, current string) error {
	var (
		data []byte
		sig  []byte
		err  error
	)
	if data, err = fetch(url); err != nil {
		return fmt.Errorf("could not retrieve data: %w", err)
	}
	if sig, err = fetch(fmt.Sprintf("%v.minisig", url)); err != nil {
		return fmt.Errorf("could not retrieve signature: %w", err)
	}
	if err = verifySignature(GethPubkey, data, sig); err != nil {
		return err
	}
	var vulns []vulnJson
	if err = json.Unmarshal(data, &vulns); err != nil {
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

// fetch makes an HTTP request to the given url and returns the response body
func fetch(url string) ([]byte, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// verifySignature checks that the sigData is a valid signature of the given
// data, for pubkey GethPubkey
func verifySignature(pubkey string, data, sigdata []byte) error {
	pub, err := minisign.NewPublicKey(pubkey)
	if err != nil {
		return err
	}
	sig, err := minisign.DecodeSignature(string(sigdata))
	if err != nil {
		return err
	}
	_, err = pub.Verify(data, sig)
	return err
}
