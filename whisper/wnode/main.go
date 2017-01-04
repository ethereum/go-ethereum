// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// This is a simple Whisper node.
// It also implements a command line chat between the peers sharing the same credentials.

package main

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/console"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/nat"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
	"golang.org/x/crypto/pbkdf2"
)

var done chan struct{}
var server *p2p.Server
var shh *whisper.Whisper
var asymKey, nodeid *ecdsa.PrivateKey // used for asymmetric decryption and signing
var pub *ecdsa.PublicKey              // used for asymmetric encryption
var symKey []byte                     // used for symmetric encryption
var filterID uint32
var topic whisper.TopicType
var pow float64 = whisper.MinimumPoW
var ttl uint32 = 30
var workTime uint32 = 5
var ipAddress, enode, salt, topicStr, pubStr, NodeIdFile string
var input *bufio.Reader = bufio.NewReader(os.Stdin)

var bootstrapNode bool // does not actively connect to peers, wait for incoming connections
var daemonMode bool    // only forward messages, neither send nor track
var testMode bool      // predefined password, topic, ip and port
var echoMode bool      // shows params, etc.
var isAsymmetric bool

var (
	argNameHelp       = "-h"
	argNameBootstrap  = "-b"
	argNameDaemon     = "-d"
	argNameAsymmetric = "-a"
	argNameTest       = "-test"
	argNameEcho       = "-echo"
	argNameWorkTime   = "-work"
	argNameTTL        = "-ttl"
	argNameIP         = "-ip"
	argNameTopic      = "-topic"
	argNamePoW        = "-pow"
	argNameSalt       = "-salt"
	argNamePub        = "-pub"
	argNameEnode      = "-enode"
	argNameIdSrc      = "-idfile"
	quitCommand       = "~Q"
)

func padRight(str string) string {
	res := str
	for len(res) < 9 {
		res += " "
	}
	return res
}

func printHelp() {
	fmt.Println("wchat is a stand-alone whisper node with command-line interface.")
	fmt.Println("wchat also allows to set up a chat using either symmetric or asymmetric encryption.")
	fmt.Println()
	fmt.Println("usage: wchat [arguments]")
	fmt.Printf("    %s print this help and exit \n", padRight(argNameHelp))
	fmt.Printf("    %s boostrap node: don't actively connect to peers, wait for incoming connections\n", padRight(argNameBootstrap))
	fmt.Printf("    %s daemon mode: only forward messages, neither send nor decrypt messages \n", padRight(argNameDaemon))
	fmt.Printf("    %s use asymmetric encryption \n", padRight(argNameAsymmetric))
	fmt.Printf("    %s test mode: predefined password, salt, topic, ip and port \n", padRight(argNameTest))
	fmt.Printf("    %s echo mode: prints arguments (diagnostic) \n", padRight(argNameEcho))
	fmt.Printf("    %s IP address and port of this node (e.g. %s=127.0.0.1:30303) \n", padRight(argNameIP), argNameIP)
	fmt.Printf("    %s time-to-live for messages in seconds (e.g. %s=30) \n", padRight(argNameTTL), argNameTTL)
	fmt.Printf("    %s PoW in integer format \n", padRight(argNamePoW))
	fmt.Printf("    %s work time in seconds (e.g. %s=12) \n", padRight(argNameWorkTime), argNameWorkTime)
	fmt.Printf("    %s salt (for topic and key derivation) \n", padRight(argNameSalt))
	fmt.Printf("    %s topic in hexadecimal format (e.g. %s=70a4beef) \n", padRight(argNameTopic), argNameTopic)
	fmt.Printf("    %s public key for asymmetric encryption \n", padRight(argNamePub))
	fmt.Printf("    %s file name with node id (private key) \n", padRight(argNameIdSrc))
	fmt.Printf("    %s enode \n", padRight(argNameEnode))
}

func main() {
	parseArgs()
	initialize()
	run()
}

func parseArgs() {
	var err error
	var p uint32
	help1 := checkMode(argNameHelp)
	help2 := checkMode("-help")
	help3 := checkMode("--help")
	help4 := checkMode("-?")
	if help1 || help2 || help3 || help4 {
		printHelp()
		os.Exit(0)
	}

	bootstrapNode = checkMode(argNameBootstrap)
	daemonMode = checkMode(argNameDaemon)
	testMode = checkMode(argNameTest)
	echoMode = checkMode(argNameEcho)
	isAsymmetric = checkMode(argNameAsymmetric)

	checkIntArg(argNameTTL, &ttl)
	checkIntArg(argNameWorkTime, &workTime)
	checkIntArg(argNamePoW, &p)
	checkStringArg(argNameIP, &ipAddress)
	checkStringArg(argNameEnode, &enode)
	checkStringArg(argNameSalt, &salt)
	checkStringArg(argNameTopic, &topicStr)
	checkStringArg(argNamePub, &pubStr)
	checkStringArg(argNameIdSrc, &NodeIdFile)

	if len(NodeIdFile) > 0 {
		nodeid, err = crypto.LoadECDSA(NodeIdFile)
		if err != nil {
			utils.Fatalf("Failed to load file [%s]: %s.", NodeIdFile, err)
		}
	}

	if len(enode) > 0 {
		prefix := "enode://"
		if enode[:len(prefix)] != prefix {
			enode = prefix + enode
		}
	}

	if p > 0 {
		pow = float64(p)
	}

	if len(topicStr) > 0 {
		var x []byte
		x, err := hex.DecodeString(topicStr)
		if err != nil {
			utils.Fatalf("Failed to parse the topic: %s", err)
		}
		topic = whisper.BytesToTopic(x)
	}

	if isAsymmetric && len(pubStr) > 0 {
		pub = crypto.ToECDSAPub(common.FromHex(pubStr))
		if !isKeyValid(pub) {
			utils.Fatalf("invalid public key")
		}
	}

	if echoMode {
		fmt.Printf("ttl = %d \n", ttl)
		fmt.Printf("workTime = %d \n", workTime)
		fmt.Printf("pow = %f \n", pow)
		fmt.Printf("ip = %s \n", ipAddress)
		fmt.Printf("salt = %s \n", salt)
		fmt.Printf("topic = %x \n", topic)
		fmt.Printf("pub = %s \n", common.ToHex(crypto.FromECDSAPub(pub)))
		fmt.Printf("enode = %s \n", enode)
	}
}

func checkIntArg(pattern string, dst *uint32) {
	pattern += "="
	sz := len(pattern)
	for _, arg := range os.Args {
		if len(arg) < sz {
			continue
		}

		prefix := arg[:sz]
		if prefix == pattern {
			s := arg[sz:]
			i, err := strconv.ParseUint(s, 10, 0)
			if err != nil {
				utils.Fatalf("Failed to parse argument %s: %s", pattern, err)
			}
			if err == nil && i > 0 {
				*dst = uint32(i)
			}
			return
		}
	}
}

func checkStringArg(pattern string, dst *string) {
	pattern += "="
	sz := len(pattern)
	for _, arg := range os.Args {
		if len(arg) < sz {
			continue
		}

		prefix := arg[:sz]
		if prefix == pattern {
			s := arg[sz:]
			if len(s) > 0 {
				*dst = s
			}
			return
		}
	}
}

func checkMode(pattern string) bool {
	for _, arg := range os.Args {
		if arg == pattern {
			return true
		}
	}
	return false
}

func initialize() {
	glog.SetV(logger.Warn)
	glog.SetToStderr(true)

	done = make(chan struct{})
	var peers []*discover.Node

	if testMode {
		password := []byte("this is a test password for symmetric encryption")
		salt := []byte("this is a test salt for symmetric encryption")
		symKey = pbkdf2.Key(password, salt, 64, 32, sha256.New)
		topic = whisper.TopicType{0xFF, 0xFF, 0xFF, 0xFF}
	}

	if bootstrapNode {
		if len(ipAddress) == 0 {
			if testMode {
				ipAddress = "127.0.0.1:30303"
			} else {
				fmt.Printf("Please enter your IP and port (e.g. 127.0.0.1:30303): ")
				fmt.Scanln(&ipAddress)
			}
		}
	} else {
		if len(enode) == 0 {
			fmt.Printf("Please enter the peer's enode: ")
			fmt.Scanln(&enode)
		}
		peer := discover.MustParseNode(enode)
		peers = append(peers, peer)
	}

	shh = whisper.NewWhisper(nil)
	asymKey = shh.NewIdentity()
	if nodeid == nil {
		nodeid = shh.NewIdentity()
	}

	server = &p2p.Server{
		Config: p2p.Config{
			PrivateKey:     nodeid,
			MaxPeers:       128,
			Name:           common.MakeName("whisper-go", "5.0"),
			Protocols:      shh.Protocols(),
			ListenAddr:     ipAddress,
			NAT:            nat.Any(),
			BootstrapNodes: peers,
			StaticNodes:    peers,
			TrustedNodes:   peers,
		},
	}
}

func startServer() {
	err := server.Start()
	if err != nil {
		utils.Fatalf("Failed to start Whsiper peer: %s.", err)
	}

	fmt.Printf("my public key: %s \n", common.ToHex(crypto.FromECDSAPub(&asymKey.PublicKey)))
	fmt.Println(server.NodeInfo().Enode)

	if bootstrapNode {
		configureChat()
		fmt.Println("Bootstrap Whisper node started")
		waitForConnection(false)
	} else {
		fmt.Println("Whisper node started")
		// first see if we can establish connection, then require futher user input
		waitForConnection(true)
		configureChat()
	}

	if !daemonMode {
		fmt.Printf("Chat is enabled. Please type the message. To quit type: '%s'\n", quitCommand)
	}
}

func isKeyValid(k *ecdsa.PublicKey) bool {
	return k.X != nil && k.Y != nil
}

func configureChat() {
	if daemonMode {
		return
	}

	if isAsymmetric && len(pubStr) == 0 {
		pubStr = scanLine("Please enter the peer's public key: ")
		pub = crypto.ToECDSAPub(common.FromHex(pubStr))
		if !isKeyValid(pub) {
			utils.Fatalf("Error: invalid public key")
		}
	}

	if !isAsymmetric && !testMode {
		pass, err := console.Stdin.PromptPassword("Please enter the password: ")
		if err != nil {
			utils.Fatalf("Failed to read passphrase: %v", err)
		}

		if len(salt) == 0 {
			salt = scanLine("Please enter the salt: ")
		}

		symKey = pbkdf2.Key([]byte(pass), []byte(salt), 65356, 32, sha256.New)

		if len(topicStr) == 0 {
			generateTopic([]byte(pass), []byte(salt))
		}
	}

	filter := &whisper.Filter{KeySym: symKey, KeyAsym: asymKey, Topics: []whisper.TopicType{topic}}
	filterID = shh.Watch(filter)
}

func generateTopic(password, salt []byte) {
	const rounds = 4000
	const size = 128
	x1 := pbkdf2.Key(password, salt, rounds, size, sha512.New)
	x2 := pbkdf2.Key(password, salt, rounds, size, sha1.New)
	x3 := pbkdf2.Key(x1, x2, rounds, size, sha256.New)

	for i := 0; i < size; i++ {
		topic[i%whisper.TopicLength] ^= x3[i]
	}
}

func waitForConnection(timeout bool) {
	var cnt int
	var connected bool
	for !connected {
		time.Sleep(time.Millisecond * 50)
		connected = server.PeerCount() > 0
		if timeout {
			cnt++
			if cnt > 1000 {
				utils.Fatalf("Timeout expired, failed to connect")
			}
		}
	}

	fmt.Println("Connected to peer.")
}

func run() {
	startServer()
	defer server.Stop()
	shh.Start(nil)
	defer shh.Stop()

	if !daemonMode {
		go messageLoop()
	}

	for {
		s := scanLine("")
		if s == quitCommand {
			fmt.Println("Quit command received")
			close(done)
			break
		}
		sendMsg([]byte(s))

		if isAsymmetric {
			// print your own message for convenience,
			// because in asymmetric mode it is impossible to decrypt it
			hour, min, sec := time.Now().Clock()
			from := crypto.PubkeyToAddress(asymKey.PublicKey)
			fmt.Printf("\n%02d:%02d:%02d <%x>: %s\n", hour, min, sec, from, s)
		}
	}
}

func scanLine(prompt string) string {
	if len(prompt) > 0 {
		fmt.Print(prompt)
	}
	//txt, err := console.Stdin.PromptInput(prompt) // todo: delete
	txt, err := input.ReadString('\n')
	if err != nil {
		utils.Fatalf("input error: %s", err)
	}
	return strings.TrimRight(txt, "\n\r")
}

func sendMsg(payload []byte) {
	params := whisper.MessageParams{
		Src:      asymKey,
		Dst:      pub,
		KeySym:   symKey,
		Payload:  payload,
		Topic:    topic,
		TTL:      ttl,
		PoW:      pow,
		WorkTime: workTime,
	}

	msg := whisper.NewSentMessage(&params)
	envelope, err := msg.Wrap(&params)
	if err != nil {
		fmt.Printf("failed to seal message: %v \n", err)
		return
	}

	err = shh.Send(envelope)
	if err != nil {
		fmt.Printf("failed to send message: %v \n", err)
	}
}

func messageLoop() {
	f := shh.GetFilter(filterID)
	if f == nil {
		utils.Fatalf("filter is not installed")
	}

	ticker := time.NewTicker(time.Millisecond * 50)

	for {
		select {
		case <-ticker.C:
			messages := f.Retrieve()
			for _, msg := range messages {
				printMessageInfo(msg)
			}
		case <-done:
			return
		}
	}
}

func printMessageInfo(msg *whisper.ReceivedMessage) {
	hour, min, sec := time.Now().Clock()
	timestamp := fmt.Sprintf("%02d:%02d:%02d", hour, min, sec)
	text := string(msg.Payload)

	var address common.Address
	if msg.Src != nil {
		address = crypto.PubkeyToAddress(*msg.Src)
	}

	if whisper.IsPubKeyEqual(msg.Src, &asymKey.PublicKey) {
		fmt.Printf("\n%s <%x>: %s\n", timestamp, address, text) // message from myself
	} else {
		fmt.Printf("\n%s [%x]: %s\n", timestamp, address, text) // message from a peer
	}
}
