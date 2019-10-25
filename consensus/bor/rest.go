package bor

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
)

// ResponseWithHeight defines a response object type that wraps an original
// response with a height.
type ResponseWithHeight struct {
	Height string          `json:"height"`
	Result json.RawMessage `json:"result"`
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
	fmt.Println("body", string(body))
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	fmt.Println("response", response.Result)
	return &response, nil
}
