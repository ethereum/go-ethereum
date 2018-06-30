package main

//@NOTE SHYFT main func for api, sets up router and spins up a server
//to run server 'go run shyftRingWalletConn/*.go'
import (
  "net"
  "fmt"
  "os"
  "encoding/json"
  "github.com/ethereum/go-ethereum/crypto"
)

const (
	CONN_HOST = "localhost"
	CONN_PORT = "3333"
	CONN_TYPE = "tcp"
)

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
		var msg = []byte(dat["msg"].(string))
		var sig = []byte(dat["sig"].(string))
		new_msg := crypto.Keccak256(msg)
		var address, err = crypto.Ecrecover(new_msg, sig)
		if err != nil {
			fmt.Println("The error is ")
			fmt.Println(err)
		}
		s := string(address[:])
		fmt.Println("the address is ")
		fmt.Println(s)
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
