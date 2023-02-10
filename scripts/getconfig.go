package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/internal/cli/server"
)

// YesFV: Both, Flags and their values has changed
// YesF:  Only the Flag has changed, not their value
var flagMap = map[string][]string{
	"networkid":                        {"notABoolFlag", "YesFV"},
	"miner.gastarget":                  {"notABoolFlag", "No"},
	"pprof":                            {"BoolFlag", "No"},
	"pprof.port":                       {"notABoolFlag", "No"},
	"pprof.addr":                       {"notABoolFlag", "No"},
	"pprof.memprofilerate":             {"notABoolFlag", "No"},
	"pprof.blockprofilerate":           {"notABoolFlag", "No"},
	"pprof.cpuprofile":                 {"notABoolFlag", "No"},
	"jsonrpc.corsdomain":               {"notABoolFlag", "YesF"},
	"jsonrpc.vhosts":                   {"notABoolFlag", "YesF"},
	"http.modules":                     {"notABoolFlag", "YesF"},
	"ws.modules":                       {"notABoolFlag", "YesF"},
	"config":                           {"notABoolFlag", "No"},
	"datadir.ancient":                  {"notABoolFlag", "No"},
	"datadir.minfreedisk":              {"notABoolFlag", "No"},
	"usb":                              {"BoolFlag", "No"},
	"pcscdpath":                        {"notABoolFlag", "No"},
	"mainnet":                          {"BoolFlag", "No"},
	"goerli":                           {"BoolFlag", "No"},
	"bor-mumbai":                       {"BoolFlag", "No"},
	"bor-mainnet":                      {"BoolFlag", "No"},
	"rinkeby":                          {"BoolFlag", "No"},
	"ropsten":                          {"BoolFlag", "No"},
	"sepolia":                          {"BoolFlag", "No"},
	"kiln":                             {"BoolFlag", "No"},
	"exitwhensynced":                   {"BoolFlag", "No"},
	"light.serve":                      {"notABoolFlag", "No"},
	"light.ingress":                    {"notABoolFlag", "No"},
	"light.egress":                     {"notABoolFlag", "No"},
	"light.maxpeers":                   {"notABoolFlag", "No"},
	"ulc.servers":                      {"notABoolFlag", "No"},
	"ulc.fraction":                     {"notABoolFlag", "No"},
	"ulc.onlyannounce":                 {"BoolFlag", "No"},
	"light.nopruning":                  {"BoolFlag", "No"},
	"light.nosyncserve":                {"BoolFlag", "No"},
	"dev.gaslimit":                     {"notABoolFlag", "No"},
	"ethash.cachedir":                  {"notABoolFlag", "No"},
	"ethash.cachesinmem":               {"notABoolFlag", "No"},
	"ethash.cachesondisk":              {"notABoolFlag", "No"},
	"ethash.cacheslockmmap":            {"BoolFlag", "No"},
	"ethash.dagdir":                    {"notABoolFlag", "No"},
	"ethash.dagsinmem":                 {"notABoolFlag", "No"},
	"ethash.dagsondisk":                {"notABoolFlag", "No"},
	"ethash.dagslockmmap":              {"BoolFlag", "No"},
	"fdlimit":                          {"notABoolFlag", "No"},
	"signer":                           {"notABoolFlag", "No"},
	"authrpc.jwtsecret":                {"notABoolFlag", "No"},
	"authrpc.addr":                     {"notABoolFlag", "No"},
	"authrpc.port":                     {"notABoolFlag", "No"},
	"authrpc.vhosts":                   {"notABoolFlag", "No"},
	"graphql.corsdomain":               {"notABoolFlag", "No"},
	"graphql.vhosts":                   {"notABoolFlag", "No"},
	"rpc.evmtimeout":                   {"notABoolFlag", "No"},
	"rpc.allow-unprotected-txs":        {"BoolFlag", "No"},
	"jspath":                           {"notABoolFlag", "No"},
	"exec":                             {"notABoolFlag", "No"},
	"preload":                          {"notABoolFlag", "No"},
	"discovery.dns":                    {"notABoolFlag", "No"},
	"netrestrict":                      {"notABoolFlag", "No"},
	"nodekey":                          {"notABoolFlag", "No"},
	"nodekeyhex":                       {"notABoolFlag", "No"},
	"miner.threads":                    {"notABoolFlag", "No"},
	"miner.notify":                     {"notABoolFlag", "No"},
	"miner.notify.full":                {"BoolFlag", "No"},
	"miner.recommit":                   {"notABoolFlag", "No"},
	"miner.noverify":                   {"BoolFlag", "No"},
	"vmdebug":                          {"BoolFlag", "No"},
	"fakepow":                          {"BoolFlag", "No"},
	"nocompaction":                     {"BoolFlag", "No"},
	"metrics.addr":                     {"notABoolFlag", "No"},
	"metrics.port":                     {"notABoolFlag", "No"},
	"whitelist":                        {"notABoolFlag", "No"},
	"snapshot":                         {"BoolFlag", "YesF"},
	"bloomfilter.size":                 {"notABoolFlag", "No"},
	"bor.logs":                         {"BoolFlag", "No"},
	"override.arrowglacier":            {"notABoolFlag", "No"},
	"override.terminaltotaldifficulty": {"notABoolFlag", "No"},
	"verbosity":                        {"notABoolFlag", "YesFV"},
	"ws.origins":                       {"notABoolFlag", "No"},
}

// map from cli flags to corresponding toml tags
var nameTagMap = map[string]string{
	"chain":                   "chain",
	"identity":                "identity",
	"verbosity":               "verbosity",
	"datadir":                 "datadir",
	"keystore":                "keystore",
	"syncmode":                "syncmode",
	"gcmode":                  "gcmode",
	"eth.requiredblocks":      "eth.requiredblocks",
	"0-snapshot":              "snapshot",
	"\"bor.logs\"":            "bor.logs",
	"url":                     "bor.heimdall",
	"\"bor.without\"":         "bor.withoutheimdall",
	"grpc-address":            "bor.heimdallgRPC",
	"\"bor.runheimdall\"":     "bor.runheimdall",
	"\"bor.runheimdallargs\"": "bor.runheimdallargs",
	"locals":                  "txpool.locals",
	"nolocals":                "txpool.nolocals",
	"journal":                 "txpool.journal",
	"rejournal":               "txpool.rejournal",
	"pricelimit":              "txpool.pricelimit",
	"pricebump":               "txpool.pricebump",
	"accountslots":            "txpool.accountslots",
	"globalslots":             "txpool.globalslots",
	"accountqueue":            "txpool.accountqueue",
	"globalqueue":             "txpool.globalqueue",
	"lifetime":                "txpool.lifetime",
	"mine":                    "mine",
	"etherbase":               "miner.etherbase",
	"extradata":               "miner.extradata",
	"gaslimit":                "miner.gaslimit",
	"gasprice":                "miner.gasprice",
	"ethstats":                "ethstats",
	"blocks":                  "gpo.blocks",
	"percentile":              "gpo.percentile",
	"maxprice":                "gpo.maxprice",
	"ignoreprice":             "gpo.ignoreprice",
	"cache":                   "cache",
	"1-database":              "cache.database",
	"trie":                    "cache.trie",
	"trie.journal":            "cache.journal",
	"trie.rejournal":          "cache.rejournal",
	"gc":                      "cache.gc",
	"1-snapshot":              "cache.snapshot",
	"noprefetch":              "cache.noprefetch",
	"preimages":               "cache.preimages",
	"txlookuplimit":           "txlookuplimit",
	"gascap":                  "rpc.gascap",
	"txfeecap":                "rpc.txfeecap",
	"ipcdisable":              "ipcdisable",
	"ipcpath":                 "ipcpath",
	"1-corsdomain":            "http.corsdomain",
	"1-vhosts":                "http.vhosts",
	"origins":                 "ws.origins",
	"3-corsdomain":            "graphql.corsdomain",
	"3-vhosts":                "graphql.vhosts",
	"1-enabled":               "http",
	"1-host":                  "http.addr",
	"1-port":                  "http.port",
	"1-prefix":                "http.rpcprefix",
	"1-api":                   "http.api",
	"2-enabled":               "ws",
	"2-host":                  "ws.addr",
	"2-port":                  "ws.port",
	"2-prefix":                "ws.rpcprefix",
	"2-api":                   "ws.api",
	"3-enabled":               "graphql",
	"bind":                    "bind",
	"0-port":                  "port",
	"bootnodes":               "bootnodes",
	"maxpeers":                "maxpeers",
	"maxpendpeers":            "maxpendpeers",
	"nat":                     "nat",
	"nodiscover":              "nodiscover",
	"v5disc":                  "v5disc",
	"metrics":                 "metrics",
	"expensive":               "metrics.expensive",
	"influxdb":                "metrics.influxdb",
	"endpoint":                "metrics.influxdb.endpoint",
	"0-database":              "metrics.influxdb.database",
	"username":                "metrics.influxdb.username",
	"0-password":              "metrics.influxdb.password",
	"tags":                    "metrics.influxdb.tags",
	"prometheus-addr":         "metrics.prometheus-addr",
	"opencollector-endpoint":  "metrics.opencollector-endpoint",
	"influxdbv2":              "metrics.influxdbv2",
	"token":                   "metrics.influxdb.token",
	"bucket":                  "metrics.influxdb.bucket",
	"organization":            "metrics.influxdb.organization",
	"unlock":                  "unlock",
	"1-password":              "password",
	"allow-insecure-unlock":   "allow-insecure-unlock",
	"lightkdf":                "lightkdf",
	"disable-bor-wallet":      "disable-bor-wallet",
	"addr":                    "grpc.addr",
	"dev":                     "dev",
	"period":                  "dev.period",
}

var removedFlagsAndValues = map[string]string{}

var replacedFlagsMapFlagAndValue = map[string]map[string]map[string]string{
	"networkid": {
		"flag": {
			"networkid": "chain",
		},
		"value": {
			"'137'":   "mainnet",
			"137":     "mainnet",
			"'80001'": "mumbai",
			"80001":   "mumbai",
		},
	},
	"verbosity": {
		"flag": {
			"verbosity": "verbosity",
		},
		"value": {
			"0": "CRIT",
			"1": "ERROR",
			"2": "WARN",
			"3": "INFO",
			"4": "DEBUG",
			"5": "TRACE",
		},
	},
}

// Do not remove
var replacedFlagsMapFlag = map[string]string{}

var currentBoolFlags = []string{
	"snapshot",
	"bor.logs",
	"bor.withoutheimdall",
	"bor.runheimdall",
	"txpool.nolocals",
	"mine",
	"cache.noprefetch",
	"cache.preimages",
	"ipcdisable",
	"http",
	"ws",
	"graphql",
	"nodiscover",
	"v5disc",
	"metrics",
	"metrics.expensive",
	"metrics.influxdb",
	"metrics.influxdbv2",
	"allow-insecure-unlock",
	"lightkdf",
	"disable-bor-wallet",
	"dev",
}

func contains(s []string, str string) (bool, int) {
	for ind, v := range s {
		if v == str || v == "-"+str || v == "--"+str {
			return true, ind
		}
	}

	return false, -1
}

func indexOf(s []string, str string) int {
	for k, v := range s {
		if v == str || v == "-"+str || v == "--"+str {
			return k
		}
	}

	return -1
}

func remove1(s []string, idx int) []string {
	removedFlagsAndValues[s[idx]] = ""
	return append(s[:idx], s[idx+1:]...)
}

func remove2(s []string, idx int) []string {
	removedFlagsAndValues[s[idx]] = s[idx+1]
	return append(s[:idx], s[idx+2:]...)
}

func checkFlag(allFlags []string, checkFlags []string) []string {
	outOfDateFlags := []string{}

	for _, flag := range checkFlags {
		t1, _ := contains(allFlags, flag)
		if !t1 {
			outOfDateFlags = append(outOfDateFlags, flag)
		}
	}

	return outOfDateFlags
}

func checkFileExists(path string) bool {
	_, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		fmt.Println("WARN: File does not exist", path)
		return false
	} else {
		return true
	}
}

func writeTempStaticJSON(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}

	var conf interface{}
	if err := json.Unmarshal(data, &conf); err != nil {
		log.Fatal(err)
	}

	temparr := []string{}
	for _, item := range conf.([]interface{}) {
		temparr = append(temparr, item.(string))
	}

	// write to a temp file
	err = os.WriteFile("./tempStaticNodes.json", []byte(strings.Join(temparr, "\", \"")), 0600)
	if err != nil {
		log.Fatal(err)
	}
}

func writeTempStaticTrustedTOML(path string) {
	data, err := toml.LoadFile(path)
	if err != nil {
		log.Fatal(err)
	}

	if data.Has("Node.P2P.StaticNodes") {
		temparr := []string{}
		for _, item := range data.Get("Node.P2P.StaticNodes").([]interface{}) {
			temparr = append(temparr, item.(string))
		}

		err = os.WriteFile("./tempStaticNodes.toml", []byte(strings.Join(temparr, "\", \"")), 0600)
		if err != nil {
			log.Fatal(err)
		}
	}

	if data.Has("Node.P2P.TrustedNodes") {
		temparr := []string{}
		for _, item := range data.Get("Node.P2P.TrustedNodes").([]interface{}) {
			temparr = append(temparr, item.(string))
		}

		err = os.WriteFile("./tempTrustedNodes.toml", []byte(strings.Join(temparr, "\", \"")), 0600)
		if err != nil {
			log.Fatal(err)
		}
	}

	if data.Has("Node.HTTPTimeouts.ReadTimeout") {
		err = os.WriteFile("./tempHTTPTimeoutsReadTimeout.toml", []byte(data.Get("Node.HTTPTimeouts.ReadTimeout").(string)), 0600)
		if err != nil {
			log.Fatal(err)
		}
	}

	if data.Has("Node.HTTPTimeouts.WriteTimeout") {
		err = os.WriteFile("./tempHTTPTimeoutsWriteTimeout.toml", []byte(data.Get("Node.HTTPTimeouts.WriteTimeout").(string)), 0600)
		if err != nil {
			log.Fatal(err)
		}
	}

	if data.Has("Node.HTTPTimeouts.IdleTimeout") {
		err = os.WriteFile("./tempHTTPTimeoutsIdleTimeout.toml", []byte(data.Get("Node.HTTPTimeouts.IdleTimeout").(string)), 0600)
		if err != nil {
			log.Fatal(err)
		}
	}

	if data.Has("Eth.TrieTimeout") {
		err = os.WriteFile("./tempHTTPTimeoutsTrieTimeout.toml", []byte(data.Get("Eth.TrieTimeout").(string)), 0600)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func getStaticTrustedNodes(args []string) {
	// if config flag is passed, it should be only a .toml file
	t1, t2 := contains(args, "config")
	// nolint: nestif
	if t1 {
		path := args[t2+1]
		if !checkFileExists(path) {
			return
		}

		if path[len(path)-4:] == "toml" {
			writeTempStaticTrustedTOML(path)
		} else {
			fmt.Println("only TOML config file is supported through CLI")
		}
	} else {
		path := "./static-nodes.json"
		if !checkFileExists(path) {
			return
		}
		writeTempStaticJSON(path)
	}
}

func getFlagsToCheck(args []string) []string {
	flagsToCheck := []string{}

	for _, item := range args {
		if strings.HasPrefix(item, "-") {
			if item[1] == '-' {
				temp := item[2:]
				flagsToCheck = append(flagsToCheck, temp)
			} else {
				temp := item[1:]
				flagsToCheck = append(flagsToCheck, temp)
			}
		}
	}

	return flagsToCheck
}

func getFlagType(flag string) string {
	return flagMap[flag][0]
}

func updateArgsClean(args []string, outOfDateFlags []string) []string {
	updatedArgs := []string{}
	updatedArgs = append(updatedArgs, args...)

	// iterate through outOfDateFlags and remove the flags from updatedArgs along with their value (if any)
	for _, item := range outOfDateFlags {
		idx := indexOf(updatedArgs, item)

		if getFlagType(item) == "BoolFlag" {
			// remove the element at index idx
			updatedArgs = remove1(updatedArgs, idx)
		} else {
			// remove the element at index idx and idx + 1
			updatedArgs = remove2(updatedArgs, idx)
		}
	}

	return updatedArgs
}

func updateArgsAdd(args []string) []string {
	for flag, value := range removedFlagsAndValues {
		if strings.HasPrefix(flag, "--") {
			flag = flag[2:]
		} else {
			flag = flag[1:]
		}

		if flagMap[flag][1] == "YesFV" {
			temp := "--" + replacedFlagsMapFlagAndValue[flag]["flag"][flag] + " " + replacedFlagsMapFlagAndValue[flag]["value"][value]
			args = append(args, temp)
		} else if flagMap[flag][1] == "YesF" {
			temp := "--" + replacedFlagsMapFlag[flag] + " " + value
			args = append(args, temp)
		}
	}

	return args
}

func handlePrometheus(args []string, updatedArgs []string) []string {
	var newUpdatedArgs []string

	mAddr := ""
	mPort := ""

	pAddr := ""
	pPort := ""

	newUpdatedArgs = append(newUpdatedArgs, updatedArgs...)

	for i, val := range args {
		if strings.Contains(val, "metrics.addr") && strings.HasPrefix(val, "-") {
			mAddr = args[i+1]
		}

		if strings.Contains(val, "metrics.port") && strings.HasPrefix(val, "-") {
			mPort = args[i+1]
		}

		if strings.Contains(val, "pprof.addr") && strings.HasPrefix(val, "-") {
			pAddr = args[i+1]
		}

		if strings.Contains(val, "pprof.port") && strings.HasPrefix(val, "-") {
			pPort = args[i+1]
		}
	}

	if mAddr != "" && mPort != "" {
		newUpdatedArgs = append(newUpdatedArgs, "--metrics.prometheus-addr")
		newUpdatedArgs = append(newUpdatedArgs, mAddr+":"+mPort)
	} else if pAddr != "" && pPort != "" {
		newUpdatedArgs = append(newUpdatedArgs, "--metrics.prometheus-addr")
		newUpdatedArgs = append(newUpdatedArgs, pAddr+":"+pPort)
	}

	return newUpdatedArgs
}

func dumpFlags(args []string) {
	err := os.WriteFile("./temp", []byte(strings.Join(args, " ")), 0600)
	if err != nil {
		fmt.Println("Error in WriteFile")
	} else {
		fmt.Println("WriteFile Done")
	}
}

// nolint: gocognit
func commentFlags(path string, updatedArgs []string) {
	const cache = "[cache]"

	const telemetry = "[telemetry]"

	// snapshot: "[cache]"
	cacheFlag := 0

	// corsdomain, vhosts, enabled, host, api, port, prefix: "[p2p]", "  [jsonrpc.http]", "  [jsonrpc.ws]", "  [jsonrpc.graphql]"
	p2pHttpWsGraphFlag := -1

	// database: "[telemetry]", "[cache]"
	databaseFlag := -1

	// password: "[telemetry]", "[accounts]"
	passwordFlag := -1

	ignoreLineFlag := false

	canonicalPath, err := common.VerifyPath(path)
	if err != nil {
		fmt.Println("path not verified: " + err.Error())
		return
	}

	input, err := os.ReadFile(canonicalPath)
	if err != nil {
		log.Fatalln(err)
	}

	lines := strings.Split(string(input), "\n")

	var newLines []string
	newLines = append(newLines, lines...)

	for i, line := range lines {
		if line == cache {
			cacheFlag += 1
		}

		if line == "[p2p]" || line == "  [jsonrpc.http]" || line == "  [jsonrpc.ws]" || line == "  [jsonrpc.graphql]" {
			p2pHttpWsGraphFlag += 1
		}

		if line == telemetry || line == cache {
			databaseFlag += 1
		}

		if line == telemetry || line == "[accounts]" {
			passwordFlag += 1
		}

		if line == "[\"eth.requiredblocks\"]" || line == "    [telemetry.influx.tags]" {
			ignoreLineFlag = true
		} else if line != "" {
			if strings.HasPrefix(strings.Fields(line)[0], "[") {
				ignoreLineFlag = false
			}
		}

		// nolint: nestif
		if !(strings.HasPrefix(line, "[") || strings.HasPrefix(line, "  [") || strings.HasPrefix(line, "    [") || line == "" || ignoreLineFlag) {
			flag := strings.Fields(line)[0]
			if flag == "snapshot" {
				flag = strconv.Itoa(cacheFlag) + "-" + flag
			} else if flag == "corsdomain" {
				flag = strconv.Itoa(p2pHttpWsGraphFlag) + "-" + flag
			} else if flag == "vhosts" {
				flag = strconv.Itoa(p2pHttpWsGraphFlag) + "-" + flag
			} else if flag == "enabled" {
				flag = strconv.Itoa(p2pHttpWsGraphFlag) + "-" + flag
			} else if flag == "host" {
				flag = strconv.Itoa(p2pHttpWsGraphFlag) + "-" + flag
			} else if flag == "api" {
				flag = strconv.Itoa(p2pHttpWsGraphFlag) + "-" + flag
			} else if flag == "port" {
				flag = strconv.Itoa(p2pHttpWsGraphFlag) + "-" + flag
			} else if flag == "prefix" {
				flag = strconv.Itoa(p2pHttpWsGraphFlag) + "-" + flag
			} else if flag == "database" {
				flag = strconv.Itoa(databaseFlag) + "-" + flag
			} else if flag == "password" {
				flag = strconv.Itoa(passwordFlag) + "-" + flag
			}

			if flag != "static-nodes" && flag != "trusted-nodes" && flag != "read" && flag != "write" && flag != "idle" && flag != "timeout" {
				flag = nameTagMap[flag]

				tempFlag := false

				for _, val := range updatedArgs {
					if strings.Contains(val, flag) && (strings.Contains(val, "-") || strings.Contains(val, "--")) {
						tempFlag = true
					}
				}

				if !tempFlag || flag == "" {
					newLines[i] = "# " + line
				}
			}
		}
	}

	output := strings.Join(newLines, "\n")

	err = os.WriteFile(canonicalPath, []byte(output), 0600)
	if err != nil {
		log.Fatalln(err)
	}
}

func checkBoolFlags(val string) bool {
	returnFlag := false

	if strings.Contains(val, "=") {
		val = strings.Split(val, "=")[0]
	}

	for _, flag := range currentBoolFlags {
		if val == "-"+flag || val == "--"+flag {
			returnFlag = true
		}
	}

	return returnFlag
}

func beautifyArgs(args []string) ([]string, []string) {
	newArgs := []string{}

	ignoreForNow := []string{}

	temp := []string{}

	for _, val := range args {
		// nolint: nestif
		if !(checkBoolFlags(val)) {
			if strings.HasPrefix(val, "-") {
				if strings.Contains(val, "=") {
					temparr := strings.Split(val, "=")
					newArgs = append(newArgs, temparr...)
				} else {
					newArgs = append(newArgs, val)
				}
			} else {
				newArgs = append(newArgs, val)
			}
		} else {
			ignoreForNow = append(ignoreForNow, val)
		}
	}

	for j, val := range newArgs {
		if val == "-unlock" || val == "--unlock" {
			temp = append(temp, "--miner.etherbase")
			temp = append(temp, newArgs[j+1])
		}
	}

	newArgs = append(newArgs, temp...)

	return newArgs, ignoreForNow
}

func main() {
	const notYet = "notYet"

	temp := os.Args[1]
	args := os.Args[2:]

	args, ignoreForNow := beautifyArgs(args)

	c := server.Command{}
	flags := c.Flags()
	allFlags := flags.GetAllFlags()
	flagsToCheck := getFlagsToCheck(args)

	if temp == notYet {
		getStaticTrustedNodes(args)
	}

	outOfDateFlags := checkFlag(allFlags, flagsToCheck)
	updatedArgs := updateArgsClean(args, outOfDateFlags)
	updatedArgs = updateArgsAdd(updatedArgs)
	updatedArgs = handlePrometheus(args, updatedArgs)

	if temp == notYet {
		updatedArgs = append(updatedArgs, ignoreForNow...)
		dumpFlags(updatedArgs)
	}

	if temp != notYet {
		updatedArgs = append(updatedArgs, ignoreForNow...)
		commentFlags(temp, updatedArgs)
	}
}
