package http

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/swarm/api"
	"github.com/ethereum/go-ethereum/swarm/chunk"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/sctx"
	"github.com/ethereum/go-ethereum/swarm/spancontext"
	"github.com/pborman/uuid"
)

// Adapt chains h (main request handler) main handler to adapters (middleware handlers)
// Please note that the order of execution for `adapters` is FIFO (adapters[0] will be executed first)
func Adapt(h http.Handler, adapters ...Adapter) http.Handler {
	for i := range adapters {
		adapter := adapters[len(adapters)-1-i]
		h = adapter(h)
	}
	return h
}

type Adapter func(http.Handler) http.Handler

func SetRequestID(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(SetRUID(r.Context(), uuid.New()[:8]))
		metrics.GetOrRegisterCounter(fmt.Sprintf("http.request.%s", r.Method), nil).Inc(1)
		log.Info("created ruid for request", "ruid", GetRUID(r.Context()), "method", r.Method, "url", r.RequestURI)

		h.ServeHTTP(w, r)
	})
}

func SetRequestHost(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(sctx.SetHost(r.Context(), r.Host))
		log.Info("setting request host", "ruid", GetRUID(r.Context()), "host", sctx.GetHost(r.Context()))

		h.ServeHTTP(w, r)
	})
}

func ParseURI(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uri, err := api.Parse(strings.TrimLeft(r.URL.Path, "/"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			respondError(w, r, fmt.Sprintf("invalid URI %q", r.URL.Path), http.StatusBadRequest)
			return
		}
		if uri.Addr != "" && strings.HasPrefix(uri.Addr, "0x") {
			uri.Addr = strings.TrimPrefix(uri.Addr, "0x")

			msg := fmt.Sprintf(`The requested hash seems to be prefixed with '0x'. You will be redirected to the correct URL within 5 seconds.<br/>
			Please click <a href='%[1]s'>here</a> if your browser does not redirect you within 5 seconds.<script>setTimeout("location.href='%[1]s';",5000);</script>`, "/"+uri.String())
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(msg))
			return
		}

		ctx := r.Context()
		r = r.WithContext(SetURI(ctx, uri))
		log.Debug("parsed request path", "ruid", GetRUID(r.Context()), "method", r.Method, "uri.Addr", uri.Addr, "uri.Path", uri.Path, "uri.Scheme", uri.Scheme)

		h.ServeHTTP(w, r)
	})
}

func InitLoggingResponseWriter(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tn := time.Now()

		writer := newLoggingResponseWriter(w)
		h.ServeHTTP(writer, r)

		ts := time.Since(tn)
		log.Info("request served", "ruid", GetRUID(r.Context()), "code", writer.statusCode, "time", ts)
		metrics.GetOrRegisterResettingTimer(fmt.Sprintf("http.request.%s.time", r.Method), nil).Update(ts)
		metrics.GetOrRegisterResettingTimer(fmt.Sprintf("http.request.%s.%d.time", r.Method, writer.statusCode), nil).Update(ts)
	})
}

// InitUploadTag creates a new tag for an upload to the local HTTP proxy
// if a tag is not named using the SwarmTagHeaderName, a fallback name will be used
// when the Content-Length header is set, an ETA on chunking will be available since the
// number of chunks to be split is known in advance (not including enclosing manifest chunks)
// the tag can later be accessed using the appropriate identifier in the request context
func InitUploadTag(h http.Handler, tags *chunk.Tags) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			tagName        string
			err            error
			estimatedTotal int64 = 0
			contentType          = r.Header.Get("Content-Type")
			headerTag            = r.Header.Get(SwarmTagHeaderName)
		)
		if headerTag != "" {
			tagName = headerTag
			log.Trace("got tag name from http header", "tagName", tagName)
		} else {
			tagName = fmt.Sprintf("unnamed_tag_%d", time.Now().Unix())
		}

		if !strings.Contains(contentType, "multipart") && r.ContentLength > 0 {
			log.Trace("calculating tag size", "contentType", contentType, "contentLength", r.ContentLength)
			uri := GetURI(r.Context())
			if uri != nil {
				log.Debug("got uri from context")
				if uri.Addr == "encrypt" {
					estimatedTotal = calculateNumberOfChunks(r.ContentLength, true)
				} else {
					estimatedTotal = calculateNumberOfChunks(r.ContentLength, false)
				}
			}
		}

		log.Trace("creating tag", "tagName", tagName, "estimatedTotal", estimatedTotal)

		t, err := tags.New(tagName, estimatedTotal)
		if err != nil {
			log.Error("error creating tag", "err", err, "tagName", tagName)
		}

		log.Trace("setting tag id to context", "uid", t.Uid)
		ctx := sctx.SetTag(r.Context(), t.Uid)

		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

func InstrumentOpenTracing(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uri := GetURI(r.Context())
		if uri == nil || r.Method == "" || (uri != nil && uri.Scheme == "") {
			h.ServeHTTP(w, r) // soft fail
			return
		}
		spanName := fmt.Sprintf("http.%s.%s", r.Method, uri.Scheme)
		ctx, sp := spancontext.StartSpan(r.Context(), spanName)

		defer sp.Finish()
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RecoverPanic(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Error("panic recovery!", "stack trace", string(debug.Stack()), "url", r.URL.String(), "headers", r.Header)
			}
		}()
		h.ServeHTTP(w, r)
	})
}
