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

type IHeimdallClient interface {
	Fetch(paths ...string) (*ResponseWithHeight, error)
	FetchWithRetry(paths ...string) (*ResponseWithHeight, error)
}

type HeimdallClient struct {
	u      *url.URL
	client http.Client
}

func NewHeimdallClient(urlString string) (*HeimdallClient, error) {
	u, err := url.Parse(urlString)
	if err != nil {
		return nil, err
	}

	h := &HeimdallClient{
		u: u,
		client: http.Client{
			Timeout: time.Duration(5 * time.Second),
		},
	}
	return h, nil
}

func (h *HeimdallClient) Fetch(paths ...string) (*ResponseWithHeight, error) {
	for _, e := range paths {
		if e != "" {
			h.u.Path = path.Join(h.u.Path, e)
		}
	}
	return h.internalFetch()
}

// FetchWithRetry returns data from heimdall with retry
func (h *HeimdallClient) FetchWithRetry(paths ...string) (*ResponseWithHeight, error) {
	for _, e := range paths {
		if e != "" {
			h.u.Path = path.Join(h.u.Path, e)
		}
	}

	for {
		res, err := h.internalFetch()
		if err == nil && res != nil {
			return res, nil
		}
		log.Info("Retrying again in 5 seconds for next Heimdall span", "path", h.u.Path)
		time.Sleep(5 * time.Second)
	}
}

// internal fetch method
func (h *HeimdallClient) internalFetch() (*ResponseWithHeight, error) {
	res, err := h.client.Get(h.u.String())
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
