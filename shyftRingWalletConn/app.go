package main

//@NOTE SHYFT main func for api, sets up router and spins up a server
//to run server 'go run shyftRingWalletConn/*.go'
import (
  "net"
  "fmt"
  "os"
  "encoding/json"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

const (
	CONN_HOST = "localhost"
	CONN_PORT = "3333"
	CONN_TYPE = "tcp"
)

// This gives context to the signed message and prevents signing of transactions.
func signHash(data []byte) []byte {
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data)
	return crypto.Keccak256([]byte(msg))
}

func main() {

	l, err := net.Listen(CONN_TYPE, CONN_HOST+":"+CONN_PORT)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	defer l.Close()

	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		// Handle connections in a new goroutine.
		go handleRequest(conn)
	}
}

// Handles incoming requests.
func handleRequest(conn net.Conn) {
	// Make a buffer to hold incoming data.
	buf := make([]byte, 1024)
	// Read the incoming connection into the buffer.
	msg, err := conn.Read(buf)

	if err == nil {
		fmt.Println("Message is ", string(buf[:msg]))
		var dat map[string]interface{}

		if err := json.Unmarshal(buf[:msg], &dat); err != nil {
			panic(err)
		}
		fmt.Println(dat["address"])
		fmt.Println(dat["msg"])
		fmt.Println(dat["sig"])
		//Ecrecover takes 2 args, bytes[] and bytes[]
		// we need to pass in the hash of the message as bytes
		// and the bytes array of the signature
		var msg = dat["msg"].(string)
		var sig = dat["sig"].(string)
		fmt.Println("the first sig is ")
		fmt.Println(sig)
		var new_byte_array, err2 = hexutil.Decode(msg)
		var new_sig_byte_array, err3 = hexutil.Decode(sig)
		if err2 != nil {
			fmt.Println("the err2 is ")
			fmt.Println(err2)
		}
		if err3 != nil {
			fmt.Println("the err3 is ")
			fmt.Println(err3)
		}
		var fizz = hexutil.Bytes(new_byte_array)
		var buzz = hexutil.Bytes(new_sig_byte_array)
		buzz[64] -= 27
		fmt.Println("that bytes is")
		fmt.Println(fizz)
		//var bytes_msg = []byte(msg)
		//var foo, err = hex.DecodeString(msg)
		//fmt.Println("msg is ")
		//fmt.Println(msg)
		//fmt.Println("bytes_msg is ")
		//fmt.Println(bytes_msg)

		//var sig = dat["sig"].(string)
		//fmt.Println("The sig is ")
		//fmt.Println(sig)
		//decoded, er := hex.DecodeString(sig)
		//if er != nil {
		//	log.Fatal(err)
		//}
		new_msg := signHash(fizz)
		fmt.Println("the new_msg is ")
		fmt.Println(new_msg)
		fmt.Println("the buzz is ")
		fmt.Println(buzz)

		var rpk, err = crypto.Ecrecover(new_msg, buzz)
		if err != nil {
			fmt.Println("The error is ")
			fmt.Println(err)
		}

		pubKey := crypto.ToECDSAPub(rpk)
		recoveredAddr := crypto.PubkeyToAddress(*pubKey)
		fmt.Println("the address is ")
		fmt.Println(recoveredAddr)
		fmt.Println(recoveredAddr.Hex())
		//res, _ := new_add.MarshalText()
		//fmt.Println(hexutil.Encode(address))
		//s := string(address[:])
		//
		//fmt.Println(s)

		//fmt.Println(address)
		//fmt.Println("Message is ", msg)
	}
	if err != nil {
		fmt.Println("Error reading:", err.Error())
	}
	// Send a response back to person contacting us.
	conn.Write([]byte("Message received."))
	// Close the connection when you're done with it.
	conn.Close()
}