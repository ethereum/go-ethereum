package ethconfig

import (
    "encoding/json"
    "io/ioutil"
	"net/http"
    "strings"

    "github.com/ethereum/go-ethereum/log"
)

type PeriConfig struct {
    Active                  bool
    Targeted                bool
    Period                  uint64
    ReplaceRatio            float64
    DialRatio               int
    MinInbound              int
    MaxDelayPenalty         uint64
    MaxDeliveryTolerance    int64

	ObservedTxRatio         int

	ShowTxDelivery          bool // Controls whether the console prints all txs

	TargetAccountList       []string
	NoPeerIPList            []string
	NoDropList              []string
}

var SelfIP string

type IP struct {
	Query string
}

func init() {
	req, err := http.Get("http://ip-api.com/json/")
	if err != nil {
		panic(err)
	}

	defer req.Body.Close()

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		panic(err)
	}

	var ip IP
	json.Unmarshal(body, &ip)

	SelfIP = ip.Query

	log.Info("Initialized Perigee Config, detected self IP = " + SelfIP)
}

func NewPeriConfig(path string) (*PeriConfig, error) {
	pcfg := &PeriConfig{}

	if path == "" {
		pcfg = &PeriConfig{
			Active:               false,
			Period:               0,
			ReplaceRatio:         0.,
			MaxDelayPenalty:      0,
			MaxDeliveryTolerance: 10000,
			ObservedTxRatio:      0,
			DialRatio:            3,
			MinInbound:           0,
			ShowTxDelivery:       false,
			TargetAccountList:    []string{},
			NoPeerIPList:         []string{},
			NoDropList:           []string{},
		}
	} else {
		file, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal([]byte(file), pcfg)
		if err != nil {
			return nil, err
		}
	}

	out, err := json.MarshalIndent(pcfg, "  ", "    ")
	if err != nil {
		panic(err)
	}

	log.Info("Perigee config: " + string(out))
	return pcfg, nil
}

func (pcfg PeriConfig) IsBanned(id string) bool {
    //Check for self
	for _, forbidden := range pcfg.NoPeerIPList {
		if strings.Contains(id, forbidden) {
			return true
		}
	}
	return false
}
