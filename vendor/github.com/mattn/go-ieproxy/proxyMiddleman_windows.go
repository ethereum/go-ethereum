package ieproxy

import (
	"net/http"
	"net/url"

	"golang.org/x/net/http/httpproxy"
)

func proxyMiddleman() func(req *http.Request) (i *url.URL, e error) {
	// Get the proxy configuration
	conf := GetConf()
	envcfg := httpproxy.FromEnvironment()

	if envcfg.HTTPProxy != "" || envcfg.HTTPSProxy != "" {
		// If the user manually specifies environment variables, prefer those over the Windows config.
		return http.ProxyFromEnvironment
	} else if conf.Automatic.Active {
		// If automatic proxy obtaining is specified
		return func(req *http.Request) (i *url.URL, e error) {
			host := conf.Automatic.FindProxyForURL(req.URL.String())
			if host == "" {
				return nil, nil
			}
			return &url.URL{Host: host}, nil
		}
	} else if conf.Static.Active {
		// If static proxy obtaining is specified
		prox := httpproxy.Config{
			HTTPSProxy: mapFallback("https", "", conf.Static.Protocols),
			HTTPProxy:  mapFallback("http", "", conf.Static.Protocols),
			NoProxy:    conf.Static.NoProxy,
		}

		return func(req *http.Request) (i *url.URL, e error) {
			return prox.ProxyFunc()(req.URL)
		}
	} else {
		// Final fallthrough case; use the environment variables.
		return http.ProxyFromEnvironment
	}
}

// Return oKey or fbKey if oKey doesn't exist in the map.
func mapFallback(oKey, fbKey string, m map[string]string) string {
	if v, ok := m[oKey]; ok {
		return v
	} else {
		return m[fbKey]
	}
}
