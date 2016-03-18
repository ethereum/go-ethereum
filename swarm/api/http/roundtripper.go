package http

import (
	"fmt"
	"net/http"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

/*
http roundtripper to register for bzz url scheme
see https://github.com/ethereum/go-ethereum/issues/2040
Usage:

import (
 "github.com/ethereum/go-ethereum/common/httpclient"
 "github.com/ethereum/go-ethereum/swarm/api/http"
)
client := httpclient.New()
// for (private) swarm proxy running locally
client.RegisterScheme("bzz", &http.RoundTripper{Port: port})
client.RegisterScheme("bzzi", &http.RoundTripper{Port: port})
client.RegisterScheme("bzzr", &http.RoundTripper{Port: port})

The port you give the Roundtripper is the port the swarm proxy is listening on.
If Host is left empty, localhost is assumed.

Using a public gateway, the above few lines gives you the leanest
bzz-scheme aware read-only http client. You really only ever need this
if you need go-native swarm access to bzz addresses, e.g.,
github.com/ethereum/go-ethereum/common/natspec

*/

type RoundTripper struct {
	Host string
	Port string
}

func (self *RoundTripper) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	host := self.Host
	if len(host) == 0 {
		host = "localhost"
	}
	url := fmt.Sprintf("http://%s:%s/%s:/%s/%s", host, self.Port, req.Proto, req.URL.Host, req.URL.Path)
	glog.V(logger.Info).Infof("[BZZ] roundtripper: proxying request '%s' to '%s'", req.RequestURI, url)
	reqProxy, err := http.NewRequest(req.Method, url, req.Body)
	if err != nil {
		return nil, err
	}
	return http.DefaultClient.Do(reqProxy)
}
