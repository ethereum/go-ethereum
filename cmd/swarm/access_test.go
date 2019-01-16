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
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/ioutil"
	gorand "math/rand"
	"net/http"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm/api"
	swarmapi "github.com/ethereum/go-ethereum/swarm/api/client"
	"github.com/ethereum/go-ethereum/swarm/testutil"
	"golang.org/x/crypto/sha3"
)

const (
	hashRegexp = `[a-f\d]{128}`
	data       = "notsorandomdata"
)

var DefaultCurve = crypto.S256()

func TestACT(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}

	initCluster(t)

	cases := []struct {
		name string
		f    func(t *testing.T)
	}{
		{"Password", testPassword},
		{"PK", testPK},
		{"ACTWithoutBogus", testACTWithoutBogus},
		{"ACTWithBogus", testACTWithBogus},
	}

	for _, tc := range cases {
		t.Run(tc.name, tc.f)
	}
}

// testPassword tests for the correct creation of an ACT manifest protected by a password.
// The test creates bogus content, uploads it encrypted, then creates the wrapping manifest with the Access entry
// The parties participating - node (publisher), uploads to second node then disappears. Content which was uploaded
// is then fetched through 2nd node. since the tested code is not key-aware - we can just
// fetch from the 2nd node using HTTP BasicAuth
func testPassword(t *testing.T) {
	dataFilename := testutil.TempFileWithContent(t, data)
	defer os.RemoveAll(dataFilename)

	// upload the file with 'swarm up' and expect a hash
	up := runSwarm(t,
		"--bzzapi",
		cluster.Nodes[0].URL,
		"up",
		"--encrypt",
		dataFilename)
	_, matches := up.ExpectRegexp(hashRegexp)
	up.ExpectExit()

	if len(matches) < 1 {
		t.Fatal("no matches found")
	}

	ref := matches[0]
	tmp, err := ioutil.TempDir("", "swarm-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	password := "smth"
	passwordFilename := testutil.TempFileWithContent(t, "smth")
	defer os.RemoveAll(passwordFilename)

	up = runSwarm(t,
		"access",
		"new",
		"pass",
		"--dry-run",
		"--password",
		passwordFilename,
		ref,
	)

	_, matches = up.ExpectRegexp(".+")
	up.ExpectExit()

	if len(matches) == 0 {
		t.Fatalf("stdout not matched")
	}

	var m api.Manifest

	err = json.Unmarshal([]byte(matches[0]), &m)
	if err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}

	if len(m.Entries) != 1 {
		t.Fatalf("expected one manifest entry, got %v", len(m.Entries))
	}

	e := m.Entries[0]

	ct := "application/bzz-manifest+json"
	if e.ContentType != ct {
		t.Errorf("expected %q content type, got %q", ct, e.ContentType)
	}

	if e.Access == nil {
		t.Fatal("manifest access is nil")
	}

	a := e.Access

	if a.Type != "pass" {
		t.Errorf(`got access type %q, expected "pass"`, a.Type)
	}
	if len(a.Salt) < 32 {
		t.Errorf(`got salt with length %v, expected not less the 32 bytes`, len(a.Salt))
	}
	if a.KdfParams == nil {
		t.Fatal("manifest access kdf params is nil")
	}
	if a.Publisher != "" {
		t.Fatal("should be empty")
	}

	client := swarmapi.NewClient(cluster.Nodes[0].URL)

	hash, err := client.UploadManifest(&m, false)
	if err != nil {
		t.Fatal(err)
	}

	url := cluster.Nodes[0].URL + "/" + "bzz:/" + hash

	httpClient := &http.Client{}
	response, err := httpClient.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatal("should be a 401")
	}
	authHeader := response.Header.Get("WWW-Authenticate")
	if authHeader == "" {
		t.Fatal("should be something here")
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.SetBasicAuth("", password)

	response, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Errorf("expected status %v, got %v", http.StatusOK, response.StatusCode)
	}
	d, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(d) != data {
		t.Errorf("expected decrypted data %q, got %q", data, string(d))
	}

	wrongPasswordFilename := testutil.TempFileWithContent(t, "just wr0ng")
	defer os.RemoveAll(wrongPasswordFilename)

	//download file with 'swarm down' with wrong password
	up = runSwarm(t,
		"--bzzapi",
		cluster.Nodes[0].URL,
		"down",
		"bzz:/"+hash,
		tmp,
		"--password",
		wrongPasswordFilename)

	_, matches = up.ExpectRegexp("unauthorized")
	if len(matches) != 1 && matches[0] != "unauthorized" {
		t.Fatal(`"unauthorized" not found in output"`)
	}
	up.ExpectExit()
}

// testPK tests for the correct creation of an ACT manifest between two parties (publisher and grantee).
// The test creates bogus content, uploads it encrypted, then creates the wrapping manifest with the Access entry
// The parties participating - node (publisher), uploads to second node (which is also the grantee) then disappears.
// Content which was uploaded is then fetched through the grantee's http proxy. Since the tested code is private-key aware,
// the test will fail if the proxy's given private key is not granted on the ACT.
func testPK(t *testing.T) {
	dataFilename := testutil.TempFileWithContent(t, data)
	defer os.RemoveAll(dataFilename)

	// upload the file with 'swarm up' and expect a hash
	up := runSwarm(t,
		"--bzzapi",
		cluster.Nodes[0].URL,
		"up",
		"--encrypt",
		dataFilename)
	_, matches := up.ExpectRegexp(hashRegexp)
	up.ExpectExit()

	if len(matches) < 1 {
		t.Fatal("no matches found")
	}

	ref := matches[0]
	pk := cluster.Nodes[0].PrivateKey
	granteePubKey := crypto.CompressPubkey(&pk.PublicKey)

	publisherDir, err := ioutil.TempDir("", "swarm-account-dir-temp")
	if err != nil {
		t.Fatal(err)
	}

	passwordFilename := testutil.TempFileWithContent(t, testPassphrase)
	defer os.RemoveAll(passwordFilename)

	_, publisherAccount := getTestAccount(t, publisherDir)
	up = runSwarm(t,
		"--bzzaccount",
		publisherAccount.Address.String(),
		"--password",
		passwordFilename,
		"--datadir",
		publisherDir,
		"--bzzapi",
		cluster.Nodes[0].URL,
		"access",
		"new",
		"pk",
		"--dry-run",
		"--grant-key",
		hex.EncodeToString(granteePubKey),
		ref,
	)

	_, matches = up.ExpectRegexp(".+")
	up.ExpectExit()

	if len(matches) == 0 {
		t.Fatalf("stdout not matched")
	}

	//get the public key from the publisher directory
	publicKeyFromDataDir := runSwarm(t,
		"--bzzaccount",
		publisherAccount.Address.String(),
		"--password",
		passwordFilename,
		"--datadir",
		publisherDir,
		"print-keys",
		"--compressed",
	)
	_, publicKeyString := publicKeyFromDataDir.ExpectRegexp(".+")
	publicKeyFromDataDir.ExpectExit()
	pkComp := strings.Split(publicKeyString[0], "=")[1]
	var m api.Manifest

	err = json.Unmarshal([]byte(matches[0]), &m)
	if err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}

	if len(m.Entries) != 1 {
		t.Fatalf("expected one manifest entry, got %v", len(m.Entries))
	}

	e := m.Entries[0]

	ct := "application/bzz-manifest+json"
	if e.ContentType != ct {
		t.Errorf("expected %q content type, got %q", ct, e.ContentType)
	}

	if e.Access == nil {
		t.Fatal("manifest access is nil")
	}

	a := e.Access

	if a.Type != "pk" {
		t.Errorf(`got access type %q, expected "pk"`, a.Type)
	}
	if len(a.Salt) < 32 {
		t.Errorf(`got salt with length %v, expected not less the 32 bytes`, len(a.Salt))
	}
	if a.KdfParams != nil {
		t.Fatal("manifest access kdf params should be nil")
	}
	if a.Publisher != pkComp {
		t.Fatal("publisher key did not match")
	}
	client := swarmapi.NewClient(cluster.Nodes[0].URL)

	hash, err := client.UploadManifest(&m, false)
	if err != nil {
		t.Fatal(err)
	}

	httpClient := &http.Client{}

	url := cluster.Nodes[0].URL + "/" + "bzz:/" + hash
	response, err := httpClient.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatal("should be a 200")
	}
	d, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(d) != data {
		t.Errorf("expected decrypted data %q, got %q", data, string(d))
	}
}

// testACTWithoutBogus tests the creation of the ACT manifest end-to-end, without any bogus entries (i.e. default scenario = 3 nodes 1 unauthorized)
func testACTWithoutBogus(t *testing.T) {
	testACT(t, 0)
}

// testACTWithBogus tests the creation of the ACT manifest end-to-end, with 100 bogus entries (i.e. 100 EC keys + default scenario = 3 nodes 1 unauthorized = 103 keys in the ACT manifest)
func testACTWithBogus(t *testing.T) {
	testACT(t, 100)
}

// testACT tests the e2e creation, uploading and downloading of an ACT access control with both EC keys AND password protection
// the test fires up a 3 node cluster, then randomly picks 2 nodes which will be acting as grantees to the data
// set and also protects the ACT with a password. the third node should fail decoding the reference as it will not be granted access.
// the third node then then tries to download using a correct password (and succeeds) then uses a wrong password and fails.
// the publisher uploads through one of the nodes then disappears.
func testACT(t *testing.T, bogusEntries int) {
	var uploadThroughNode = cluster.Nodes[0]
	client := swarmapi.NewClient(uploadThroughNode.URL)

	r1 := gorand.New(gorand.NewSource(time.Now().UnixNano()))
	nodeToSkip := r1.Intn(clusterSize) // a number between 0 and 2 (node indices in `cluster`)
	dataFilename := testutil.TempFileWithContent(t, data)
	defer os.RemoveAll(dataFilename)

	// upload the file with 'swarm up' and expect a hash
	up := runSwarm(t,
		"--bzzapi",
		cluster.Nodes[0].URL,
		"up",
		"--encrypt",
		dataFilename)
	_, matches := up.ExpectRegexp(hashRegexp)
	up.ExpectExit()

	if len(matches) < 1 {
		t.Fatal("no matches found")
	}

	ref := matches[0]
	grantees := []string{}
	for i, v := range cluster.Nodes {
		if i == nodeToSkip {
			continue
		}
		pk := v.PrivateKey
		granteePubKey := crypto.CompressPubkey(&pk.PublicKey)
		grantees = append(grantees, hex.EncodeToString(granteePubKey))
	}

	if bogusEntries > 0 {
		bogusGrantees := []string{}

		for i := 0; i < bogusEntries; i++ {
			prv, err := ecies.GenerateKey(rand.Reader, DefaultCurve, nil)
			if err != nil {
				t.Fatal(err)
			}
			bogusGrantees = append(bogusGrantees, hex.EncodeToString(crypto.CompressPubkey(&prv.ExportECDSA().PublicKey)))
		}
		r2 := gorand.New(gorand.NewSource(time.Now().UnixNano()))
		for i := 0; i < len(grantees); i++ {
			insertAtIdx := r2.Intn(len(bogusGrantees))
			bogusGrantees = append(bogusGrantees[:insertAtIdx], append([]string{grantees[i]}, bogusGrantees[insertAtIdx:]...)...)
		}
		grantees = bogusGrantees
	}
	granteesPubkeyListFile := testutil.TempFileWithContent(t, strings.Join(grantees, "\n"))
	defer os.RemoveAll(granteesPubkeyListFile)

	publisherDir, err := ioutil.TempDir("", "swarm-account-dir-temp")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(publisherDir)

	passwordFilename := testutil.TempFileWithContent(t, testPassphrase)
	defer os.RemoveAll(passwordFilename)
	actPasswordFilename := testutil.TempFileWithContent(t, "smth")
	defer os.RemoveAll(actPasswordFilename)
	_, publisherAccount := getTestAccount(t, publisherDir)
	up = runSwarm(t,
		"--bzzaccount",
		publisherAccount.Address.String(),
		"--password",
		passwordFilename,
		"--datadir",
		publisherDir,
		"--bzzapi",
		cluster.Nodes[0].URL,
		"access",
		"new",
		"act",
		"--grant-keys",
		granteesPubkeyListFile,
		"--password",
		actPasswordFilename,
		ref,
	)

	_, matches = up.ExpectRegexp(`[a-f\d]{64}`)
	up.ExpectExit()

	if len(matches) == 0 {
		t.Fatalf("stdout not matched")
	}

	//get the public key from the publisher directory
	publicKeyFromDataDir := runSwarm(t,
		"--bzzaccount",
		publisherAccount.Address.String(),
		"--password",
		passwordFilename,
		"--datadir",
		publisherDir,
		"print-keys",
		"--compressed",
	)
	_, publicKeyString := publicKeyFromDataDir.ExpectRegexp(".+")
	publicKeyFromDataDir.ExpectExit()
	pkComp := strings.Split(publicKeyString[0], "=")[1]

	hash := matches[0]
	m, _, err := client.DownloadManifest(hash)
	if err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}

	if len(m.Entries) != 1 {
		t.Fatalf("expected one manifest entry, got %v", len(m.Entries))
	}

	e := m.Entries[0]

	ct := "application/bzz-manifest+json"
	if e.ContentType != ct {
		t.Errorf("expected %q content type, got %q", ct, e.ContentType)
	}

	if e.Access == nil {
		t.Fatal("manifest access is nil")
	}

	a := e.Access

	if a.Type != "act" {
		t.Fatalf(`got access type %q, expected "act"`, a.Type)
	}
	if len(a.Salt) < 32 {
		t.Fatalf(`got salt with length %v, expected not less the 32 bytes`, len(a.Salt))
	}

	if a.Publisher != pkComp {
		t.Fatal("publisher key did not match")
	}
	httpClient := &http.Client{}

	// all nodes except the skipped node should be able to decrypt the content
	for i, node := range cluster.Nodes {
		log.Debug("trying to fetch from node", "node index", i)

		url := node.URL + "/" + "bzz:/" + hash
		response, err := httpClient.Get(url)
		if err != nil {
			t.Fatal(err)
		}
		log.Debug("got response from node", "response code", response.StatusCode)

		if i == nodeToSkip {
			log.Debug("reached node to skip", "status code", response.StatusCode)

			if response.StatusCode != http.StatusUnauthorized {
				t.Fatalf("should be a 401")
			}

			// try downloading using a password instead, using the unauthorized node
			passwordUrl := strings.Replace(url, "http://", "http://:smth@", -1)
			response, err = httpClient.Get(passwordUrl)
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != http.StatusOK {
				t.Fatal("should be a 200")
			}

			// now try with the wrong password, expect 401
			passwordUrl = strings.Replace(url, "http://", "http://:smthWrong@", -1)
			response, err = httpClient.Get(passwordUrl)
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != http.StatusUnauthorized {
				t.Fatal("should be a 401")
			}
			continue
		}

		if response.StatusCode != http.StatusOK {
			t.Fatal("should be a 200")
		}
		d, err := ioutil.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(d) != data {
			t.Errorf("expected decrypted data %q, got %q", data, string(d))
		}
	}
}

// TestKeypairSanity is a sanity test for the crypto scheme for ACT. it asserts the correct shared secret according to
// the specs at https://github.com/ethersphere/swarm-docs/blob/eb857afda906c6e7bb90d37f3f334ccce5eef230/act.md
func TestKeypairSanity(t *testing.T) {
	salt := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		t.Fatalf("reading from crypto/rand failed: %v", err.Error())
	}
	sharedSecret := "a85586744a1ddd56a7ed9f33fa24f40dd745b3a941be296a0d60e329dbdb896d"

	for i, v := range []struct {
		publisherPriv string
		granteePub    string
	}{
		{
			publisherPriv: "ec5541555f3bc6376788425e9d1a62f55a82901683fd7062c5eddcc373a73459",
			granteePub:    "0226f213613e843a413ad35b40f193910d26eb35f00154afcde9ded57479a6224a",
		},
		{
			publisherPriv: "70c7a73011aa56584a0009ab874794ee7e5652fd0c6911cd02f8b6267dd82d2d",
			granteePub:    "02e6f8d5e28faaa899744972bb847b6eb805a160494690c9ee7197ae9f619181db",
		},
	} {
		b, _ := hex.DecodeString(v.granteePub)
		granteePub, _ := crypto.DecompressPubkey(b)
		publisherPrivate, _ := crypto.HexToECDSA(v.publisherPriv)

		ssKey, err := api.NewSessionKeyPK(publisherPrivate, granteePub, salt)
		if err != nil {
			t.Fatal(err)
		}

		hasher := sha3.NewLegacyKeccak256()
		hasher.Write(salt)
		shared, err := hex.DecodeString(sharedSecret)
		if err != nil {
			t.Fatal(err)
		}
		hasher.Write(shared)
		sum := hasher.Sum(nil)

		if !bytes.Equal(ssKey, sum) {
			t.Fatalf("%d: got a session key mismatch", i)
		}
	}
}
