package eth

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/ethutil"
)

func WritePeers(path string, addresses []string) {
	if len(addresses) > 0 {
		data, _ := json.MarshalIndent(addresses, "", "    ")
		ethutil.WriteFile(path, data)
	}
}

func ReadPeers(path string) (ips []string, err error) {
	var data string
	data, err = ethutil.ReadAllFile(path)
	if err != nil {
		json.Unmarshal([]byte(data), &ips)
	}
	return
}
