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
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/cloudflare/cloudflare-go"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/dnsdisc"
	"github.com/urfave/cli/v2"
)

var (
	cloudflareTokenFlag = &cli.StringFlag{
		Name:    "token",
		Usage:   "CloudFlare API token",
		EnvVars: []string{"CLOUDFLARE_API_TOKEN"},
	}
	cloudflareZoneIDFlag = &cli.StringFlag{
		Name:  "zoneid",
		Usage: "CloudFlare Zone ID (optional)",
	}
)

type cloudflareClient struct {
	*cloudflare.API
	zoneID string
}

// newCloudflareClient sets up a CloudFlare API client from command line flags.
func newCloudflareClient(ctx *cli.Context) *cloudflareClient {
	token := ctx.String(cloudflareTokenFlag.Name)
	if token == "" {
		exit(errors.New("need cloudflare API token to proceed"))
	}
	api, err := cloudflare.NewWithAPIToken(token)
	if err != nil {
		exit(fmt.Errorf("can't create Cloudflare client: %v", err))
	}
	return &cloudflareClient{
		API:    api,
		zoneID: ctx.String(cloudflareZoneIDFlag.Name),
	}
}

// deploy uploads the given tree to CloudFlare DNS.
func (c *cloudflareClient) deploy(name string, t *dnsdisc.Tree) error {
	if err := c.checkZone(name); err != nil {
		return err
	}
	records := t.ToTXT(name)
	return c.uploadRecords(name, records)
}

// checkZone verifies permissions on the CloudFlare DNS Zone for name.
func (c *cloudflareClient) checkZone(name string) error {
	if c.zoneID == "" {
		log.Info(fmt.Sprintf("Finding CloudFlare zone ID for %s", name))
		id, err := c.ZoneIDByName(name)
		if err != nil {
			return err
		}
		c.zoneID = id
	}
	log.Info(fmt.Sprintf("Checking Permissions on zone %s", c.zoneID))
	zone, err := c.ZoneDetails(context.Background(), c.zoneID)
	if err != nil {
		return err
	}
	if !strings.HasSuffix(name, "."+zone.Name) {
		return fmt.Errorf("CloudFlare zone name %q does not match name %q to be deployed", zone.Name, name)
	}
	// Necessary permissions for Cloudlare management - Zone:Read, DNS:Read, Zone:Edit, DNS:Edit
	needPerms := map[string]bool{"#zone:edit": false, "#zone:read": false, "#dns_records:read": false, "#dns_records:edit": false}
	for _, perm := range zone.Permissions {
		if _, ok := needPerms[perm]; ok {
			needPerms[perm] = true
		}
	}
	for _, ok := range needPerms {
		if !ok {
			return fmt.Errorf("wrong permissions on zone %s: %v", c.zoneID, needPerms)
		}
	}
	return nil
}

// cloudflareChange is a single mutation of a TXT record.
type cloudflareChange struct {
	action string // "create", "update" or "delete"
	path   string
	value  string               // new content, for create and update
	ttl    int                  // TTL, for create
	old    cloudflare.DNSRecord // existing record, for update and delete
}

// uploadRecords updates the TXT records at a particular subdomain. All non-root records
// will have a TTL of "infinity". Existing records that are neither part of the new tree
// nor of the currently published one are deleted.
func (c *cloudflareClient) uploadRecords(name string, records map[string]string) error {
	log.Info(fmt.Sprintf("Retrieving existing TXT records on %s", name))
	entries, _, err := c.ListDNSRecords(context.Background(), cloudflare.ZoneIdentifier(c.zoneID), cloudflare.ListDNSRecordsParams{Type: "TXT"})
	if err != nil {
		return err
	}
	existing := make(map[string]cloudflare.DNSRecord)
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name, name) {
			continue
		}
		existing[strings.ToLower(entry.Name)] = entry
	}

	log.Info("Updating DNS entries")
	zone := cloudflare.ZoneIdentifier(c.zoneID)
	for _, ch := range computeCloudflareChanges(name, records, existing) {
		switch ch.action {
		case "create":
			log.Debug(fmt.Sprintf("Creating %s = %q", ch.path, ch.value))
			record := cloudflare.CreateDNSRecordParams{Type: "TXT", Name: ch.path, Content: ch.value, TTL: ch.ttl}
			_, err = c.CreateDNSRecord(context.Background(), zone, record)
		case "update":
			log.Info(fmt.Sprintf("Updating %s from %q to %q", ch.path, ch.old.Content, ch.value))
			record := cloudflare.UpdateDNSRecordParams{
				Type:     ch.old.Type,
				Name:     ch.old.Name,
				Content:  ch.value,
				Data:     ch.old.Data,
				ID:       ch.old.ID,
				Priority: ch.old.Priority,
				TTL:      ch.old.TTL,
				Proxied:  ch.old.Proxied,
				Tags:     ch.old.Tags,
			}
			_, err = c.UpdateDNSRecord(context.Background(), zone, record)
		case "delete":
			log.Debug(fmt.Sprintf("Deleting %s = %q", ch.path, ch.old.Content))
			err = c.DeleteDNSRecord(context.Background(), zone, ch.old.ID)
		}
		if err != nil {
			return fmt.Errorf("failed to %s %s: %v", ch.action, ch.path, err)
		}
	}
	return nil
}

// computeCloudflareChanges computes the record mutations needed to publish 'records'
// at 'name', given the 'existing' records (keyed by lowercase name).
//
// Changes are ordered such that the tree remains resolvable throughout the update:
// new leaf records are created first, the root is changed after all records it
// references exist, and stale records are deleted last. Records of the tree that is
// currently published are retained, so clients holding a cached copy of the previous
// root can still walk it. They are deleted by the next deployment.
func computeCloudflareChanges(name string, records map[string]string, existing map[string]cloudflare.DNSRecord) []cloudflareChange {
	// Convert all names to lowercase.
	lrecords := make(map[string]string, len(records))
	for name, r := range records {
		lrecords[strings.ToLower(name)] = r
	}
	records = lrecords
	root := strings.ToLower(name)

	var (
		changes []cloudflareChange
		skipped int
	)
	for path, val := range records {
		old, exists := existing[path]
		if !exists {
			ttl := rootTTL
			if path != root {
				ttl = treeNodeTTLCloudflare // Max TTL permitted by Cloudflare
			}
			changes = append(changes, cloudflareChange{action: "create", path: path, value: val, ttl: ttl})
		} else if old.Content != val {
			changes = append(changes, cloudflareChange{action: "update", path: path, value: val, old: old})
		} else {
			skipped++
			log.Debug(fmt.Sprintf("Skipping %s = %q", path, val))
		}
	}

	// Delete stale entries, but retain the currently published tree.
	existingContent := make(map[string]string, len(existing))
	for path, entry := range existing {
		existingContent[path] = entry.Content
	}
	previous := previousTreeRecords(name, existingContent)
	retained := 0
	for path, entry := range existing {
		if _, ok := records[path]; ok {
			continue
		}
		if previous[path] {
			retained++
			continue
		}
		changes = append(changes, cloudflareChange{action: "delete", path: path, old: entry})
	}

	// Order: leaf creates -> leaf updates -> root -> deletes.
	rank := func(ch cloudflareChange) int {
		switch {
		case ch.action == "delete":
			return 3
		case ch.path == root:
			return 2
		case ch.action == "update":
			return 1
		default:
			return 0
		}
	}
	slices.SortFunc(changes, func(a, b cloudflareChange) int {
		if c := cmp.Compare(rank(a), rank(b)); c != 0 {
			return c
		}
		return strings.Compare(a.path, b.path)
	})

	counts := make(map[string]int)
	for _, ch := range changes {
		counts[ch.action]++
	}
	log.Info("Computed DNS changes", "new", counts["create"], "updated", counts["update"], "untouched", skipped, "deleted", counts["delete"], "retained", retained)
	return changes
}
