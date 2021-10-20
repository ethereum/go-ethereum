package vm

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/core/types"
	"os"
	"path"
	"strconv"
)

func ReceiptDumpLogger(blockNumber uint64, perFolder, perFile uint64, receipts types.Receipts) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get current work dir failed: %w", err)
	}

	logPath := path.Join(cwd, "receipts", strconv.FormatUint(blockNumber/perFolder, 10), strconv.FormatUint(blockNumber/perFile, 10)+".log")
	fmt.Printf("receipt path: %v, block: %v\n", logPath, blockNumber)
	if err := os.MkdirAll(path.Dir(logPath), 0755); err != nil {
		return fmt.Errorf("mkdir for all parents [%v] failed: %w", path.Dir(logPath), err)
	}

	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		return fmt.Errorf("create file %s failed: %w", logPath, err)
	}

	encoder := json.NewEncoder(file)
	for _, receipt := range receipts {
		for _, log := range receipt.Logs {
			err := encoder.Encode(log)
			if err != nil {
				return fmt.Errorf("encode log failed: %w", err)
			}
		}
	}
	return nil
}
