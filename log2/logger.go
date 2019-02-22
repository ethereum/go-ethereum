package log2

import (
	"encoding/json"
	"log"
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
	var err error
	file, err = os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalln("Fail to create log file for timing.")
	}
}
func Record(timingLog TimingLog) error {
	b, err := json.Marshal(timingLog)
	if err != nil {
		return err
	}
	b = append(b, '\n')

	_, err = file.Write(b)
	return err
}
