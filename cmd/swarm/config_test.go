// Copyright 2017 The go-ethereum Authors
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
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm"
	"github.com/ethereum/go-ethereum/swarm/api"

	"github.com/docker/docker/pkg/reexec"
)

func TestDumpConfig(t *testing.T) {
	swarm := runSwarm(t, "dumpconfig")
	defaultConf := api.NewDefaultConfig()
	out, err := tomlSettings.Marshal(&defaultConf)
	if err != nil {
		t.Fatal(err)
	}
	swarm.Expect(string(out))
	swarm.ExpectExit()
}

func TestFailsSwapEnabledNoSwapApi(t *testing.T) {
	flags := []string{
		fmt.Sprintf("--%s", SwarmNetworkIdFlag.Name), "42",
		fmt.Sprintf("--%s", SwarmPortFlag.Name), "54545",
		fmt.Sprintf("--%s", SwarmSwapEnabledFlag.Name),
	}

	swarm := runSwarm(t, flags...)
	swarm.Expect("Fatal: " + SWARM_ERR_SWAP_SET_NO_API + "\n")
	swarm.ExpectExit()
}

func TestFailsNoBzzAccount(t *testing.T) {
	flags := []string{
		fmt.Sprintf("--%s", SwarmNetworkIdFlag.Name), "42",
		fmt.Sprintf("--%s", SwarmPortFlag.Name), "54545",
	}

	swarm := runSwarm(t, flags...)
	swarm.Expect("Fatal: " + SWARM_ERR_NO_BZZACCOUNT + "\n")
	swarm.ExpectExit()
}

func TestCmdLineOverrides(t *testing.T) {
	dir, err := ioutil.TempDir("", "bzztest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	conf, account := getTestAccount(t, dir)
	node := &testNode{Dir: dir}

	// assign ports
	httpPort, err := assignTCPPort()
	if err != nil {
		t.Fatal(err)
	}

	flags := []string{
		fmt.Sprintf("--%s", SwarmNetworkIdFlag.Name), "42",
		fmt.Sprintf("--%s", SwarmPortFlag.Name), httpPort,
		fmt.Sprintf("--%s", SwarmSyncEnabledFlag.Name),
		fmt.Sprintf("--%s", CorsStringFlag.Name), "*",
		fmt.Sprintf("--%s", SwarmAccountFlag.Name), account.Address.String(),
		fmt.Sprintf("--%s", EnsAPIFlag.Name), "",
		"--datadir", dir,
		"--ipcpath", conf.IPCPath,
	}
	node.Cmd = runSwarm(t, flags...)
	node.Cmd.InputLine(testPassphrase)
	defer func() {
		if t.Failed() {
			node.Shutdown()
		}
	}()
	// wait for the node to start
	for start := time.Now(); time.Since(start) < 10*time.Second; time.Sleep(50 * time.Millisecond) {
		node.Client, err = rpc.Dial(conf.IPCEndpoint())
		if err == nil {
			break
		}
	}
	if node.Client == nil {
		t.Fatal(err)
	}

	// load info
	var info swarm.Info
	if err := node.Client.Call(&info, "bzz_info"); err != nil {
		t.Fatal(err)
	}

	if info.Port != httpPort {
		t.Fatalf("Expected port to be %s, got %s", httpPort, info.Port)
	}

	if info.NetworkId != 42 {
		t.Fatalf("Expected network ID to be %d, got %d", 42, info.NetworkId)
	}

	if !info.SyncEnabled {
		t.Fatal("Expected Sync to be enabled, but is false")
	}

	if info.Cors != "*" {
		t.Fatalf("Expected Cors flag to be set to %s, got %s", "*", info.Cors)
	}

	node.Shutdown()
}

func TestFileOverrides(t *testing.T) {

	// assign ports
	httpPort, err := assignTCPPort()
	if err != nil {
		t.Fatal(err)
	}

	//create a config file
	//first, create a default conf
	defaultConf := api.NewDefaultConfig()
	//change some values in order to test if they have been loaded
	defaultConf.SyncEnabled = true
	defaultConf.NetworkId = 54
	defaultConf.Port = httpPort
	defaultConf.StoreParams.DbCapacity = 9000000
	defaultConf.ChunkerParams.Branches = 64
	defaultConf.HiveParams.CallInterval = 6000000000
	defaultConf.Swap.Params.Strategy.AutoCashInterval = 600 * time.Second
	defaultConf.SyncParams.KeyBufferSize = 512
	//create a TOML string
	out, err := tomlSettings.Marshal(&defaultConf)
	if err != nil {
		t.Fatalf("Error creating TOML file in TestFileOverride: %v", err)
	}
	//create file
	f, err := ioutil.TempFile("", "testconfig.toml")
	if err != nil {
		t.Fatalf("Error writing TOML file in TestFileOverride: %v", err)
	}
	//write file
	_, err = f.WriteString(string(out))
	if err != nil {
		t.Fatalf("Error writing TOML file in TestFileOverride: %v", err)
	}
	f.Sync()

	dir, err := ioutil.TempDir("", "bzztest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	conf, account := getTestAccount(t, dir)
	node := &testNode{Dir: dir}

	flags := []string{
		fmt.Sprintf("--%s", SwarmTomlConfigPathFlag.Name), f.Name(),
		fmt.Sprintf("--%s", SwarmAccountFlag.Name), account.Address.String(),
		"--ens-api", "",
		"--ipcpath", conf.IPCPath,
		"--datadir", dir,
	}
	node.Cmd = runSwarm(t, flags...)
	node.Cmd.InputLine(testPassphrase)
	defer func() {
		if t.Failed() {
			node.Shutdown()
		}
	}()
	// wait for the node to start
	for start := time.Now(); time.Since(start) < 10*time.Second; time.Sleep(50 * time.Millisecond) {
		node.Client, err = rpc.Dial(conf.IPCEndpoint())
		if err == nil {
			break
		}
	}
	if node.Client == nil {
		t.Fatal(err)
	}

	// load info
	var info swarm.Info
	if err := node.Client.Call(&info, "bzz_info"); err != nil {
		t.Fatal(err)
	}

	if info.Port != httpPort {
		t.Fatalf("Expected port to be %s, got %s", httpPort, info.Port)
	}

	if info.NetworkId != 54 {
		t.Fatalf("Expected network ID to be %d, got %d", 54, info.NetworkId)
	}

	if !info.SyncEnabled {
		t.Fatal("Expected Sync to be enabled, but is false")
	}

	if info.StoreParams.DbCapacity != 9000000 {
		t.Fatalf("Expected network ID to be %d, got %d", 54, info.NetworkId)
	}

	if info.ChunkerParams.Branches != 64 {
		t.Fatalf("Expected chunker params branches to be %d, got %d", 64, info.ChunkerParams.Branches)
	}

	if info.HiveParams.CallInterval != 6000000000 {
		t.Fatalf("Expected HiveParams CallInterval to be %d, got %d", uint64(6000000000), uint64(info.HiveParams.CallInterval))
	}

	if info.Swap.Params.Strategy.AutoCashInterval != 600*time.Second {
		t.Fatalf("Expected SwapParams AutoCashInterval to be %ds, got %d", 600, info.Swap.Params.Strategy.AutoCashInterval)
	}

	if info.SyncParams.KeyBufferSize != 512 {
		t.Fatalf("Expected info.SyncParams.KeyBufferSize to be %d, got %d", 512, info.SyncParams.KeyBufferSize)
	}

	node.Shutdown()
}

func TestEnvVars(t *testing.T) {
	// assign ports
	httpPort, err := assignTCPPort()
	if err != nil {
		t.Fatal(err)
	}

	envVars := os.Environ()
	envVars = append(envVars, fmt.Sprintf("%s=%s", SwarmPortFlag.EnvVar, httpPort))
	envVars = append(envVars, fmt.Sprintf("%s=%s", SwarmNetworkIdFlag.EnvVar, "999"))
	envVars = append(envVars, fmt.Sprintf("%s=%s", CorsStringFlag.EnvVar, "*"))
	envVars = append(envVars, fmt.Sprintf("%s=%s", SwarmSyncEnabledFlag.EnvVar, "true"))

	dir, err := ioutil.TempDir("", "bzztest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	conf, account := getTestAccount(t, dir)
	node := &testNode{Dir: dir}
	flags := []string{
		fmt.Sprintf("--%s", SwarmAccountFlag.Name), account.Address.String(),
		"--ens-api", "",
		"--datadir", dir,
		"--ipcpath", conf.IPCPath,
	}

	//node.Cmd = runSwarm(t,flags...)
	//node.Cmd.cmd.Env = envVars
	//the above assignment does not work, so we need a custom Cmd here in order to pass envVars:
	cmd := &exec.Cmd{
		Path:   reexec.Self(),
		Args:   append([]string{"swarm-test"}, flags...),
		Stderr: os.Stderr,
		Stdout: os.Stdout,
	}
	cmd.Env = envVars
	//stdout, err := cmd.StdoutPipe()
	//if err != nil {
	//	t.Fatal(err)
	//}
	//stdout = bufio.NewReader(stdout)
	var stdin io.WriteCloser
	if stdin, err = cmd.StdinPipe(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	//cmd.InputLine(testPassphrase)
	io.WriteString(stdin, testPassphrase+"\n")
	defer func() {
		if t.Failed() {
			node.Shutdown()
			cmd.Process.Kill()
		}
	}()
	// wait for the node to start
	for start := time.Now(); time.Since(start) < 10*time.Second; time.Sleep(50 * time.Millisecond) {
		node.Client, err = rpc.Dial(conf.IPCEndpoint())
		if err == nil {
			break
		}
	}

	if node.Client == nil {
		t.Fatal(err)
	}

	// load info
	var info swarm.Info
	if err := node.Client.Call(&info, "bzz_info"); err != nil {
		t.Fatal(err)
	}

	if info.Port != httpPort {
		t.Fatalf("Expected port to be %s, got %s", httpPort, info.Port)
	}

	if info.NetworkId != 999 {
		t.Fatalf("Expected network ID to be %d, got %d", 999, info.NetworkId)
	}

	if info.Cors != "*" {
		t.Fatalf("Expected Cors flag to be set to %s, got %s", "*", info.Cors)
	}

	if !info.SyncEnabled {
		t.Fatal("Expected Sync to be enabled, but is false")
	}

	node.Shutdown()
	cmd.Process.Kill()
}

func TestCmdLineOverridesFile(t *testing.T) {

	// assign ports
	httpPort, err := assignTCPPort()
	if err != nil {
		t.Fatal(err)
	}

	//create a config file
	//first, create a default conf
	defaultConf := api.NewDefaultConfig()
	//change some values in order to test if they have been loaded
	defaultConf.SyncEnabled = false
	defaultConf.NetworkId = 54
	defaultConf.Port = "8588"
	defaultConf.StoreParams.DbCapacity = 9000000
	defaultConf.ChunkerParams.Branches = 64
	defaultConf.HiveParams.CallInterval = 6000000000
	defaultConf.Swap.Params.Strategy.AutoCashInterval = 600 * time.Second
	defaultConf.SyncParams.KeyBufferSize = 512
	//create a TOML file
	out, err := tomlSettings.Marshal(&defaultConf)
	if err != nil {
		t.Fatalf("Error creating TOML file in TestFileOverride: %v", err)
	}
	//write file
	f, err := ioutil.TempFile("", "testconfig.toml")
	if err != nil {
		t.Fatalf("Error writing TOML file in TestFileOverride: %v", err)
	}
	//write file
	_, err = f.WriteString(string(out))
	if err != nil {
		t.Fatalf("Error writing TOML file in TestFileOverride: %v", err)
	}
	f.Sync()

	dir, err := ioutil.TempDir("", "bzztest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	conf, account := getTestAccount(t, dir)
	node := &testNode{Dir: dir}

	expectNetworkId := uint64(77)

	flags := []string{
		fmt.Sprintf("--%s", SwarmNetworkIdFlag.Name), "77",
		fmt.Sprintf("--%s", SwarmPortFlag.Name), httpPort,
		fmt.Sprintf("--%s", SwarmSyncEnabledFlag.Name),
		fmt.Sprintf("--%s", SwarmTomlConfigPathFlag.Name), f.Name(),
		fmt.Sprintf("--%s", SwarmAccountFlag.Name), account.Address.String(),
		"--ens-api", "",
		"--datadir", dir,
		"--ipcpath", conf.IPCPath,
	}
	node.Cmd = runSwarm(t, flags...)
	node.Cmd.InputLine(testPassphrase)
	defer func() {
		if t.Failed() {
			node.Shutdown()
		}
	}()
	// wait for the node to start
	for start := time.Now(); time.Since(start) < 10*time.Second; time.Sleep(50 * time.Millisecond) {
		node.Client, err = rpc.Dial(conf.IPCEndpoint())
		if err == nil {
			break
		}
	}
	if node.Client == nil {
		t.Fatal(err)
	}

	// load info
	var info swarm.Info
	if err := node.Client.Call(&info, "bzz_info"); err != nil {
		t.Fatal(err)
	}

	if info.Port != httpPort {
		t.Fatalf("Expected port to be %s, got %s", httpPort, info.Port)
	}

	if info.NetworkId != expectNetworkId {
		t.Fatalf("Expected network ID to be %d, got %d", expectNetworkId, info.NetworkId)
	}

	if !info.SyncEnabled {
		t.Fatal("Expected Sync to be enabled, but is false")
	}

	if info.StoreParams.DbCapacity != 9000000 {
		t.Fatalf("Expected network ID to be %d, got %d", 54, info.NetworkId)
	}

	if info.ChunkerParams.Branches != 64 {
		t.Fatalf("Expected chunker params branches to be %d, got %d", 64, info.ChunkerParams.Branches)
	}

	if info.HiveParams.CallInterval != 6000000000 {
		t.Fatalf("Expected HiveParams CallInterval to be %d, got %d", uint64(6000000000), uint64(info.HiveParams.CallInterval))
	}

	if info.Swap.Params.Strategy.AutoCashInterval != 600*time.Second {
		t.Fatalf("Expected SwapParams AutoCashInterval to be %ds, got %d", 600, info.Swap.Params.Strategy.AutoCashInterval)
	}

	if info.SyncParams.KeyBufferSize != 512 {
		t.Fatalf("Expected info.SyncParams.KeyBufferSize to be %d, got %d", 512, info.SyncParams.KeyBufferSize)
	}

	node.Shutdown()
}
