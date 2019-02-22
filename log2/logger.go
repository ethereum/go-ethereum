package log2

import (
	"encoding/hex"
	"encoding/json"
	"go-ethereum-timing/core"
	"log"
	"os"
	"time"
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
func Record(message core.Message, timeCost time.Duration) error {
	timingLog := TimingLog{
		From:              message.From().String(),
		To:                message.From().String(),
		TimeCost:          timeCost.Nanoseconds(),
		DataFirst4ByteHex: hex.EncodeToString(message.Data()[0:4]),
		Timestamp:         time.Now().Unix(),
	}
	b, err := json.Marshal(timingLog)
	if err != nil {
		return err
	}
	b = append(b, '\n')

	_, err = file.Write(b)
	return err
}
