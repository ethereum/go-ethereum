package les

import (
	"sync"
	"strconv"
	"time"
	"log"

	"github.com/ethereum/go-ethereum/common/mclock"
	client "github.com/influxdata/influxdb/client/v2"
)

type influxLogger struct {
	clnt client.Client
	dBName   string
	username string
	password string
}

var iLogger *influxLogger
var once sync.Once

func (il *influxLogger) WriteData(msgCode uint64, reqCount uint64, cost uint64, actionTime mclock.AbsTime, peerId string) {
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  il.dBName,
		Precision: "s",
	})
	if err != nil {
		log.Fatal(err)
	}
	// Create a point and add to batch
	tags := map[string]string{"peer": "peer-" + peerId, "msgCode": "msgCode-" + strconv.Itoa(int(msgCode))}
	fields := map[string]interface{}{
		"msgCode":  msgCode,
		"reqCount": reqCount,
		"cost":     cost,
		"absTime":  actionTime,
	}
	pt, err := client.NewPoint("ETHLightStats", tags, fields, time.Now())
	if err != nil {
		log.Fatal(err)
	}
	bp.AddPoint(pt)

	// Write the batch
	if err := il.clnt.Write(bp); err != nil {
		log.Fatal("INFLUX WRITE FAILED: ", err)
	}
}

func GetInfluxLoggerInstance() *influxLogger {
	once.Do(func(){
		iLogger = createClient()
	})
	return iLogger
}

func createClient() *influxLogger {
	cli, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     "http://localhost:8086",
		Username: "",
		Password: "",
	})
	if err != nil{
		log.Fatal("CREATE INFLUX CLIENT FAILED: ", err)
	}

	return &influxLogger{
		dBName: "ethereumStatistics",
		username: "",
		password: "",
		clnt: cli,
	}
}