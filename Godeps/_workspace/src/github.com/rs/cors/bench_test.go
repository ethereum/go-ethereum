package cors

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkWithout(b *testing.B) {
	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://example.com/foo", nil)

	for i := 0; i < b.N; i++ {
		testHandler.ServeHTTP(res, req)
	}
}

func BenchmarkDefault(b *testing.B) {
	res := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://example.com/foo", nil)
	handler := Default()

	for i := 0; i < b.N; i++ {
		handler.Handler(testHandler).ServeHTTP(res, req)
	}
}

func BenchmarkPreflight(b *testing.B) {
	res := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "http://example.com/foo", nil)
	req.Header.Add("Access-Control-Request-Method", "GET")
	handler := Default()

	for i := 0; i < b.N; i++ {
		handler.Handler(testHandler).ServeHTTP(res, req)
	}
}
