package main

//@NOTE SHYFT main func for api, sets up router and spins up a server
//to run server 'go run shyftRingWalletConn/*.go'
import (
  "net"
  "fmt"
  "os"
	//"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"bytes"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type Msg struct {
	Message string `json:"message"`
	HashedMessage string `json:"hashed_message"`
	Signature string `json:"signature"`
	Address string `json:"address"`
}

const (
	CONN_HOST = "localhost"
	CONN_PORT = "3333"
	CONN_TYPE = "tcp"
)

var testAddrHex = "14791697260E4c9A71f18484C9f997B308e59325"
var testPrivHex = "0123456789012345678901234567890123456789012345678901234567890123"

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
	// Read the incoming connection into the buffer.

	go func() {
		buf := make([]byte, 1024)
		msgBuf := make([]byte, 0)
		var prevMsg []byte
		var addressOfClient []byte
		var signatureFromClient []byte
		var msgFromClient []byte

		for {
			//var addressOfClient []byte
			//var msgFromClient []byte
			//var sigFromClient []byte
			msg, err := conn.Read(buf)

			if err == nil {
				fmt.Println("Message is ", buf[:msg])
				fmt.Println("The old msg buf is ", msgBuf)
				msgBuf = append(msgBuf, buf[:msg]...)
				fmt.Println("THE msgBuf is ")
				fmt.Println(msgBuf)
				//fmt.Println("Message is ", string(buf[:msg]))
			}
			index := bytes.IndexByte(msgBuf, 0x0a)
			for index != -1 {
				newMsg := msgBuf[:index]
				rest := msgBuf[(index + 1):len(msgBuf)]
				msgBuf = rest
				index = bytes.IndexByte(msgBuf, 0x0a)
				fmt.Println("new msg")
				fmt.Println(newMsg)
				fmt.Println(string(newMsg[:]))
				fmt.Println("prev message ", prevMsg)
				fmt.Println((prevMsg == nil))
				if prevMsg != nil {
					fmt.Println("not nil")
					s := string(prevMsg[:])
					if s == "-- ADDRESS --" {
						fmt.Println("the address should be ")
						fmt.Println(newMsg)
						addressOfClient = newMsg
					}
					if s == "-- SIGNATURE --" {
						signatureFromClient = newMsg
					}
					if s == "-- MESSAGE --" {
						msgFromClient = newMsg
					}
					prevMsg = nil
				} else {
					fmt.Println("nil")
					prevMsg = newMsg
					fmt.Println("prev message is in else block ", prevMsg)
					fmt.Println("index ", index)
					fmt.Println("msgBuf ", msgBuf)
					fmt.Println(bytes.IndexByte(msgBuf, 0x0a))

				}
			}

			if(addressOfClient != nil && signatureFromClient != nil && msgFromClient != nil ){
				fmt.Println("ALL COMPONENTS RECEIVED")
				msg := string(msgFromClient[:])
				//new_msg := signHash(msgFromClient)
				sig := string(signatureFromClient[:])
				addr := string(addressOfClient[:])
				fmt.Println(addr)
				fmt.Println(msg)
				fmt.Println(sig)

				var msgByteArr = []byte(msg)
				var hexMsg = hexutil.Encode(msgByteArr)
				fizz, err2 := hexutil.Decode(hexMsg)

				var sigByteArr, err3 = hexutil.Decode(sig)
				if err2 != nil {
					fmt.Println("the err2 is ")
					fmt.Println(err2)
				}
				if err3 != nil {
					fmt.Println("the err3 is ")
					fmt.Println(err3)
				}

				var sigHex = hexutil.Bytes(sigByteArr)
				sigHex[64] -= 27

				signedMsgHash := signHash(fizz)

				var rpk, err = crypto.Ecrecover(signedMsgHash, sigHex)
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
		}
	}()
	go func() {
		key, _ := crypto.HexToECDSA(testPrivHex)
		//addr := common.HexToAddress(testAddrHex)

		f_msg := "Hello World"
		first_message := []byte(f_msg)
		new_msg2 := crypto.Keccak256(first_message)
		fmt.Println("the hash is ", hexutil.Encode(new_msg2))

		//send_message := append(new_msg2, []byte{byte(10)}...)
		new_sig , err := crypto.Sign(new_msg2, key)
		if err != nil {
			fmt.Println("The crypto.Sign err is ", err)
		}
		hex_sig := hexutil.Encode(new_sig)
		fmt.Println("THE hex sig is ", hex_sig)

		conn.Write([]byte("Broadcasting Message"))
		conn.Write([]byte("\n"))
		conn.Write([]byte(f_msg))
		conn.Write([]byte("\n"))
		conn.Write(new_sig)
		conn.Write([]byte("\n"))
	}()
	//conn.Write([]byte{byte(0x0f)})

	// Send a response back to person contacting us.

	// Close the connection when you're done with it.
	//conn.Close()
}

func intArrToByteArr(foo []int) []byte {

	ret := []byte{}
	for _, value := range foo {
		ret = append(ret, byte(value))
	}
	return ret
}
