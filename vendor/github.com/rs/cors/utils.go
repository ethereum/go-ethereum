package cors

import (
	"net/http"
	"strings"
)

type converter func(string) string

// convert converts a list of string using the passed converter function
func convert(s []string, c converter) []string {
	out := []string{}
	for _, i := range s {
		out = append(out, c(i))
	}
	return out
}

func parseHeaderList(headerList string) (headers []string) {
	for _, header := range strings.Split(headerList, ",") {
		header = http.CanonicalHeaderKey(strings.TrimSpace(header))
		if header != "" {
			headers = append(headers, header)
		}
	}
	return headers
}
