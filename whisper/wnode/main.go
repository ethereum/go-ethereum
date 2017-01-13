// Copyright 2016 The go-ethereum Authors
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

// This is a simple Whisper node. It could be used as a stand-alone bootstrap node.
// Also, could be used for different test and diagnostics purposes.

package main

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
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
	"github.com/ethereum/go-ethereum/rlp"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
	"golang.org/x/crypto/pbkdf2"
)

const sizeOfInt = 4

// singletons
var (
	server     *p2p.Server
	shh        *whisper.Whisper
	mailServer WMailServer

	done  chan struct{}
	input *bufio.Reader = bufio.NewReader(os.Stdin)
)

// encryption/decryption
var (
	pub     *ecdsa.PublicKey
	asymKey *ecdsa.PrivateKey
	nodeid  *ecdsa.PrivateKey
	symKey  []byte
)

// parameters
var (
	filterID uint32
	topic    whisper.TopicType
	ttl      uint32 = 30
	workTime uint32 = 5
	msPoW    float64
	timeLow  uint32
	timeUpp  uint32
	pow      float64 = whisper.MinimumPoW

	ipAddress, enode, salt, topicStr, pubStr, NodeIdFile, dbPath, msPassword string
)

var (
	bootstrapMode  bool // does not actively connect to peers, wait for incoming connections
	forwarderMode  bool // only forward messages, neither send nor track
	mailServerMode bool // delivers expired messages on demand
	testMode       bool // predefined password, topic, ip and port
	echoMode       bool // shows params, etc.
	isAsymmetric   bool
	requestMail    bool
)

var (
	argNameHelp       = "-h"
	argBootstrapMode  = "-b"
	argNameForwarder  = "-f"
	argNameAsymmetric = "-a"
	argMailServerMode = "-m"
	argRequestMail    = "-r"
	argNameTest       = "-test"
	argNameEcho       = "-echo"
	argNameWorkTime   = "-work"
	argNameTTL        = "-ttl"
	argNameIP         = "-ip"
	argNameTopic      = "-topic"
	argNamePoW        = "-pow"
	argNameSalt       = "-salt"
	argNamePub        = "-pub"
	argNameBoot       = "-boot"
	argNameIdSrc      = "-idfile"
	argNameDbPath     = "-dbpath"
	argNameMSPoW      = "-mspow"

	quitCommand = "~Q"
	enodePrefix = "enode://"
)

// this is a temporary stub, will be expanded later
type WMailServer struct{}

func (s *WMailServer) Archive(env *whisper.Envelope)                                      {}
func (s *WMailServer) DeliverMail(peer *whisper.Peer, data []byte)                        {}
func (s *WMailServer) Init(w *whisper.Whisper, path string, password string, pow float64) {}
func (s *WMailServer) Close()                                                             {}

func padRight(str string) string {
	res := str
	for len(res) < 9 {
		res += " "
	}
	return res
}

func printHelp() {
	fmt.Println("wnode is a stand-alone whisper node with command-line interface.")
	fmt.Println()
	fmt.Println("usage: wnode [arguments]")
	fmt.Printf("    %s print this help and exit \n", padRight(argNameHelp))
	fmt.Printf("    %s boostrap node: don't actively connect to peers, wait for incoming connections\n", padRight(argBootstrapMode))
	fmt.Printf("    %s forwarder mode: only forward messages, neither send nor decrypt messages \n", padRight(argNameForwarder))
	fmt.Printf("    %s use asymmetric encryption \n", padRight(argNameAsymmetric))
	fmt.Printf("    %s mail server mode: delivers expired messages on demand \n", padRight(argMailServerMode))
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
	fmt.Printf("    %s path to the DB directory \n", padRight(argNameDbPath))
	fmt.Printf("    %s PoW requirement for Mail Server (int) \n", padRight(argNameMSPoW))
	fmt.Printf("    %s request old (expired) messages from the bootstrap server \n", padRight(argRequestMail))
	fmt.Printf("    %s bootstrap node you want to connect to (e.g. %s=enode://e454......08d50@52.176.211.200:16428) \n", padRight(argNameBoot), argNameBoot)
}

func main() {
	parseArgs()
	initialize()
	run()
}

func parseArgs() {
	var err error
	var p, x uint32
	help1 := checkMode(argNameHelp)
	help2 := checkMode("-help")
	help3 := checkMode("--help")
	help4 := checkMode("-?")
	if help1 || help2 || help3 || help4 {
		printHelp()
		os.Exit(0)
	}

	bootstrapMode = checkMode(argBootstrapMode)
	forwarderMode = checkMode(argNameForwarder)
	mailServerMode = checkMode(argMailServerMode)
	testMode = checkMode(argNameTest)
	echoMode = checkMode(argNameEcho)
	isAsymmetric = checkMode(argNameAsymmetric)
	requestMail = checkMode(argRequestMail)

	checkIntArg(argNameTTL, &ttl)
	checkIntArg(argNameWorkTime, &workTime)
	checkIntArg(argNamePoW, &p)
	checkIntArg(argNameMSPoW, &x)
	checkStringArg(argNameIP, &ipAddress)
	checkStringArg(argNameBoot, &enode)
	checkStringArg(argNameSalt, &salt)
	checkStringArg(argNameTopic, &topicStr)
	checkStringArg(argNamePub, &pubStr)
	checkStringArg(argNameIdSrc, &NodeIdFile)
	checkStringArg(argNameDbPath, &dbPath)

	if len(NodeIdFile) > 0 {
		nodeid, err = crypto.LoadECDSA(NodeIdFile)
		if err != nil {
			utils.Fatalf("Failed to load file [%s]: %s.", NodeIdFile, err)
		}
	}

	if len(enode) > 0 {
		if enode[:len(enodePrefix)] != enodePrefix {
			enode = enodePrefix + enode
		}
	}

	if p > 0 {
		pow = float64(p)
	}

	if x > 0 {
		msPoW = float64(x)
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
		fmt.Printf("boot = %s \n", enode)
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
	var err error

	if testMode {
		password := []byte("this is a test password for symmetric encryption")
		salt := []byte("this is a test salt for symmetric encryption")
		symKey = pbkdf2.Key(password, salt, 64, 32, sha256.New)
		topic = whisper.TopicType{0xFF, 0xFF, 0xFF, 0xFF}
	}

	if bootstrapMode {
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

	if mailServerMode {
		msPassword, err = console.Stdin.PromptPassword("Please enter the Mail Server password: ")
		if err != nil {
			utils.Fatalf("Failed to read Mail Server password: %s", err)
		}

		shh = whisper.NewWhisper(&mailServer)
		mailServer.Init(shh, dbPath, msPassword, msPoW)
	} else {
		shh = whisper.NewWhisper(nil)
	}

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

	if bootstrapMode {
		configureNode()
		fmt.Println("Bootstrap Whisper node started")
		//waitForConnection(false) // todo: review
	} else {
		fmt.Println("Whisper node started")
		// first see if we can establish connection, then require futher user input
		waitForConnection(true)
		configureNode()
	}

	if !forwarderMode {
		fmt.Printf("Connection is established. Please type the message. To quit type: '%s'\n", quitCommand)
	}
}

func isKeyValid(k *ecdsa.PublicKey) bool {
	return k.X != nil && k.Y != nil
}

func configureNode() {
	var i int
	var s string
	var err error

	if forwarderMode {
		return
	}

	if isAsymmetric && len(pubStr) == 0 {
		pubStr = scanLine("Please enter the peer's public key: ")
		pub = crypto.ToECDSAPub(common.FromHex(pubStr))
		if !isKeyValid(pub) {
			utils.Fatalf("Error: invalid public key")
		}
	}

	if !isAsymmetric && !testMode && !forwarderMode {
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

	if mailServerMode {
		if len(dbPath) == 0 {
			dbPath = scanLine("Please enter the path to DB file: ")
		}

		if msPoW == 0.0 {
			s = scanLine("Please enter the PoW requirement for the Mail Server (int): ")
			i, err = strconv.Atoi(s)
			if err != nil {
				utils.Fatalf("Fail to parse the PoW: %s", err)
			}
			msPoW = float64(i)
		}
	}

	if requestMail {
		msPassword, err = console.Stdin.PromptPassword("Please enter the Mail Server password: ")
		if err != nil {
			utils.Fatalf("Failed to read Mail Server password: %s", err)
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
	defer mailServer.Close()
	startServer()
	defer server.Stop()
	shh.Start(nil)
	defer shh.Stop()

	if !forwarderMode {
		go messageLoop()
	}

	if requestMail {
		requestExpiredMessagesLoop()
	} else {
		sendLoop()
	}
}

func sendLoop() {
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
	timestamp := fmt.Sprintf("%d", msg.Sent) // unix timestamp for diagnostics
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

func requestExpiredMessagesLoop() {
	var key, peerID []byte
	err := shh.AddSymKey(argRequestMail, []byte(msPassword))
	if err != nil {
		utils.Fatalf("Failed to create symmetric key for mail request: %s", err)
	}
	key = shh.GetSymKey(argRequestMail)
	peerID = extractIdFromEnode(enode)
	shh.MarkPeerTrusted(peerID)

	for {
		s := scanLine("Please enter the lower limit for the time range (unix timestamp): ")
		i, err := strconv.Atoi(s)
		if err != nil {
			utils.Fatalf("Fail to parse the lower time limit: %s", err)
		}
		timeLow = uint32(i)

		s = scanLine("Please enter the upper limit for the time range (unix timestamp): ")
		i, err = strconv.Atoi(s)
		if err != nil {
			utils.Fatalf("Fail to parse the upper time limit: %s", err)
		}
		timeUpp = uint32(i)
		if timeUpp == 0 {
			timeUpp = 0xFFFFFFFF
		}

		data := make([]byte, sizeOfInt*2)
		binary.BigEndian.PutUint32(data, timeLow)
		binary.BigEndian.PutUint32(data[sizeOfInt:], timeUpp)
		// todo: add topic

		var params whisper.MessageParams
		params.PoW = msPoW
		params.Payload = data
		params.KeySym = key
		params.Src = nodeid
		params.WorkTime = 5

		msg := whisper.NewSentMessage(&params)
		env, err := msg.Wrap(&params)
		if err != nil {
			utils.Fatalf("Wrap failed: %s", err)
		}

		encoded, err := rlp.EncodeToBytes(env)
		if err != nil {
			utils.Fatalf("RLP encoding failed: %s", err)
		}

		err = shh.RequestHistoricMessages(peerID, encoded)
		if err != nil {
			utils.Fatalf("Failed to send P2P message: %s", err)
		}

		time.Sleep(time.Second * 5)
	}
}

func extractIdFromEnode(s string) []byte {
	if len(s) == 0 {
		return nil
	}

	p := len(enodePrefix)
	if s[:p] == enodePrefix {
		s = s[p:]
	}

	i := strings.Index(s, "@")
	if i > 0 {
		s = s[:i]
	}

	b, err := hex.DecodeString(s)
	if err != nil {
		utils.Fatalf("Failed to decode enode: %s", err)
		return nil
	}

	return b
}
