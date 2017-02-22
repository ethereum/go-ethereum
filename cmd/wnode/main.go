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
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
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
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
	"golang.org/x/crypto/pbkdf2"
)

const quitCommand = "~Q"
const symKeyName = "da919ea33001b04dfc630522e33078ec0df11"

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
	symKey     []byte
	pub        *ecdsa.PublicKey
	asymKey    *ecdsa.PrivateKey
	nodeid     *ecdsa.PrivateKey
	topic      whisper.TopicType
	filterID   string
	symPass    string
	msPassword string
)

// cmd arguments
var (
	echoMode       = flag.Bool("e", false, "echo mode: prints some arguments for diagnostics")
	bootstrapMode  = flag.Bool("b", false, "boostrap node: don't actively connect to peers, wait for incoming connections")
	forwarderMode  = flag.Bool("f", false, "forwarder mode: only forward messages, neither send nor decrypt messages")
	mailServerMode = flag.Bool("s", false, "mail server mode: delivers expired messages on demand")
	requestMail    = flag.Bool("r", false, "request expired messages from the bootstrap server")
	asymmetricMode = flag.Bool("a", false, "use asymmetric encryption")
	testMode       = flag.Bool("t", false, "use of predefined parameters for diagnostics")
	generateKey    = flag.Bool("k", false, "generate and show the private key")

	argVerbosity = flag.Int("verbosity", int(log.LvlWarn), "log verbosity level")
	argTTL       = flag.Uint("ttl", 30, "time-to-live for messages in seconds")
	argWorkTime  = flag.Uint("work", 5, "work time in seconds")
	argPoW       = flag.Float64("pow", whisper.MinimumPoW, "PoW for normal messages in float format (e.g. 2.7)")
	argServerPoW = flag.Float64("mspow", whisper.MinimumPoW, "PoW requirement for Mail Server request")

	argIP     = flag.String("ip", "", "IP address and port of this node (e.g. 127.0.0.1:30303)")
	argPub    = flag.String("pub", "", "public key for asymmetric encryption")
	argDBPath = flag.String("dbpath", "", "path to the server's DB directory")
	argIDFile = flag.String("idfile", "", "file name with node id (private key)")
	argEnode  = flag.String("boot", "", "bootstrap node you want to connect to (e.g. enode://e454......08d50@52.176.211.200:16428)")
	argTopic  = flag.String("topic", "", "topic in hexadecimal format (e.g. 70a4beef)")
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
			log.Crit(fmt.Sprintf("Failed to load file [%s]: %s.", *argIDFile, err))
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
			log.Crit(fmt.Sprintf("Failed to parse the topic: %s", err))
		}
		topic = whisper.BytesToTopic(x)
	}

	if *asymmetricMode && len(*argPub) > 0 {
		pub = crypto.ToECDSAPub(common.FromHex(*argPub))
		if !isKeyValid(pub) {
			log.Crit(fmt.Sprintf("invalid public key"))
		}
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
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(*argVerbosity), log.StreamHandler(os.Stderr, log.TerminalFormat())))

	done = make(chan struct{})
	var peers []*discover.Node
	var err error

	if *generateKey {
		key, err := crypto.GenerateKey()
		if err != nil {
			log.Crit(fmt.Sprintf("Failed to generate private key: %s", err))
		}
		k := hex.EncodeToString(crypto.FromECDSA(key))
		fmt.Printf("Random private key: %s \n", k)
		os.Exit(0)
	}

	if *testMode {
		symPass = "wwww" // ascii code: 0x77777777
		msPassword = "mail server test password"
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

	if *mailServerMode {
		if len(msPassword) == 0 {
			msPassword, err = console.Stdin.PromptPassword("Please enter the Mail Server password: ")
			if err != nil {
				log.Crit(fmt.Sprintf("Failed to read Mail Server password: %s", err))
			}
		}
		shh = whisper.New()
		shh.RegisterServer(&mailServer)
		mailServer.Init(shh, *argDBPath, msPassword, *argServerPoW)
	} else {
		shh = whisper.New()
	}

	asymKey = shh.NewIdentity()
	if nodeid == nil {
		nodeid = shh.NewIdentity()
	}

	maxPeers := 80
	if *bootstrapMode {
		maxPeers = 800
	}

	server = &p2p.Server{
		Config: p2p.Config{
			PrivateKey:     nodeid,
			MaxPeers:       maxPeers,
			Name:           common.MakeName("whisper-go", "5.0"),
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
		log.Crit(fmt.Sprintf("Failed to start Whisper peer: %s.", err))
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
			pub = crypto.ToECDSAPub(common.FromHex(s))
			if !isKeyValid(pub) {
				log.Crit(fmt.Sprintf("Error: invalid public key"))
			}
		}
	}

	if *requestMail {
		p2pAccept = true
		if len(msPassword) == 0 {
			msPassword, err = console.Stdin.PromptPassword("Please enter the Mail Server password: ")
			if err != nil {
				log.Crit(fmt.Sprintf("Failed to read Mail Server password: %s", err))
			}
		}
	}

	if !*asymmetricMode && !*forwarderMode {
		if len(symPass) == 0 {
			symPass, err = console.Stdin.PromptPassword("Please enter the password: ")
			if err != nil {
				log.Crit(fmt.Sprintf("Failed to read passphrase: %v", err))
			}
		}

		shh.AddSymKey(symKeyName, []byte(symPass))
		symKey = shh.GetSymKey(symKeyName)
		if len(*argTopic) == 0 {
			generateTopic([]byte(symPass))
		}
	}

	if *mailServerMode {
		if len(*argDBPath) == 0 {
			argDBPath = scanLineA("Please enter the path to DB file: ")
		}
	}

	filter := whisper.Filter{
		KeySym:    symKey,
		KeyAsym:   asymKey,
		Topics:    []whisper.TopicType{topic},
		AcceptP2P: p2pAccept,
	}
	filterID, err = shh.Watch(&filter)
	if err != nil {
		utils.Fatalf("Failed to install filter: %s", err)
	}
	fmt.Printf("Filter is configured for the topic: %x \n", topic)
}

func generateTopic(password []byte) {
	x := pbkdf2.Key(password, password, 8196, 128, sha512.New)
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
				log.Crit(fmt.Sprintf("Timeout expired, failed to connect"))
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

func scanLine(prompt string) string {
	if len(prompt) > 0 {
		fmt.Print(prompt)
	}
	txt, err := input.ReadString('\n')
	if err != nil {
		log.Crit(fmt.Sprintf("input error: %s", err))
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
		log.Crit(fmt.Sprintf("Fail to parse the lower time limit: %s", err))
	}
	return uint32(i)
}

func sendMsg(payload []byte) {
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
		log.Crit(fmt.Sprintf("filter is not installed"))
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
	var timeLow, timeUpp uint32
	var t string
	var xt, empty whisper.TopicType

	err := shh.AddSymKey(mailserver.MailServerKeyName, []byte(msPassword))
	if err != nil {
		log.Crit(fmt.Sprintf("Failed to create symmetric key for mail request: %s", err))
	}
	key = shh.GetSymKey(mailserver.MailServerKeyName)
	peerID = extractIdFromEnode(*argEnode)
	shh.MarkPeerTrusted(peerID)

	for {
		timeLow = scanUint("Please enter the lower limit of the time range (unix timestamp): ")
		timeUpp = scanUint("Please enter the upper limit of the time range (unix timestamp): ")
		t = scanLine("Please enter the topic (hexadecimal): ")
		if len(t) >= whisper.TopicLength*2 {
			x, err := hex.DecodeString(t)
			if err != nil {
				log.Crit(fmt.Sprintf("Failed to parse the topic: %s", err))
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

		msg := whisper.NewSentMessage(&params)
		env, err := msg.Wrap(&params)
		if err != nil {
			log.Crit(fmt.Sprintf("Wrap failed: %s", err))
		}

		err = shh.RequestHistoricMessages(peerID, env)
		if err != nil {
			log.Crit(fmt.Sprintf("Failed to send P2P message: %s", err))
		}

		time.Sleep(time.Second * 5)
	}
}

func extractIdFromEnode(s string) []byte {
	n, err := discover.ParseNode(s)
	if err != nil {
		log.Crit(fmt.Sprintf("Failed to parse enode: %s", err))
		return nil
	}
	return n.ID[:]
}
