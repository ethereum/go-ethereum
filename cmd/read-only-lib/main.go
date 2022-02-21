package main

import (
	"C"

	"github.com/ethereum/go-ethereum/internal/debug"
)
import (
	"context"
	"os"
	"unsafe"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
)

var stack *node.Node

//export open_database
func open_database(datadir *C.char) C.int {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(0)
	log.Root().SetHandler(glogger)
	if stack != nil {
		return C.int(-1)
	}
	go_datadir := C.GoString(datadir)
	stack, _ = makeReadOnlyNode(go_datadir)
	if err := startNode(stack, false); err != nil {
		return C.int(-1)
	}
	return C.int(0)
}

//export wrapper_call
func wrapper_call(cargs *C.char, clen C.int) (*C.char, C.int) {
	rawData := C.GoBytes(unsafe.Pointer(cargs), clen)
	server, _ := stack.RPCHandler()
	msg, _ := rpc.ParseMessage(rawData)
	var test = rpc.NewFuncCodec(nil, nil, nil)
	h := rpc.NewHandler(context.Background(), test, server.Services())
	tmp := h.HandleCallMsg(rpc.DefaultCallProc(), msg[0]).String()
	res := C.CString(tmp)
	return res, C.int(len(tmp))
}

//export close_database
func close_database() {
	stack.Close()
}

func startNode(stack *node.Node, isConsole bool) error {
	debug.Memsize.Add("node", stack)
	StartNode(stack)
	return nil
}

func main() {

}
