package bor

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/maticnetwork/bor/log"
)

// ResponseWithHeight defines a response object type that wraps an original
// response with a height.
type ResponseWithHeight struct {
	Height string          `json:"height"`
	Result json.RawMessage `json:"result"`
}

// internal fetch method
func internalFetch(client http.Client, u *url.URL) (*ResponseWithHeight, error) {
	res, err := client.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	// check status code
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("Error while fetching data from Heimdall")
	}

	// get response
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	// unmarshall data from buffer
	var response ResponseWithHeight
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// FetchFromHeimdallWithRetry returns data from heimdall with retry
func FetchFromHeimdallWithRetry(client http.Client, urlString string, paths ...string) (*ResponseWithHeight, error) {
	u, err := url.Parse(urlString)
	if err != nil {
		return nil, err
	}

	for _, e := range paths {
		if e != "" {
			u.Path = path.Join(u.Path, e)
		}
	}

	for {
		res, err := internalFetch(client, u)
		if err == nil && res != nil {
			return res, nil
		}
		log.Info("Retrying again in 5 seconds", u.String())
		time.Sleep(5 * time.Second)
	}
}

// FetchFromHeimdall returns data from heimdall
func FetchFromHeimdall(client http.Client, urlString string, paths ...string) (*ResponseWithHeight, error) {
	u, err := url.Parse(urlString)
	if err != nil {
		return nil, err
	}

	for _, e := range paths {
		if e != "" {
			u.Path = path.Join(u.Path, e)
		}
	}

	return internalFetch(client, u)
}
