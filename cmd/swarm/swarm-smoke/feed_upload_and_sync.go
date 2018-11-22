package main

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm/multihash"
	"github.com/ethereum/go-ethereum/swarm/storage/feed"
	colorable "github.com/mattn/go-colorable"
	"github.com/pborman/uuid"
	cli "gopkg.in/urfave/cli.v1"
)

const (
	feedRandomDataLength = 8
)

// TODO: retrieve with manifest + extract repeating code
func cliFeedUploadAndSync(c *cli.Context) error {

	log.Root().SetHandler(log.CallerFileHandler(log.LvlFilterHandler(log.Lvl(verbosity), log.StreamHandler(colorable.NewColorableStderr(), log.TerminalFormat(true)))))

	defer func(now time.Time) { log.Info("total time", "time", time.Since(now), "size (kb)", filesize) }(time.Now())

	generateEndpoints(scheme, cluster, from, to)

	log.Info("generating and uploading MRUs to " + endpoints[0] + " and syncing")

	// create a random private key to sign updates with and derive the address
	pkFile, err := ioutil.TempFile("", "swarm-feed-smoke-test")
	if err != nil {
		return err
	}
	defer pkFile.Close()
	defer os.Remove(pkFile.Name())

	privkeyHex := "0000000000000000000000000000000000000000000000000000000000001976"
	privKey, err := crypto.HexToECDSA(privkeyHex)
	if err != nil {
		return err
	}
	user := crypto.PubkeyToAddress(privKey.PublicKey)
	userHex := hexutil.Encode(user.Bytes())

	// save the private key to a file
	_, err = io.WriteString(pkFile, privkeyHex)
	if err != nil {
		return err
	}

	// keep hex strings for topic and subtopic
	var topicHex string
	var subTopicHex string

	// and create combination hex topics for bzz-feed retrieval
	// xor'ed with topic (zero-value topic if no topic)
	var subTopicOnlyHex string
	var mergedSubTopicHex string

	// generate random topic and subtopic and put a hex on them
	topicBytes, err := generateRandomData(feed.TopicLength)
	topicHex = hexutil.Encode(topicBytes)
	subTopicBytes, err := generateRandomData(8)
	subTopicHex = hexutil.Encode(subTopicBytes)
	if err != nil {
		return err
	}
	mergedSubTopic, err := feed.NewTopic(subTopicHex, topicBytes)
	if err != nil {
		return err
	}
	mergedSubTopicHex = hexutil.Encode(mergedSubTopic[:])
	subTopicOnlyBytes, err := feed.NewTopic(subTopicHex, nil)
	if err != nil {
		return err
	}
	subTopicOnlyHex = hexutil.Encode(subTopicOnlyBytes[:])

	// create feed manifest, topic only
	var out bytes.Buffer
	cmd := exec.Command("swarm", "--bzzapi", endpoints[0], "feed", "create", "--topic", topicHex, "--user", userHex)
	cmd.Stdout = &out
	log.Debug("create feed manifest topic cmd", "cmd", cmd)
	err = cmd.Run()
	if err != nil {
		return err
	}
	manifestWithTopic := strings.TrimRight(out.String(), string([]byte{0x0a}))
	if len(manifestWithTopic) != 64 {
		return fmt.Errorf("unknown feed create manifest hash format (topic): (%d) %s", len(out.String()), manifestWithTopic)
	}
	log.Debug("create topic feed", "manifest", manifestWithTopic)
	out.Reset()

	// create feed manifest, subtopic only
	cmd = exec.Command("swarm", "--bzzapi", endpoints[0], "feed", "create", "--name", subTopicHex, "--user", userHex)
	cmd.Stdout = &out
	log.Debug("create feed manifest subtopic cmd", "cmd", cmd)
	err = cmd.Run()
	if err != nil {
		return err
	}
	manifestWithSubTopic := strings.TrimRight(out.String(), string([]byte{0x0a}))
	if len(manifestWithSubTopic) != 64 {
		return fmt.Errorf("unknown feed create manifest hash format (subtopic): (%d) %s", len(out.String()), manifestWithSubTopic)
	}
	log.Debug("create subtopic feed", "manifest", manifestWithTopic)
	out.Reset()

	// create feed manifest, merged topic
	cmd = exec.Command("swarm", "--bzzapi", endpoints[0], "feed", "create", "--topic", topicHex, "--name", subTopicHex, "--user", userHex)
	cmd.Stdout = &out
	log.Debug("create feed manifest mergetopic cmd", "cmd", cmd)
	err = cmd.Run()
	if err != nil {
		log.Error(err.Error())
		return err
	}
	manifestWithMergedTopic := strings.TrimRight(out.String(), string([]byte{0x0a}))
	if len(manifestWithMergedTopic) != 64 {
		return fmt.Errorf("unknown feed create manifest hash format (mergedtopic): (%d) %s", len(out.String()), manifestWithMergedTopic)
	}
	log.Debug("create mergedtopic feed", "manifest", manifestWithMergedTopic)
	out.Reset()

	// create test data
	data, err := generateRandomData(feedRandomDataLength)
	if err != nil {
		return err
	}
	h := md5.New()
	h.Write(data)
	dataHash := h.Sum(nil)
	dataHex := hexutil.Encode(data)

	// update with topic
	cmd = exec.Command("swarm", "--bzzaccount", pkFile.Name(), "--bzzapi", endpoints[0], "feed", "update", "--topic", topicHex, dataHex)
	cmd.Stdout = &out
	log.Debug("update feed manifest topic cmd", "cmd", cmd)
	err = cmd.Run()
	if err != nil {
		return err
	}
	log.Debug("feed update topic", "out", out)
	out.Reset()

	// update with subtopic
	cmd = exec.Command("swarm", "--bzzaccount", pkFile.Name(), "--bzzapi", endpoints[0], "feed", "update", "--name", subTopicHex, dataHex)
	cmd.Stdout = &out
	log.Debug("update feed manifest subtopic cmd", "cmd", cmd)
	err = cmd.Run()
	if err != nil {
		return err
	}
	log.Debug("feed update subtopic", "out", out)
	out.Reset()

	// update with merged topic
	cmd = exec.Command("swarm", "--bzzaccount", pkFile.Name(), "--bzzapi", endpoints[0], "feed", "update", "--topic", topicHex, "--name", subTopicHex, dataHex)
	cmd.Stdout = &out
	log.Debug("update feed manifest merged topic cmd", "cmd", cmd)
	err = cmd.Run()
	if err != nil {
		return err
	}
	log.Debug("feed update mergedtopic", "out", out)
	out.Reset()

	time.Sleep(3 * time.Second)

	// retrieve the data
	wg := sync.WaitGroup{}
	for _, endpoint := range endpoints {
		// raw retrieve, topic only
		for _, hex := range []string{topicHex, subTopicOnlyHex, mergedSubTopicHex} {
			wg.Add(1)
			ruid := uuid.New()[:8]
			go func(hex string, endpoint string, ruid string) {
				for {
					err := fetchFeed(hex, userHex, endpoint, dataHash, ruid)
					if err != nil {
						continue
					}

					wg.Done()
					return
				}
			}(hex, endpoint, ruid)

		}
	}
	wg.Wait()
	log.Info("all endpoints synced random data successfully")

	// upload test file
	log.Info("uploading to " + endpoints[0] + " and syncing")

	f, cleanup := generateRandomFile(filesize * 1000)
	defer cleanup()

	hash, err := upload(f, endpoints[0])
	if err != nil {
		return err
	}
	hashBytes, err := hexutil.Decode("0x" + hash)
	if err != nil {
		return err
	}
	multihashHex := hexutil.Encode(multihash.ToMultihash(hashBytes))

	fileHash, err := digest(f)
	if err != nil {
		return err
	}

	log.Info("uploaded successfully", "hash", hash, "digest", fmt.Sprintf("%x", fileHash))

	// update file with topic
	cmd = exec.Command("swarm", "--bzzaccount", pkFile.Name(), "--bzzapi", endpoints[0], "feed", "update", "--topic", topicHex, multihashHex)
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		return err
	}
	log.Debug("feed update topic", "out", out)
	out.Reset()

	// update file with subtopic
	cmd = exec.Command("swarm", "--bzzaccount", pkFile.Name(), "--bzzapi", endpoints[0], "feed", "update", "--name", subTopicHex, multihashHex)
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		return err
	}
	log.Debug("feed update subtopic", "out", out)
	out.Reset()

	// update file with merged topic
	cmd = exec.Command("swarm", "--bzzaccount", pkFile.Name(), "--bzzapi", endpoints[0], "feed", "update", "--topic", topicHex, "--name", subTopicHex, multihashHex)
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		return err
	}
	log.Debug("feed update mergedtopic", "out", out)
	out.Reset()

	time.Sleep(3 * time.Second)

	for _, endpoint := range endpoints {

		// manifest retrieve, topic only
		for _, url := range []string{manifestWithTopic, manifestWithSubTopic, manifestWithMergedTopic} {
			wg.Add(1)
			ruid := uuid.New()[:8]
			go func(url string, endpoint string, ruid string) {
				for {
					err := fetch(url, endpoint, fileHash, ruid)
					if err != nil {
						continue
					}

					wg.Done()
					return
				}
			}(url, endpoint, ruid)
		}

	}
	wg.Wait()
	log.Info("all endpoints synced random file successfully")

	return nil
}

func fetchFeed(topic string, user string, endpoint string, original []byte, ruid string) error {
	log.Trace("sleeping", "ruid", ruid)
	time.Sleep(3 * time.Second)

	log.Trace("http get request (feed)", "ruid", ruid, "api", endpoint, "topic", topic, "user", user)
	res, err := http.Get(endpoint + "/bzz-feed:/?topic=" + topic + "&user=" + user)
	if err != nil {
		return err
	}
	log.Trace("http get response (feed)", "ruid", ruid, "api", endpoint, "topic", topic, "user", user, "code", res.StatusCode, "len", res.ContentLength)

	if res.StatusCode != 200 {
		return fmt.Errorf("expected status code %d, got %v (ruid %v)", 200, res.StatusCode, ruid)
	}

	defer res.Body.Close()

	rdigest, err := digest(res.Body)
	if err != nil {
		log.Warn(err.Error(), "ruid", ruid)
		return err
	}

	if !bytes.Equal(rdigest, original) {
		err := fmt.Errorf("downloaded imported file md5=%x is not the same as the generated one=%x", rdigest, original)
		log.Warn(err.Error(), "ruid", ruid)
		return err
	}

	log.Trace("downloaded file matches random file", "ruid", ruid, "len", res.ContentLength)

	return nil
}
