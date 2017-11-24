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
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/log"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

// sshClient is a small wrapper around Go's SSH client with a few utility methods
// implemented on top.
type sshClient struct {
	server  string // Server name or IP without port number
	address string // IP address of the remote server
	pubkey  []byte // RSA public key to authenticate the server
	client  *ssh.Client
	logger  log.Logger
}

// dial establishes an SSH connection to a remote node using the current user and
// the user's configured private RSA key. If that fails, password authentication
// is fallen back to. The caller may override the login user via user@server:port.
func dial(server string, pubkey []byte) (*sshClient, error) {
	// Figure out a label for the server and a logger
	label := server
	if strings.Contains(label, ":") {
		label = label[:strings.Index(label, ":")]
	}
	login := ""
	if strings.Contains(server, "@") {
		login = label[:strings.Index(label, "@")]
		label = label[strings.Index(label, "@")+1:]
		server = server[strings.Index(server, "@")+1:]
	}
	logger := log.New("server", label)
	logger.Debug("Attempting to establish SSH connection")

	user, err := user.Current()
	if err != nil {
		return nil, err
	}
	if login == "" {
		login = user.Username
	}
	// Configure the supported authentication methods (private key and password)
	var auths []ssh.AuthMethod

	path := filepath.Join(user.HomeDir, ".ssh", "id_rsa")
	if buf, err := ioutil.ReadFile(path); err != nil {
		log.Warn("No SSH key, falling back to passwords", "path", path, "err", err)
	} else {
		key, err := ssh.ParsePrivateKey(buf)
		if err != nil {
			fmt.Printf("What's the decryption password for %s? (won't be echoed)\n>", path)
			blob, err := terminal.ReadPassword(int(os.Stdin.Fd()))
			fmt.Println()
			if err != nil {
				log.Warn("Couldn't read password", "err", err)
			}
			key, err := ssh.ParsePrivateKeyWithPassphrase(buf, blob)
			if err != nil {
				log.Warn("Failed to decrypt SSH key, falling back to passwords", "path", path, "err", err)
			} else {
				auths = append(auths, ssh.PublicKeys(key))
			}
		} else {
			auths = append(auths, ssh.PublicKeys(key))
		}
	}
	auths = append(auths, ssh.PasswordCallback(func() (string, error) {
		fmt.Printf("What's the login password for %s at %s? (won't be echoed)\n> ", login, server)
		blob, err := terminal.ReadPassword(int(os.Stdin.Fd()))

		fmt.Println()
		return string(blob), err
	}))
	// Resolve the IP address of the remote server
	addr, err := net.LookupHost(label)
	if err != nil {
		return nil, err
	}
	if len(addr) == 0 {
		return nil, errors.New("no IPs associated with domain")
	}
	// Try to dial in to the remote server
	logger.Trace("Dialing remote SSH server", "user", login)
	if !strings.Contains(server, ":") {
		server += ":22"
	}
	keycheck := func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		// If no public key is known for SSH, ask the user to confirm
		if pubkey == nil {
			fmt.Println()
			fmt.Printf("The authenticity of host '%s (%s)' can't be established.\n", hostname, remote)
			fmt.Printf("SSH key fingerprint is %s [MD5]\n", ssh.FingerprintLegacyMD5(key))
			fmt.Printf("Are you sure you want to continue connecting (yes/no)? ")

			text, err := bufio.NewReader(os.Stdin).ReadString('\n')
			switch {
			case err != nil:
				return err
			case strings.TrimSpace(text) == "yes":
				pubkey = key.Marshal()
				return nil
			default:
				return fmt.Errorf("unknown auth choice: %v", text)
			}
		}
		// If a public key exists for this SSH server, check that it matches
		if bytes.Equal(pubkey, key.Marshal()) {
			return nil
		}
		// We have a mismatch, forbid connecting
		return errors.New("ssh key mismatch, readd the machine to update")
	}
	client, err := ssh.Dial("tcp", server, &ssh.ClientConfig{User: login, Auth: auths, HostKeyCallback: keycheck})
	if err != nil {
		return nil, err
	}
	// Connection established, return our utility wrapper
	c := &sshClient{
		server:  label,
		address: addr[0],
		pubkey:  pubkey,
		client:  client,
		logger:  logger,
	}
	if err := c.init(); err != nil {
		client.Close()
		return nil, err
	}
	return c, nil
}

// init runs some initialization commands on the remote server to ensure it's
// capable of acting as puppeth target.
func (client *sshClient) init() error {
	client.logger.Debug("Verifying if docker is available")
	if out, err := client.Run("docker version"); err != nil {
		if len(out) == 0 {
			return err
		}
		return fmt.Errorf("docker configured incorrectly: %s", out)
	}
	client.logger.Debug("Verifying if docker-compose is available")
	if out, err := client.Run("docker-compose version"); err != nil {
		if len(out) == 0 {
			return err
		}
		return fmt.Errorf("docker-compose configured incorrectly: %s", out)
	}
	return nil
}

// Close terminates the connection to an SSH server.
func (client *sshClient) Close() error {
	return client.client.Close()
}

// Run executes a command on the remote server and returns the combined output
// along with any error status.
func (client *sshClient) Run(cmd string) ([]byte, error) {
	// Establish a single command session
	session, err := client.client.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	// Execute the command and return any output
	client.logger.Trace("Running command on remote server", "cmd", cmd)
	return session.CombinedOutput(cmd)
}

// Stream executes a command on the remote server and streams all outputs into
// the local stdout and stderr streams.
func (client *sshClient) Stream(cmd string) error {
	// Establish a single command session
	session, err := client.client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	// Execute the command and return any output
	client.logger.Trace("Streaming command on remote server", "cmd", cmd)
	return session.Run(cmd)
}

// Upload copied the set of files to a remote server via SCP, creating any non-
// existing folder in te mean time.
func (client *sshClient) Upload(files map[string][]byte) ([]byte, error) {
	// Establish a single command session
	session, err := client.client.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	// Create a goroutine that streams the SCP content
	go func() {
		out, _ := session.StdinPipe()
		defer out.Close()

		for file, content := range files {
			client.logger.Trace("Uploading file to server", "file", file, "bytes", len(content))

			fmt.Fprintln(out, "D0755", 0, filepath.Dir(file))             // Ensure the folder exists
			fmt.Fprintln(out, "C0644", len(content), filepath.Base(file)) // Create the actual file
			out.Write(content)                                            // Stream the data content
			fmt.Fprint(out, "\x00")                                       // Transfer end with \x00
			fmt.Fprintln(out, "E")                                        // Leave directory (simpler)
		}
	}()
	return session.CombinedOutput("/usr/bin/scp -v -tr ./")
}
