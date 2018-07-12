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

var testAddrHex = "970e8128ab834e8eac17ab8e3812f010678cf791"
var testPrivHex = "289c2857d4598e37fb9647507e47a309d6133539bf21a8b9cb6df88fd5232032"

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

		var msg = dat["msg"].(string)
		var sig = dat["sig"].(string)
		fmt.Println("the first sig is ")
		var new_byte_array = []byte(msg)
		var bazz = hexutil.Encode(new_byte_array)
		fizz, err2 := hexutil.Decode(bazz)

		var new_sig_byte_array, err3 = hexutil.Decode(sig)
		if err2 != nil {
			fmt.Println("the err2 is ")
			fmt.Println(err2)
		}
		if err3 != nil {
			fmt.Println("the err3 is ")
			fmt.Println(err3)
		}

		var buzz = hexutil.Bytes(new_sig_byte_array)
		buzz[64] -= 27

		new_msg := signHash(fizz)

		var rpk, err = crypto.Ecrecover(new_msg, buzz)
		if err != nil {
			fmt.Println("The error is ")
			fmt.Println(err)
		}

		pubKey := crypto.ToECDSAPub(rpk)
		recoveredAddr := crypto.PubkeyToAddress(*pubKey)
		fmt.Println("the address is ")
		//fmt.Println(recoveredAddr)
		fmt.Println(recoveredAddr.Hex())

	}
	if err != nil {
		fmt.Println("Error reading:", err.Error())
	}
	// Send a response back to person contacting us.
	conn.Write([]byte("Message received."))
	// Close the connection when you're done with it.
	//conn.Close()
}