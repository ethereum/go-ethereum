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

// This is a simple Whisper node. It could be used as a stand-alone bootstrap node.
// Also, could be used for different test and diagnostics purposes.

package main

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/console"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/ethereum/go-ethereum/whisper/mailserver"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
	"golang.org/x/crypto/pbkdf2"
)

const quitCommand = "~Q"

// singletons
var (
	server     *p2p.Server
	shh        *whisper.Whisper
	done       chan struct{}
	mailServer mailserver.WMailServer

	input = bufio.NewReader(os.Stdin)
)

// encryption
var (
	symKey  []byte
	pub     *ecdsa.PublicKey
	asymKey *ecdsa.PrivateKey
	nodeid  *ecdsa.PrivateKey
	topic   whisper.TopicType

	asymKeyID    string
	asymFilterID string
	symFilterID  string
	symPass      string
	msPassword   string
)

// cmd arguments
var (
	bootstrapMode  = flag.Bool("standalone", false, "boostrap node: don't actively connect to peers, wait for incoming connections")
	forwarderMode  = flag.Bool("forwarder", false, "forwarder mode: only forward messages, neither send nor decrypt messages")
	mailServerMode = flag.Bool("mailserver", false, "mail server mode: delivers expired messages on demand")
	requestMail    = flag.Bool("mailclient", false, "request expired messages from the bootstrap server")
	asymmetricMode = flag.Bool("asym", false, "use asymmetric encryption")
	generateKey    = flag.Bool("generatekey", false, "generate and show the private key")
	fileExMode     = flag.Bool("fileexchange", false, "file exchange mode")
	testMode       = flag.Bool("test", false, "use of predefined parameters for diagnostics")
	echoMode       = flag.Bool("echo", false, "echo mode: prints some arguments for diagnostics")

	argVerbosity = flag.Int("verbosity", int(log.LvlError), "log verbosity level")
	argTTL       = flag.Uint("ttl", 30, "time-to-live for messages in seconds")
	argWorkTime  = flag.Uint("work", 5, "work time in seconds")
	argMaxSize   = flag.Uint("maxsize", uint(whisper.DefaultMaxMessageSize), "max size of message")
	argPoW       = flag.Float64("pow", whisper.DefaultMinimumPoW, "PoW for normal messages in float format (e.g. 2.7)")
	argServerPoW = flag.Float64("mspow", whisper.DefaultMinimumPoW, "PoW requirement for Mail Server request")

	argIP      = flag.String("ip", "", "IP address and port of this node (e.g. 127.0.0.1:30303)")
	argPub     = flag.String("pub", "", "public key for asymmetric encryption")
	argDBPath  = flag.String("dbpath", "", "path to the server's DB directory")
	argIDFile  = flag.String("idfile", "", "file name with node id (private key)")
	argEnode   = flag.String("boot", "", "bootstrap node you want to connect to (e.g. enode://e454......08d50@52.176.211.200:16428)")
	argTopic   = flag.String("topic", "", "topic in hexadecimal format (e.g. 70a4beef)")
	argSaveDir = flag.String("savedir", "", "directory where incoming messages will be saved as files")
)

func main() {
	processArgs()
	initialize()
	run()
}

func processArgs() {
	flag.Parse()

	if len(*argIDFile) > 0 {
		var err error
		nodeid, err = crypto.LoadECDSA(*argIDFile)
		if err != nil {
			utils.Fatalf("Failed to load file [%s]: %s.", *argIDFile, err)
		}
	}

	const enodePrefix = "enode://"
	if len(*argEnode) > 0 {
		if (*argEnode)[:len(enodePrefix)] != enodePrefix {
			*argEnode = enodePrefix + *argEnode
		}
	}

	if len(*argTopic) > 0 {
		x, err := hex.DecodeString(*argTopic)
		if err != nil {
			utils.Fatalf("Failed to parse the topic: %s", err)
		}
		topic = whisper.BytesToTopic(x)
	}

	if *asymmetricMode && len(*argPub) > 0 {
		pub = crypto.ToECDSAPub(common.FromHex(*argPub))
		if !isKeyValid(pub) {
			utils.Fatalf("invalid public key")
		}
	}

	if len(*argSaveDir) > 0 {
		if _, err := os.Stat(*argSaveDir); os.IsNotExist(err) {
			utils.Fatalf("Download directory '%s' does not exist", *argSaveDir)
		}
	} else if *fileExMode {
		utils.Fatalf("Parameter 'savedir' is mandatory for file exchange mode")
	}

	if *echoMode {
		echo()
	}
}

func echo() {
	fmt.Printf("ttl = %d \n", *argTTL)
	fmt.Printf("workTime = %d \n", *argWorkTime)
	fmt.Printf("pow = %f \n", *argPoW)
	fmt.Printf("mspow = %f \n", *argServerPoW)
	fmt.Printf("ip = %s \n", *argIP)
	fmt.Printf("pub = %s \n", common.ToHex(crypto.FromECDSAPub(pub)))
	fmt.Printf("idfile = %s \n", *argIDFile)
	fmt.Printf("dbpath = %s \n", *argDBPath)
	fmt.Printf("boot = %s \n", *argEnode)
}

func initialize() {
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(*argVerbosity), log.StreamHandler(os.Stderr, log.TerminalFormat(false))))

	done = make(chan struct{})
	var peers []*discover.Node
	var err error

	if *generateKey {
		key, err := crypto.GenerateKey()
		if err != nil {
			utils.Fatalf("Failed to generate private key: %s", err)
		}
		k := hex.EncodeToString(crypto.FromECDSA(key))
		fmt.Printf("Random private key: %s \n", k)
		os.Exit(0)
	}

	if *testMode {
		symPass = "wwww" // ascii code: 0x77777777
		msPassword = "wwww"
	}

	if *bootstrapMode {
		if len(*argIP) == 0 {
			argIP = scanLineA("Please enter your IP and port (e.g. 127.0.0.1:30348): ")
		}
	} else {
		if len(*argEnode) == 0 {
			argEnode = scanLineA("Please enter the peer's enode: ")
		}
		peer := discover.MustParseNode(*argEnode)
		peers = append(peers, peer)
	}

	cfg := &whisper.Config{
		MaxMessageSize:     uint32(*argMaxSize),
		MinimumAcceptedPOW: *argPoW,
	}

	if *mailServerMode {
		if len(msPassword) == 0 {
			msPassword, err = console.Stdin.PromptPassword("Please enter the Mail Server password: ")
			if err != nil {
				utils.Fatalf("Failed to read Mail Server password: %s", err)
			}
		}

		shh = whisper.New(cfg)
		shh.RegisterServer(&mailServer)
		mailServer.Init(shh, *argDBPath, msPassword, *argServerPoW)
	} else {
		shh = whisper.New(cfg)
	}

	if *argPoW != whisper.DefaultMinimumPoW {
		err := shh.SetMinimumPoW(*argPoW)
		if err != nil {
			utils.Fatalf("Failed to set PoW: %s", err)
		}
	}

	if uint32(*argMaxSize) != whisper.DefaultMaxMessageSize {
		err := shh.SetMaxMessageSize(uint32(*argMaxSize))
		if err != nil {
			utils.Fatalf("Failed to set max message size: %s", err)
		}
	}

	asymKeyID, err = shh.NewKeyPair()
	if err != nil {
		utils.Fatalf("Failed to generate a new key pair: %s", err)
	}

	asymKey, err = shh.GetPrivateKey(asymKeyID)
	if err != nil {
		utils.Fatalf("Failed to retrieve a new key pair: %s", err)
	}

	if nodeid == nil {
		tmpID, err := shh.NewKeyPair()
		if err != nil {
			utils.Fatalf("Failed to generate a new key pair: %s", err)
		}

		nodeid, err = shh.GetPrivateKey(tmpID)
		if err != nil {
			utils.Fatalf("Failed to retrieve a new key pair: %s", err)
		}
	}

	maxPeers := 80
	if *bootstrapMode {
		maxPeers = 800
	}

	server = &p2p.Server{
		Config: p2p.Config{
			PrivateKey:     nodeid,
			MaxPeers:       maxPeers,
			Name:           common.MakeName("wnode", "5.0"),
			Protocols:      shh.Protocols(),
			ListenAddr:     *argIP,
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
		utils.Fatalf("Failed to start Whisper peer: %s.", err)
	}

	fmt.Printf("my public key: %s \n", common.ToHex(crypto.FromECDSAPub(&asymKey.PublicKey)))
	fmt.Println(server.NodeInfo().Enode)

	if *bootstrapMode {
		configureNode()
		fmt.Println("Bootstrap Whisper node started")
	} else {
		fmt.Println("Whisper node started")
		// first see if we can establish connection, then ask for user input
		waitForConnection(true)
		configureNode()
	}

	if !*forwarderMode {
		fmt.Printf("Please type the message. To quit type: '%s'\n", quitCommand)
	}
}

func isKeyValid(k *ecdsa.PublicKey) bool {
	return k.X != nil && k.Y != nil
}

func configureNode() {
	var err error
	var p2pAccept bool

	if *forwarderMode {
		return
	}

	if *asymmetricMode {
		if len(*argPub) == 0 {
			s := scanLine("Please enter the peer's public key: ")
			b := common.FromHex(s)
			if b == nil {
				utils.Fatalf("Error: can not convert hexadecimal string")
			}
			pub = crypto.ToECDSAPub(b)
			if !isKeyValid(pub) {
				utils.Fatalf("Error: invalid public key")
			}
		}
	}

	if *requestMail {
		p2pAccept = true
		if len(msPassword) == 0 {
			msPassword, err = console.Stdin.PromptPassword("Please enter the Mail Server password: ")
			if err != nil {
				utils.Fatalf("Failed to read Mail Server password: %s", err)
			}
		}
	}

	if !*asymmetricMode && !*forwarderMode {
		if len(symPass) == 0 {
			symPass, err = console.Stdin.PromptPassword("Please enter the password for symmetric encryption: ")
			if err != nil {
				utils.Fatalf("Failed to read passphrase: %v", err)
			}
		}

		symKeyID, err := shh.AddSymKeyFromPassword(symPass)
		if err != nil {
			utils.Fatalf("Failed to create symmetric key: %s", err)
		}
		symKey, err = shh.GetSymKey(symKeyID)
		if err != nil {
			utils.Fatalf("Failed to save symmetric key: %s", err)
		}
		if len(*argTopic) == 0 {
			generateTopic([]byte(symPass))
		}

		fmt.Printf("Filter is configured for the topic: %x \n", topic)
	}

	if *mailServerMode {
		if len(*argDBPath) == 0 {
			argDBPath = scanLineA("Please enter the path to DB file: ")
		}
	}

	symFilter := whisper.Filter{
		KeySym:   symKey,
		Topics:   [][]byte{topic[:]},
		AllowP2P: p2pAccept,
	}
	symFilterID, err = shh.Subscribe(&symFilter)
	if err != nil {
		utils.Fatalf("Failed to install filter: %s", err)
	}

	asymFilter := whisper.Filter{
		KeyAsym:  asymKey,
		Topics:   [][]byte{topic[:]},
		AllowP2P: p2pAccept,
	}
	asymFilterID, err = shh.Subscribe(&asymFilter)
	if err != nil {
		utils.Fatalf("Failed to install filter: %s", err)
	}
}

func generateTopic(password []byte) {
	x := pbkdf2.Key(password, password, 4096, 128, sha512.New)
	for i := 0; i < len(x); i++ {
		topic[i%whisper.TopicLength] ^= x[i]
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

	if !*forwarderMode {
		go messageLoop()
	}

	if *requestMail {
		requestExpiredMessagesLoop()
	} else if *fileExMode {
		sendFilesLoop()
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

		if *asymmetricMode {
			// print your own message for convenience,
			// because in asymmetric mode it is impossible to decrypt it
			timestamp := time.Now().Unix()
			from := crypto.PubkeyToAddress(asymKey.PublicKey)
			fmt.Printf("\n%d <%x>: %s\n", timestamp, from, s)
		}
	}
}

func sendFilesLoop() {
	for {
		s := scanLine("")
		if s == quitCommand {
			fmt.Println("Quit command received")
			close(done)
			break
		}
		b, err := ioutil.ReadFile(s)
		if err != nil {
			fmt.Printf(">>> Error: %s \n", err)
			continue
		} else {
			h := sendMsg(b)
			if (h == common.Hash{}) {
				fmt.Printf(">>> Error: message was not sent \n")
			} else {
				timestamp := time.Now().Unix()
				from := crypto.PubkeyToAddress(asymKey.PublicKey)
				fmt.Printf("\n%d <%x>: sent message with hash %x\n", timestamp, from, h)
			}
		}
	}
}

func scanLine(prompt string) string {
	if len(prompt) > 0 {
		fmt.Print(prompt)
	}
	txt, err := input.ReadString('\n')
	if err != nil {
		utils.Fatalf("input error: %s", err)
	}
	txt = strings.TrimRight(txt, "\n\r")
	return txt
}

func scanLineA(prompt string) *string {
	s := scanLine(prompt)
	return &s
}

func scanUint(prompt string) uint32 {
	s := scanLine(prompt)
	i, err := strconv.Atoi(s)
	if err != nil {
		utils.Fatalf("Fail to parse the lower time limit: %s", err)
	}
	return uint32(i)
}

func sendMsg(payload []byte) common.Hash {
	params := whisper.MessageParams{
		Src:      asymKey,
		Dst:      pub,
		KeySym:   symKey,
		Payload:  payload,
		Topic:    topic,
		TTL:      uint32(*argTTL),
		PoW:      *argPoW,
		WorkTime: uint32(*argWorkTime),
	}

	msg, err := whisper.NewSentMessage(&params)
	if err != nil {
		utils.Fatalf("failed to create new message: %s", err)
	}
	envelope, err := msg.Wrap(&params)
	if err != nil {
		fmt.Printf("failed to seal message: %v \n", err)
		return common.Hash{}
	}

	err = shh.Send(envelope)
	if err != nil {
		fmt.Printf("failed to send message: %v \n", err)
		return common.Hash{}
	}

	return envelope.Hash()
}

func messageLoop() {
	sf := shh.GetFilter(symFilterID)
	if sf == nil {
		utils.Fatalf("symmetric filter is not installed")
	}

	af := shh.GetFilter(asymFilterID)
	if af == nil {
		utils.Fatalf("asymmetric filter is not installed")
	}

	ticker := time.NewTicker(time.Millisecond * 50)

	for {
		select {
		case <-ticker.C:
			messages := sf.Retrieve()
			for _, msg := range messages {
				if *fileExMode || len(msg.Payload) > 2048 {
					writeMessageToFile(*argSaveDir, msg)
				} else {
					printMessageInfo(msg)
				}
			}

			messages = af.Retrieve()
			for _, msg := range messages {
				if *fileExMode || len(msg.Payload) > 2048 {
					writeMessageToFile(*argSaveDir, msg)
				} else {
					printMessageInfo(msg)
				}
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

func writeMessageToFile(dir string, msg *whisper.ReceivedMessage) {
	timestamp := fmt.Sprintf("%d", msg.Sent)
	name := fmt.Sprintf("%x", msg.EnvelopeHash)

	var address common.Address
	if msg.Src != nil {
		address = crypto.PubkeyToAddress(*msg.Src)
	}

	if whisper.IsPubKeyEqual(msg.Src, &asymKey.PublicKey) {
		// message from myself: don't save, only report
		fmt.Printf("\n%s <%x>: message received: '%s'\n", timestamp, address, name)
	} else if len(dir) > 0 {
		fullpath := filepath.Join(dir, name)
		err := ioutil.WriteFile(fullpath, msg.Payload, 0644)
		if err != nil {
			fmt.Printf("\n%s {%x}: message received but not saved: %s\n", timestamp, address, err)
		} else {
			fmt.Printf("\n%s {%x}: message received and saved as '%s' (%d bytes)\n", timestamp, address, name, len(msg.Payload))
		}
	} else {
		fmt.Printf("\n%s {%x}: big message received (%d bytes), but not saved: %s\n", timestamp, address, len(msg.Payload), name)
	}
}

func requestExpiredMessagesLoop() {
	var key, peerID []byte
	var timeLow, timeUpp uint32
	var t string
	var xt, empty whisper.TopicType

	keyID, err := shh.AddSymKeyFromPassword(msPassword)
	if err != nil {
		utils.Fatalf("Failed to create symmetric key for mail request: %s", err)
	}
	key, err = shh.GetSymKey(keyID)
	if err != nil {
		utils.Fatalf("Failed to save symmetric key for mail request: %s", err)
	}
	peerID = extractIDFromEnode(*argEnode)
	shh.AllowP2PMessagesFromPeer(peerID)

	for {
		timeLow = scanUint("Please enter the lower limit of the time range (unix timestamp): ")
		timeUpp = scanUint("Please enter the upper limit of the time range (unix timestamp): ")
		t = scanLine("Please enter the topic (hexadecimal): ")
		if len(t) >= whisper.TopicLength*2 {
			x, err := hex.DecodeString(t)
			if err != nil {
				utils.Fatalf("Failed to parse the topic: %s", err)
			}
			xt = whisper.BytesToTopic(x)
		}
		if timeUpp == 0 {
			timeUpp = 0xFFFFFFFF
		}

		data := make([]byte, 8+whisper.TopicLength)
		binary.BigEndian.PutUint32(data, timeLow)
		binary.BigEndian.PutUint32(data[4:], timeUpp)
		copy(data[8:], xt[:])
		if xt == empty {
			data = data[:8]
		}

		var params whisper.MessageParams
		params.PoW = *argServerPoW
		params.Payload = data
		params.KeySym = key
		params.Src = nodeid
		params.WorkTime = 5

		msg, err := whisper.NewSentMessage(&params)
		if err != nil {
			utils.Fatalf("failed to create new message: %s", err)
		}
		env, err := msg.Wrap(&params)
		if err != nil {
			utils.Fatalf("Wrap failed: %s", err)
		}

		err = shh.RequestHistoricMessages(peerID, env)
		if err != nil {
			utils.Fatalf("Failed to send P2P message: %s", err)
		}

		time.Sleep(time.Second * 5)
	}
}

func extractIDFromEnode(s string) []byte {
	n, err := discover.ParseNode(s)
	if err != nil {
		utils.Fatalf("Failed to parse enode: %s", err)
	}
	return n.ID[:]
}
