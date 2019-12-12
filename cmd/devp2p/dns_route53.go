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
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/dnsdisc"
	"gopkg.in/urfave/cli.v1"
)

var (
	route53AccessKeyFlag = cli.StringFlag{
		Name:   "access-key-id",
		Usage:  "AWS Access Key ID",
		EnvVar: "AWS_ACCESS_KEY_ID",
	}
	route53AccessSecretFlag = cli.StringFlag{
		Name:   "access-key-secret",
		Usage:  "AWS Access Key Secret",
		EnvVar: "AWS_SECRET_ACCESS_KEY",
	}
	route53ZoneIDFlag = cli.StringFlag{
		Name:  "zone-id",
		Usage: "Route53 Zone ID",
	}
)

type route53Client struct {
	api    *route53.Route53
	zoneID string
}

// newRoute53Client sets up a Route53 API client from command line flags.
func newRoute53Client(ctx *cli.Context) *route53Client {
	akey := ctx.String(route53AccessKeyFlag.Name)
	asec := ctx.String(route53AccessSecretFlag.Name)
	if akey == "" || asec == "" {
		exit(fmt.Errorf("need Route53 Access Key ID and secret proceed"))
	}
	config := &aws.Config{Credentials: credentials.NewStaticCredentials(akey, asec, "")}
	session, err := session.NewSession(config)
	if err != nil {
		exit(fmt.Errorf("can't create AWS session: %v", err))
	}
	return &route53Client{
		api:    route53.New(session),
		zoneID: ctx.String(route53ZoneIDFlag.Name),
	}
}

// deploy uploads the given tree to Route53.
func (c *route53Client) deploy(name string, t *dnsdisc.Tree) error {
	if err := c.checkZone(name); err != nil {
		return err
	}

	// Compute DNS changes.
	records := t.ToTXT(name)
	changes, err := c.computeChanges(name, records)
	if err != nil {
		return err
	}
	if len(changes) == 0 {
		log.Info("No DNS changes needed")
		return nil
	}

	// Submit change request.
	log.Info(fmt.Sprintf("Submitting %d changes to Route53", len(changes)))
	batch := new(route53.ChangeBatch)
	batch.SetChanges(changes)
	batch.SetComment(fmt.Sprintf("enrtree update of %s at seq %d", name, t.Seq()))
	req := &route53.ChangeResourceRecordSetsInput{HostedZoneId: &c.zoneID, ChangeBatch: batch}
	resp, err := c.api.ChangeResourceRecordSets(req)
	if err != nil {
		return err
	}

	// Wait for the change to be applied.
	log.Info(fmt.Sprintf("Waiting for change request %s", *resp.ChangeInfo.Id))
	wreq := &route53.GetChangeInput{Id: resp.ChangeInfo.Id}
	return c.api.WaitUntilResourceRecordSetsChanged(wreq)
}

// checkZone verifies zone information for the given domain.
func (c *route53Client) checkZone(name string) (err error) {
	if c.zoneID == "" {
		c.zoneID, err = c.findZoneID(name)
	}
	return err
}

// findZoneID searches for the Zone ID containing the given domain.
func (c *route53Client) findZoneID(name string) (string, error) {
	log.Info(fmt.Sprintf("Finding Route53 Zone ID for %s", name))
	var req route53.ListHostedZonesByNameInput
	for {
		resp, err := c.api.ListHostedZonesByName(&req)
		if err != nil {
			return "", err
		}
		for _, zone := range resp.HostedZones {
			if isSubdomain(name, *zone.Name) {
				return *zone.Id, nil
			}
		}
		if !*resp.IsTruncated {
			break
		}
		req.DNSName = resp.NextDNSName
		req.HostedZoneId = resp.NextHostedZoneId
	}
	return "", errors.New("can't find zone ID for " + name)
}

// computeChanges creates DNS changes for the given record.
func (c *route53Client) computeChanges(name string, records map[string]string) ([]*route53.Change, error) {
	// Convert all names to lowercase.
	lrecords := make(map[string]string, len(records))
	for name, r := range records {
		lrecords[strings.ToLower(name)] = r
	}
	records = lrecords

	// Get existing records.
	existing, err := c.collectRecords(name)
	if err != nil {
		return nil, err
	}
	log.Info(fmt.Sprintf("Found %d TXT records", len(existing)))

	var changes []*route53.Change
	for path, val := range records {
		ttl := 1
		if path != name {
			ttl = 2147483647
		}

		prevRecords, exists := existing[path]
		prevValue := combineTXT(prevRecords)
		if !exists {
			// Entry is unknown, push a new one
			log.Info(fmt.Sprintf("Creating %s = %q", path, val))
			changes = append(changes, newTXTChange("CREATE", path, ttl, splitTXT(val)))
		} else if prevValue != val {
			// Entry already exists, only change its content.
			log.Info(fmt.Sprintf("Updating %s from %q to %q", path, prevValue, val))
			changes = append(changes, newTXTChange("UPSERT", path, ttl, splitTXT(val)))
		} else {
			log.Info(fmt.Sprintf("Skipping %s = %q", path, val))
		}
	}

	// Iterate over the old records and delete anything stale.
	for path, values := range existing {
		if _, ok := records[path]; ok {
			continue
		}
		// Stale entry, nuke it.
		log.Info(fmt.Sprintf("Deleting %s = %q", path, combineTXT(values)))
		changes = append(changes, newTXTChange("DELETE", path, 1, values))
	}
	return changes, nil
}

// collectRecords collects all TXT records below the given name.
func (c *route53Client) collectRecords(name string) (map[string][]string, error) {
	log.Info(fmt.Sprintf("Retrieving existing TXT records on %s (%s)", name, c.zoneID))
	var req route53.ListResourceRecordSetsInput
	req.SetHostedZoneId(c.zoneID)
	existing := make(map[string][]string)
	err := c.api.ListResourceRecordSetsPages(&req, func(resp *route53.ListResourceRecordSetsOutput, last bool) bool {
		for _, set := range resp.ResourceRecordSets {
			if !isSubdomain(*set.Name, name) || *set.Type != "TXT" {
				continue
			}
			name := strings.TrimSuffix(*set.Name, ".")
			for _, rec := range set.ResourceRecords {
				existing[name] = append(existing[name], *rec.Value)
			}
		}
		return true
	})
	return existing, err
}

// newTXTChange creates a change to a TXT record.
func newTXTChange(action, name string, ttl int, values []string) *route53.Change {
	var c route53.Change
	var r route53.ResourceRecordSet
	var rrs []*route53.ResourceRecord
	for _, val := range values {
		rr := new(route53.ResourceRecord)
		rr.SetValue(val)
		rrs = append(rrs, rr)
	}
	r.SetType("TXT")
	r.SetName(name)
	r.SetTTL(int64(ttl))
	r.SetResourceRecords(rrs)
	c.SetAction(action)
	c.SetResourceRecordSet(&r)
	return &c
}

// isSubdomain returns true if name is a subdomain of domain.
func isSubdomain(name, domain string) bool {
	domain = strings.TrimSuffix(domain, ".")
	name = strings.TrimSuffix(name, ".")
	return strings.HasSuffix("."+name, "."+domain)
}

// combineTXT concatenates the given quoted strings into a single unquoted string.
func combineTXT(values []string) string {
	result := ""
	for _, v := range values {
		if v[0] == '"' {
			v = v[1 : len(v)-1]
		}
		result += v
	}
	return result
}

// splitTXT splits value into a list of quoted 255-character strings.
func splitTXT(value string) []string {
	var result []string
	for len(value) > 0 {
		rlen := len(value)
		if rlen > 253 {
			rlen = 253
		}
		result = append(result, strconv.Quote(value[:rlen]))
		value = value[rlen:]
	}
	return result
}
