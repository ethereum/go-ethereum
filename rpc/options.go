package rpc

import (
	"net/http"
)

type unimplementedOption struct{}

func (s *unimplementedOption) HTTPRoundTripper() http.RoundTripper { return nil }

type httpRoundTripper struct {
	*unimplementedOption
	Certificate http.RoundTripper
}

func (s *httpRoundTripper) HTTPRoundTripper() http.RoundTripper {
	return s.Certificate
}

func HTTPRoundTripper(roundTripper http.RoundTripper) Options {
	return &httpRoundTripper{
		Certificate: roundTripper,
	}
}
