package main

import (
	"bufio"
	"fmt"
	"io"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
)

type AuditLogger struct {
	writer *bufio.Writer
}

func (l AuditLogger) Store(record *rpc.RPCInvocationRecord) {
	l.writer.WriteString(fmt.Sprintf("%v\n%v\n", time.Now().Format(time.RFC3339), record.Method))
	for i, arg := range record.Args {
		l.writer.WriteString(fmt.Sprintf("\t%d: %v\n", i, arg))
	}
	l.writer.WriteString(fmt.Sprintf("%v\n", record.Response))
	l.writer.Flush()
}

func NewAuditLogger(writer io.Writer) *AuditLogger {
	return &AuditLogger{bufio.NewWriter(writer)}
}
