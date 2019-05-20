package cors

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

var testHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("bar"))
})

func assertHeaders(t *testing.T, resHeaders http.Header, reqHeaders map[string]string) {
	for name, value := range reqHeaders {
		if resHeaders.Get(name) != value {
			t.Errorf("Invalid header `%s', wanted `%s', got `%s'", name, value, resHeaders.Get(name))
		}
	}
}

func TestNoConfig(t *testing.T) {
	s := New(Options{
	// Intentionally left blank.
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://example.com/foo", nil)

	s.Handler(testHandler).ServeHTTP(res, req)

	assertHeaders(t, res.Header(), map[string]string{
		"Access-Control-Allow-Origin":      "",
		"Access-Control-Allow-Methods":     "",
		"Access-Control-Allow-Headers":     "",
		"Access-Control-Allow-Credentials": "",
		"Access-Control-Max-Age":           "",
		"Access-Control-Expose-Headers":    "",
	})
}

func TestWildcardOrigin(t *testing.T) {
	s := New(Options{
		AllowedOrigins: []string{"*"},
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://example.com/foo", nil)
	req.Header.Add("Origin", "http://foobar.com")

	s.Handler(testHandler).ServeHTTP(res, req)

	assertHeaders(t, res.Header(), map[string]string{
		"Access-Control-Allow-Origin":      "http://foobar.com",
		"Access-Control-Allow-Methods":     "",
		"Access-Control-Allow-Headers":     "",
		"Access-Control-Allow-Credentials": "",
		"Access-Control-Max-Age":           "",
		"Access-Control-Expose-Headers":    "",
	})
}

func TestAllowedOrigin(t *testing.T) {
	s := New(Options{
		AllowedOrigins: []string{"http://foobar.com"},
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://example.com/foo", nil)
	req.Header.Add("Origin", "http://foobar.com")

	s.Handler(testHandler).ServeHTTP(res, req)

	assertHeaders(t, res.Header(), map[string]string{
		"Access-Control-Allow-Origin":      "http://foobar.com",
		"Access-Control-Allow-Methods":     "",
		"Access-Control-Allow-Headers":     "",
		"Access-Control-Allow-Credentials": "",
		"Access-Control-Max-Age":           "",
		"Access-Control-Expose-Headers":    "",
	})
}

func TestDisallowedOrigin(t *testing.T) {
	s := New(Options{
		AllowedOrigins: []string{"http://foobar.com"},
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://example.com/foo", nil)
	req.Header.Add("Origin", "http://barbaz.com")

	s.Handler(testHandler).ServeHTTP(res, req)

	assertHeaders(t, res.Header(), map[string]string{
		"Access-Control-Allow-Origin":      "",
		"Access-Control-Allow-Methods":     "",
		"Access-Control-Allow-Headers":     "",
		"Access-Control-Allow-Credentials": "",
		"Access-Control-Max-Age":           "",
		"Access-Control-Expose-Headers":    "",
	})
}

func TestAllowedMethod(t *testing.T) {
	s := New(Options{
		AllowedOrigins: []string{"http://foobar.com"},
		AllowedMethods: []string{"PUT", "DELETE"},
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "http://example.com/foo", nil)
	req.Header.Add("Origin", "http://foobar.com")
	req.Header.Add("Access-Control-Request-Method", "PUT")

	s.Handler(testHandler).ServeHTTP(res, req)

	assertHeaders(t, res.Header(), map[string]string{
		"Access-Control-Allow-Origin":      "http://foobar.com",
		"Access-Control-Allow-Methods":     "PUT",
		"Access-Control-Allow-Headers":     "",
		"Access-Control-Allow-Credentials": "",
		"Access-Control-Max-Age":           "",
		"Access-Control-Expose-Headers":    "",
	})
}

func TestDisallowedMethod(t *testing.T) {
	s := New(Options{
		AllowedOrigins: []string{"http://foobar.com"},
		AllowedMethods: []string{"PUT", "DELETE"},
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "http://example.com/foo", nil)
	req.Header.Add("Origin", "http://foobar.com")
	req.Header.Add("Access-Control-Request-Method", "PATCH")

	s.Handler(testHandler).ServeHTTP(res, req)

	assertHeaders(t, res.Header(), map[string]string{
		"Access-Control-Allow-Origin":      "",
		"Access-Control-Allow-Methods":     "",
		"Access-Control-Allow-Headers":     "",
		"Access-Control-Allow-Credentials": "",
		"Access-Control-Max-Age":           "",
		"Access-Control-Expose-Headers":    "",
	})
}

func TestAllowedHeader(t *testing.T) {
	s := New(Options{
		AllowedOrigins: []string{"http://foobar.com"},
		AllowedHeaders: []string{"X-Header-1", "x-header-2"},
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "http://example.com/foo", nil)
	req.Header.Add("Origin", "http://foobar.com")
	req.Header.Add("Access-Control-Request-Method", "GET")
	req.Header.Add("Access-Control-Request-Headers", "X-Header-2, X-HEADER-1")

	s.Handler(testHandler).ServeHTTP(res, req)

	assertHeaders(t, res.Header(), map[string]string{
		"Access-Control-Allow-Origin":      "http://foobar.com",
		"Access-Control-Allow-Methods":     "GET",
		"Access-Control-Allow-Headers":     "X-Header-2, X-Header-1",
		"Access-Control-Allow-Credentials": "",
		"Access-Control-Max-Age":           "",
		"Access-Control-Expose-Headers":    "",
	})
}

func TestAllowedWildcardHeader(t *testing.T) {
	s := New(Options{
		AllowedOrigins: []string{"http://foobar.com"},
		AllowedHeaders: []string{"*"},
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "http://example.com/foo", nil)
	req.Header.Add("Origin", "http://foobar.com")
	req.Header.Add("Access-Control-Request-Method", "GET")
	req.Header.Add("Access-Control-Request-Headers", "X-Header-2, X-HEADER-1")

	s.Handler(testHandler).ServeHTTP(res, req)

	assertHeaders(t, res.Header(), map[string]string{
		"Access-Control-Allow-Origin":      "http://foobar.com",
		"Access-Control-Allow-Methods":     "GET",
		"Access-Control-Allow-Headers":     "X-Header-2, X-Header-1",
		"Access-Control-Allow-Credentials": "",
		"Access-Control-Max-Age":           "",
		"Access-Control-Expose-Headers":    "",
	})
}

func TestDisallowedHeader(t *testing.T) {
	s := New(Options{
		AllowedOrigins: []string{"http://foobar.com"},
		AllowedHeaders: []string{"X-Header-1", "x-header-2"},
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "http://example.com/foo", nil)
	req.Header.Add("Origin", "http://foobar.com")
	req.Header.Add("Access-Control-Request-Method", "GET")
	req.Header.Add("Access-Control-Request-Headers", "X-Header-3, X-Header-1")

	s.Handler(testHandler).ServeHTTP(res, req)

	assertHeaders(t, res.Header(), map[string]string{
		"Access-Control-Allow-Origin":      "",
		"Access-Control-Allow-Methods":     "",
		"Access-Control-Allow-Headers":     "",
		"Access-Control-Allow-Credentials": "",
		"Access-Control-Max-Age":           "",
		"Access-Control-Expose-Headers":    "",
	})
}

func TestOriginHeader(t *testing.T) {
	s := New(Options{
		AllowedOrigins: []string{"http://foobar.com"},
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "http://example.com/foo", nil)
	req.Header.Add("Origin", "http://foobar.com")
	req.Header.Add("Access-Control-Request-Method", "GET")
	req.Header.Add("Access-Control-Request-Headers", "origin")

	s.Handler(testHandler).ServeHTTP(res, req)

	assertHeaders(t, res.Header(), map[string]string{
		"Access-Control-Allow-Origin":      "http://foobar.com",
		"Access-Control-Allow-Methods":     "GET",
		"Access-Control-Allow-Headers":     "Origin",
		"Access-Control-Allow-Credentials": "",
		"Access-Control-Max-Age":           "",
		"Access-Control-Expose-Headers":    "",
	})
}

func TestExposedHeader(t *testing.T) {
	s := New(Options{
		AllowedOrigins: []string{"http://foobar.com"},
		ExposedHeaders: []string{"X-Header-1", "x-header-2"},
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://example.com/foo", nil)
	req.Header.Add("Origin", "http://foobar.com")

	s.Handler(testHandler).ServeHTTP(res, req)

	assertHeaders(t, res.Header(), map[string]string{
		"Access-Control-Allow-Origin":      "http://foobar.com",
		"Access-Control-Allow-Methods":     "",
		"Access-Control-Allow-Headers":     "",
		"Access-Control-Allow-Credentials": "",
		"Access-Control-Max-Age":           "",
		"Access-Control-Expose-Headers":    "X-Header-1, X-Header-2",
	})
}

func TestAllowedCredentials(t *testing.T) {
	s := New(Options{
		AllowedOrigins:   []string{"http://foobar.com"},
		AllowCredentials: true,
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "http://example.com/foo", nil)
	req.Header.Add("Origin", "http://foobar.com")
	req.Header.Add("Access-Control-Request-Method", "GET")

	s.Handler(testHandler).ServeHTTP(res, req)

	assertHeaders(t, res.Header(), map[string]string{
		"Access-Control-Allow-Origin":      "http://foobar.com",
		"Access-Control-Allow-Methods":     "GET",
		"Access-Control-Allow-Headers":     "",
		"Access-Control-Allow-Credentials": "true",
		"Access-Control-Max-Age":           "",
		"Access-Control-Expose-Headers":    "",
	})
}
