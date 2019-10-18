// Copyright 2018 The go-ethereum Authors
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
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/console"
	"github.com/ethereum/go-ethereum/p2p/dnsdisc"
	"github.com/ethereum/go-ethereum/p2p/enode"
	cli "gopkg.in/urfave/cli.v1"
)

var (
	dnsCommand = cli.Command{
		Name:  "dns",
		Usage: "DNS Discovery Commands",
		Subcommands: []cli.Command{
			dnsSyncCommand,
			dnsSignCommand,
			dnsTXTCommand,
			dnsCloudflareCommand,
		},
	}
	dnsSyncCommand = cli.Command{
		Name:      "sync",
		Usage:     "Download a DNS discovery tree",
		ArgsUsage: "<url> [ <directory> ]",
		Action:    dnsSync,
		Flags:     []cli.Flag{dnsTimeoutFlag},
	}
	dnsSignCommand = cli.Command{
		Name:      "sign",
		Usage:     "Sign a DNS discovery tree",
		ArgsUsage: "<tree-directory> <key-file>",
		Action:    dnsSign,
		Flags:     []cli.Flag{dnsDomainFlag, dnsSeqFlag},
	}
	dnsTXTCommand = cli.Command{
		Name:      "to-txt",
		Usage:     "Create a DNS TXT records for a discovery tree",
		ArgsUsage: "<tree-directory> <output-file>",
		Action:    dnsToTXT,
	}
	dnsCloudflareCommand = cli.Command{
		Name:      "to-cloudflare",
		Usage:     "Deploy DNS TXT records to cloudflare",
		ArgsUsage: "<tree-directory>",
		Action:    dnsToCloudflare,
		Flags:     []cli.Flag{cloudflareTokenFlag, cloudflareZoneIDFlag},
	}
)

var (
	dnsTimeoutFlag = cli.DurationFlag{
		Name:  "timeout",
		Usage: "Timeout for DNS lookups",
	}
	dnsDomainFlag = cli.StringFlag{
		Name:  "domain",
		Usage: "Domain name of the tree",
	}
	dnsSeqFlag = cli.UintFlag{
		Name:  "seq",
		Usage: "New sequence number of the tree",
	}
)

// dnsSync performs dnsSyncCommand.
func dnsSync(ctx *cli.Context) error {
	var (
		c      = dnsClient(ctx)
		url    = ctx.Args().Get(0)
		outdir = ctx.Args().Get(1)
	)
	domain, _, err := dnsdisc.ParseURL(url)
	if err != nil {
		return err
	}
	if outdir == "" {
		outdir = domain
	}

	t, err := c.SyncTree(url)
	if err != nil {
		return err
	}
	def := treeToDefinition(url, t)
	def.Meta.LastModified = time.Now()
	writeTreeMetadata(outdir, def)
	writeTreeNodes(outdir, def)
	return nil
}

func dnsSign(ctx *cli.Context) error {
	if ctx.NArg() < 2 {
		return fmt.Errorf("need tree definition directory and key file as arguments")
	}
	var (
		defdir  = ctx.Args().Get(0)
		keyfile = ctx.Args().Get(1)
		def     = loadTreeDefinition(defdir)
		domain  = directoryName(defdir)
	)
	if def.Meta.URL != "" {
		d, _, err := dnsdisc.ParseURL(def.Meta.URL)
		if err != nil {
			return fmt.Errorf("invalid 'url' field: %v", err)
		}
		domain = d
	}
	if ctx.IsSet(dnsDomainFlag.Name) {
		domain = ctx.String(dnsDomainFlag.Name)
	}
	if ctx.IsSet(dnsSeqFlag.Name) {
		def.Meta.Seq = ctx.Uint(dnsSeqFlag.Name)
	} else {
		def.Meta.Seq++ // Auto-bump sequence number if not supplied via flag.
	}
	t, err := dnsdisc.MakeTree(def.Meta.Seq, def.Nodes, def.Meta.Links)
	if err != nil {
		return err
	}

	key := loadSigningKey(keyfile)
	url, err := t.Sign(key, domain)
	if err != nil {
		return fmt.Errorf("can't sign: %v", err)
	}

	def = treeToDefinition(url, t)
	def.Meta.LastModified = time.Now()
	writeTreeMetadata(defdir, def)
	return nil
}

func directoryName(dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		exit(err)
	}
	return filepath.Base(abs)
}

// dnsToTXT peforms dnsTXTCommand.
func dnsToTXT(ctx *cli.Context) error {
	if ctx.NArg() < 1 {
		return fmt.Errorf("need tree definition directory as argument")
	}
	output := ctx.Args().Get(1)
	if output == "" {
		output = "-" // default to stdout
	}
	domain, t, err := loadTreeDefinitionForExport(ctx.Args().Get(0))
	if err != nil {
		return err
	}
	writeTXTJSON(output, t.ToTXT(domain))
	return nil
}

// dnsToCloudflare peforms dnsCloudflareCommand.
func dnsToCloudflare(ctx *cli.Context) error {
	if ctx.NArg() < 1 {
		return fmt.Errorf("need tree definition directory as argument")
	}
	domain, t, err := loadTreeDefinitionForExport(ctx.Args().Get(0))
	if err != nil {
		return err
	}
	client := newCloudflareClient(ctx)
	return client.deploy(domain, t)
}

// loadSigningKey loads a private key in Ethereum keystore format.
func loadSigningKey(keyfile string) *ecdsa.PrivateKey {
	keyjson, err := ioutil.ReadFile(keyfile)
	if err != nil {
		exit(fmt.Errorf("failed to read the keyfile at '%s': %v", keyfile, err))
	}
	password, _ := console.Stdin.PromptPassword("Please enter the password for '" + keyfile + "': ")
	key, err := keystore.DecryptKey(keyjson, password)
	if err != nil {
		exit(fmt.Errorf("error decrypting key: %v", err))
	}
	return key.PrivateKey
}

// dnsClient configures the DNS discovery client from command line flags.
func dnsClient(ctx *cli.Context) *dnsdisc.Client {
	var cfg dnsdisc.Config
	if commandHasFlag(ctx, dnsTimeoutFlag) {
		cfg.Timeout = ctx.Duration(dnsTimeoutFlag.Name)
	}
	c, _ := dnsdisc.NewClient(cfg) // cannot fail because no URLs given
	return c
}

// There are two file formats for DNS node trees on disk:
//
// The 'TXT' format is a single JSON file containing DNS TXT records
// as a JSON object where the keys are names and the values are objects
// containing the value of the record.
//
// The 'definition' format is a directory containing two files:
//
//      enrtree-info.json    -- contains sequence number & links to other trees
//      nodes.json           -- contains the nodes as a JSON array.
//
// This format exists because it's convenient to edit. nodes.json can be generated
// in multiple ways: it may be written by a DHT crawler or compiled by a human.

type dnsDefinition struct {
	Meta  dnsMetaJSON
	Nodes []*enode.Node
}

type dnsMetaJSON struct {
	URL          string    `json:"url,omitempty"`
	Seq          uint      `json:"seq"`
	Sig          string    `json:"signature,omitempty"`
	Links        []string  `json:"links"`
	LastModified time.Time `json:"lastModified"`
}

func treeToDefinition(url string, t *dnsdisc.Tree) *dnsDefinition {
	meta := dnsMetaJSON{
		URL:   url,
		Seq:   t.Seq(),
		Sig:   t.Signature(),
		Links: t.Links(),
	}
	if meta.Links == nil {
		meta.Links = []string{}
	}
	return &dnsDefinition{Meta: meta, Nodes: t.Nodes()}
}

// loadTreeDefinition loads a directory in 'definition' format.
func loadTreeDefinition(directory string) *dnsDefinition {
	metaFile, nodesFile := treeDefinitionFiles(directory)
	var def dnsDefinition
	err := common.LoadJSON(metaFile, &def.Meta)
	if err != nil && !os.IsNotExist(err) {
		exit(err)
	}
	if def.Meta.Links == nil {
		def.Meta.Links = []string{}
	}
	// Check link syntax.
	for _, link := range def.Meta.Links {
		if _, _, err := dnsdisc.ParseURL(link); err != nil {
			exit(fmt.Errorf("invalid link %q: %v", link, err))
		}
	}
	// Check/convert nodes.
	nodes := loadNodesJSON(nodesFile)
	if err := nodes.verify(); err != nil {
		exit(err)
	}
	def.Nodes = nodes.nodes()
	return &def
}

// loadTreeDefinitionForExport loads a DNS tree and ensures it is signed.
func loadTreeDefinitionForExport(dir string) (domain string, t *dnsdisc.Tree, err error) {
	metaFile, _ := treeDefinitionFiles(dir)
	def := loadTreeDefinition(dir)
	if def.Meta.URL == "" {
		return "", nil, fmt.Errorf("missing 'url' field in %v", metaFile)
	}
	domain, pubkey, err := dnsdisc.ParseURL(def.Meta.URL)
	if err != nil {
		return "", nil, fmt.Errorf("invalid 'url' field in %v: %v", metaFile, err)
	}
	if t, err = dnsdisc.MakeTree(def.Meta.Seq, def.Nodes, def.Meta.Links); err != nil {
		return "", nil, err
	}
	if err := ensureValidTreeSignature(t, pubkey, def.Meta.Sig); err != nil {
		return "", nil, err
	}
	return domain, t, nil
}

// ensureValidTreeSignature checks that sig is valid for tree and assigns it as the
// tree's signature if valid.
func ensureValidTreeSignature(t *dnsdisc.Tree, pubkey *ecdsa.PublicKey, sig string) error {
	if sig == "" {
		return fmt.Errorf("missing signature, run 'devp2p dns sign' first")
	}
	if err := t.SetSignature(pubkey, sig); err != nil {
		return fmt.Errorf("invalid signature on tree, run 'devp2p dns sign' to update it")
	}
	return nil
}

// writeTreeMetadata writes a DNS node tree metadata file to the given directory.
func writeTreeMetadata(directory string, def *dnsDefinition) {
	metaJSON, err := json.MarshalIndent(&def.Meta, "", jsonIndent)
	if err != nil {
		exit(err)
	}
	if err := os.Mkdir(directory, 0744); err != nil && !os.IsExist(err) {
		exit(err)
	}
	metaFile, _ := treeDefinitionFiles(directory)
	if err := ioutil.WriteFile(metaFile, metaJSON, 0644); err != nil {
		exit(err)
	}
}

func writeTreeNodes(directory string, def *dnsDefinition) {
	ns := make(nodeSet, len(def.Nodes))
	ns.add(def.Nodes...)
	_, nodesFile := treeDefinitionFiles(directory)
	writeNodesJSON(nodesFile, ns)
}

func treeDefinitionFiles(directory string) (string, string) {
	meta := filepath.Join(directory, "enrtree-info.json")
	nodes := filepath.Join(directory, "nodes.json")
	return meta, nodes
}

// writeTXTJSON writes TXT records in JSON format.
func writeTXTJSON(file string, txt map[string]string) {
	txtJSON, err := json.MarshalIndent(txt, "", jsonIndent)
	if err != nil {
		exit(err)
	}
	if file == "-" {
		os.Stdout.Write(txtJSON)
		fmt.Println()
		return
	}
	if err := ioutil.WriteFile(file, txtJSON, 0644); err != nil {
		exit(err)
	}
}
