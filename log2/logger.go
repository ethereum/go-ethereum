package log2

import (
	"encoding/json"
	"fmt"
	"os"
)

var file *os.File

type TimingLog struct {
	From              string
	To                string
	TimeCost          int64
	Timestamp         int64
	DataFirst4ByteHex string
}

func InitOutputFile(outputFile string) {
	if outputFile == "" {
		fmt.Fprintln(os.Stderr, "You must specify the log file, e.g. \"geth --timing.output=/path/to/file.txt\"")
		os.Exit(1)
	}
	var err error
	file, err = os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to create log file for timing.")
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	_, err = file.WriteString("Timing log initialized.\n")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to write log file for timing.")
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	err = file.Sync()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to write log file for timing.")
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func Record(timingLog TimingLog) error {
	b, err := json.Marshal(timingLog)
	if err != nil {
		return err
	}
	b = append(b, '\n')

	_, err = file.Write(b)
	if err != nil {
		return err
	}

	err = file.Sync()
	return err
}
