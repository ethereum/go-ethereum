package main

import (
	"os"
	"github.com/ethereum/go-ethereum/log"
)

type auditlogger struct{
	filename string
}
func (al auditlogger) append(text string) error{
	f, err := os.OpenFile(al.filename, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.Crit("Failed to open audit log","err", err)
		return err
	}
	defer f.Close()
	if _, err = f.WriteString(text); err != nil {
		log.Crit("Failed to write to audit log","err", err)
		return err
	}
}

