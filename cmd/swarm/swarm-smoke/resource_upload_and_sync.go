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

	"github.com/pborman/uuid"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm/multihash"
	"github.com/ethereum/go-ethereum/swarm/storage/mru"

	cli "gopkg.in/urfave/cli.v1"
)

const (
	resourceRandomDataLength = 8
)

// TODO: retrieve with manifest + extract repeating code
func cliResourceUploadAndSync(c *cli.Context) error {
	defer func(now time.Time) { log.Info("total time", "time", time.Since(now), "size (kb)", filesize) }(time.Now())

	generateEndpoints(scheme, cluster, from, to)

	log.Info("generating and uploading MRUs to " + endpoints[0] + " and syncing")

	// create a random private key to sign updates with and derive the address
	pkFile, err := ioutil.TempFile("", "swarm-resource-smoke-test")
	if err != nil {
		return err
	}
	defer pkFile.Close()
	defer os.Remove(pkFile.Name())

	privkeyHex := "0000000000000000000000000000000000000000000000000000000000001976"
	privKey, _ := crypto.HexToECDSA(privkeyHex)
	user := crypto.PubkeyToAddress(privKey.PublicKey)
	userHex := hexutil.Encode(user.Bytes())

	// save the private key to a file
	_, err = io.WriteString(pkFile, privkeyHex)
	if err != nil {
		return err
	}

	// create a random topic and subtopic
	topicBytes, err := generateRandomData(mru.TopicLength)
	topicHex := hexutil.Encode(topicBytes)
	subTopicBytes, err := generateRandomData(8)
	subTopicHex := hexutil.Encode(subTopicBytes)
	if err != nil {
		return err
	}
	mergedSubTopic, err := mru.NewTopic(subTopicHex, topicBytes)
	if err != nil {
		return err
	}
	mergedSubTopicHex := hexutil.Encode(mergedSubTopic[:])
	subTopicOnlyBytes, err := mru.NewTopic(subTopicHex, nil)
	if err != nil {
		return err
	}
	subTopicOnlyHex := hexutil.Encode(subTopicOnlyBytes[:])

	// create resource manifest, topic only
	var out bytes.Buffer
	cmd := exec.Command("swarm", "--bzzapi", endpoints[0], "resource", "create", "--topic", topicHex, "--user", userHex)
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		log.Debug("cmd", "cmd", cmd)
		return err
	}
	result := strings.Split(out.String(), "\n")
	if len(result) < 2 {
		return fmt.Errorf("unknown resource create result format (topic): %s", result)
	} else if len(result[1]) != 64 {
		return fmt.Errorf("unknown resource create manifest hash format (topic): %s", result[1])
	}
	manifestWithTopic := result[1]
	if err != nil {
		return err
	}
	log.Debug("create topic resource", "manifest", manifestWithTopic)
	out.Reset()

	// create resource manifest, subtopic only
	cmd = exec.Command("swarm", "--bzzapi", endpoints[0], "resource", "create", "--name", subTopicHex, "--user", userHex)
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		log.Error(err.Error())
		return err
	}
	result = strings.Split(out.String(), "\n")
	if len(result) < 2 {
		return fmt.Errorf("unknown resource create result format (subtopic): %s", result)
	} else if len(result[1]) != 64 {
		return fmt.Errorf("unknown resource create manifest hash format (subtopic): %s", result[1])
	}
	manifestWithSubTopic := result[1]
	if err != nil {
		return err
	}
	log.Debug("create subtopic resource", "manifest", manifestWithSubTopic)
	out.Reset()

	// create resource manifest, merged topic
	cmd = exec.Command("swarm", "--bzzapi", endpoints[0], "resource", "create", "--topic", topicHex, "--name", subTopicHex, "--user", userHex)
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		log.Error(err.Error())
		return err
	}
	result = strings.Split(out.String(), "\n")
	if len(result) < 2 {
		return fmt.Errorf("unknown resource create result format (mergedtopic): %s", result)
	} else if len(result[1]) != 64 {
		return fmt.Errorf("unknown resource create manifest hash format (mergedtopic): %s", result[1])
	}
	manifestWithMergedTopic := result[1]
	if err != nil {
		return err
	}
	log.Debug("create mergedtopic resource", "manifest", manifestWithMergedTopic)
	out.Reset()

	// create test data
	data, err := generateRandomData(resourceRandomDataLength)
	if err != nil {
		return err
	}
	h := md5.New()
	h.Write(data)
	dataHash := h.Sum(nil)
	dataHex := hexutil.Encode(data)

	// update with topic
	cmd = exec.Command("swarm", "--bzzaccount", pkFile.Name(), "--bzzapi", endpoints[0], "resource", "update", "--topic", topicHex, dataHex)
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		return err
	}
	log.Debug("resource update topic", "out", out)
	out.Reset()

	// update with topic
	cmd = exec.Command("swarm", "--bzzaccount", pkFile.Name(), "--bzzapi", endpoints[0], "resource", "update", "--name", subTopicHex, dataHex)
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		return err
	}
	log.Debug("resource update subtopic", "out", out)
	out.Reset()

	// update with merged topic
	cmd = exec.Command("swarm", "--bzzaccount", pkFile.Name(), "--bzzapi", endpoints[0], "resource", "update", "--topic", topicHex, "--name", subTopicHex, dataHex)
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		return err
	}
	log.Debug("resource update mergedtopic", "out", out)
	out.Reset()

	time.Sleep(3 * time.Second)

	// retrieve the data
	wg := sync.WaitGroup{}
	for _, endpoint := range endpoints {
		endpoint := endpoint
		wg.Add(3)

		// raw retrieve, topic only
		ruid := uuid.New()[:8]
		go func(endpoint string, ruid string) {
			for {
				err := fetchResource(topicHex, userHex, endpoint, dataHash, ruid)
				if err != nil {
					continue
				}

				wg.Done()
				return
			}
		}(endpoint, ruid)

		// raw retrieve, subtopic only
		ruid = uuid.New()[:8]
		go func(endpoint string, ruid string) {
			for {
				err := fetchResource(subTopicOnlyHex, userHex, endpoint, dataHash, ruid)
				if err != nil {
					continue
				}

				wg.Done()
				return
			}
		}(endpoint, ruid)

		// raw retrieve, merged topic
		ruid = uuid.New()[:8]
		go func(endpoint string, ruid string) {
			for {
				err := fetchResource(mergedSubTopicHex, userHex, endpoint, dataHash, ruid)
				if err != nil {
					continue
				}

				wg.Done()
				return
			}
		}(endpoint, ruid)

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
	cmd = exec.Command("swarm", "--bzzaccount", pkFile.Name(), "--bzzapi", endpoints[0], "resource", "update", "--topic", topicHex, multihashHex)
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		return err
	}
	log.Debug("resource update topic", "out", out)
	out.Reset()

	// update file with subtopic
	cmd = exec.Command("swarm", "--bzzaccount", pkFile.Name(), "--bzzapi", endpoints[0], "resource", "update", "--name", subTopicHex, multihashHex)
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		return err
	}
	log.Debug("resource update subtopic", "out", out)
	out.Reset()

	// update file with merged topic
	cmd = exec.Command("swarm", "--bzzaccount", pkFile.Name(), "--bzzapi", endpoints[0], "resource", "update", "--topic", topicHex, "--name", subTopicHex, multihashHex)
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		return err
	}
	log.Debug("resource update mergedtopic", "out", out)
	out.Reset()

	time.Sleep(3 * time.Second)

	for _, endpoint := range endpoints {
		endpoint := endpoint //????
		wg.Add(3)

		// manifest retrieve, topic only
		ruid := uuid.New()[:8]
		go func(endpoint string, ruid string) {
			for {
				err := fetch(manifestWithTopic, endpoint, fileHash, ruid)
				if err != nil {
					continue
				}

				wg.Done()
				return
			}
		}(endpoint, ruid)

		// manifest retrieve, subtopic only
		ruid = uuid.New()[:8]
		go func(endpoint string, ruid string) {
			for {
				err := fetch(manifestWithSubTopic, endpoint, fileHash, ruid)
				if err != nil {
					continue
				}

				wg.Done()
				return
			}
		}(endpoint, ruid)

		// manifest retrieve, merged topic
		ruid = uuid.New()[:8]
		go func(endpoint string, ruid string) {
			for {
				err := fetch(manifestWithMergedTopic, endpoint, fileHash, ruid)
				if err != nil {
					continue
				}

				wg.Done()
				return
			}
		}(endpoint, ruid)
	}
	wg.Wait()
	log.Info("all endpoints synced random file successfully")

	return nil
}

func fetchResource(topic string, user string, endpoint string, original []byte, ruid string) error {
	log.Trace("sleeping", "ruid", ruid)
	time.Sleep(3 * time.Second)

	log.Trace("http get request (resource)", "ruid", ruid, "api", endpoint, "topic", topic, "user", user)
	res, err := http.Get(endpoint + "/bzz-resource:/?topic=" + topic + "&user=" + user)
	if err != nil {
		log.Warn(err.Error(), "ruid", ruid)
		return err
	}
	log.Trace("http get response (resource)", "ruid", ruid, "api", endpoint, "topic", topic, "user", user, "code", res.StatusCode, "len", res.ContentLength)

	if res.StatusCode != 200 {
		err := fmt.Errorf("expected status code %d, got %v", 200, res.StatusCode)
		log.Warn(err.Error(), "ruid", ruid)
		return err
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
