/*
Package cors is net/http handler to handle CORS related requests
as defined by http://www.w3.org/TR/cors/

You can configure it by passing an option struct to cors.New:

    c := cors.New(cors.Options{
        AllowedOrigins: []string{"foo.com"},
        AllowedMethods: []string{"GET", "POST", "DELETE"},
        AllowCredentials: true,
    })

Then insert the handler in the chain:

    handler = c.Handler(handler)

See Options documentation for more options.

The resulting handler is a standard net/http handler.
*/
package cors

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// Options is a configuration container to setup the CORS middleware.
type Options struct {
	// AllowedOrigins is a list of origins a cross-domain request can be executed from.
	// If the special "*" value is present in the list, all origins will be allowed.
	// Default value is ["*"]
	AllowedOrigins []string
	// AllowedMethods is a list of methods the client is allowed to use with
	// cross-domain requests. Default value is simple methods (GET and POST)
	AllowedMethods []string
	// AllowedHeaders is list of non simple headers the client is allowed to use with
	// cross-domain requests.
	// If the special "*" value is present in the list, all headers will be allowed.
	// Default value is [] but "Origin" is always appended to the list.
	AllowedHeaders []string
	// ExposedHeaders indicates which headers are safe to expose to the API of a CORS
	// API specification
	ExposedHeaders []string
	// AllowCredentials indicates whether the request can include user credentials like
	// cookies, HTTP authentication or client side SSL certificates.
	AllowCredentials bool
	// MaxAge indicates how long (in seconds) the results of a preflight request
	// can be cached
	MaxAge int
	// Debugging flag adds additional output to debug server side CORS issues
	Debug bool
	// log object to use when debugging
	log *log.Logger
}

type Cors struct {
	// The CORS Options
	options Options
}

// New creates a new Cors handler with the provided options.
func New(options Options) *Cors {
	// Normalize options
	// Note: for origins and methods matching, the spec requires a case-sensitive matching.
	// As it may error prone, we chose to ignore the spec here.
	normOptions := Options{
		AllowedOrigins: convert(options.AllowedOrigins, strings.ToLower),
		AllowedMethods: convert(options.AllowedMethods, strings.ToUpper),
		// Origin is always appended as some browsers will always request
		// for this header at preflight
		AllowedHeaders:   convert(append(options.AllowedHeaders, "Origin"), http.CanonicalHeaderKey),
		ExposedHeaders:   convert(options.ExposedHeaders, http.CanonicalHeaderKey),
		AllowCredentials: options.AllowCredentials,
		MaxAge:           options.MaxAge,
		Debug:            options.Debug,
		log:              log.New(os.Stdout, "[cors] ", log.LstdFlags),
	}
	if len(normOptions.AllowedOrigins) == 0 {
		// Default is all origins
		normOptions.AllowedOrigins = []string{"*"}
	}
	if len(normOptions.AllowedHeaders) == 1 {
		// Add some sensible defaults
		normOptions.AllowedHeaders = []string{"Origin", "Accept", "Content-Type"}
	}
	if len(normOptions.AllowedMethods) == 0 {
		// Default is simple methods
		normOptions.AllowedMethods = []string{"GET", "POST"}
	}

	if normOptions.Debug {
		normOptions.log.Printf("Options: %v", normOptions)
	}
	return &Cors{
		options: normOptions,
	}
}

// Default creates a new Cors handler with default options
func Default() *Cors {
	return New(Options{})
}

// Handler apply the CORS specification on the request, and add relevant CORS headers
// as necessary.
func (cors *Cors) Handler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			cors.logf("Handler: Preflight request")
			cors.handlePreflight(w, r)
			// Preflight requests are standalone and should stop the chain as some other
			// middleware may not handle OPTIONS requests correctly. One typical example
			// is authentication middleware ; OPTIONS requests won't carry authentication
			// headers (see #1)
		} else {
			cors.logf("Handler: Actual request")
			cors.handleActualRequest(w, r)
			h.ServeHTTP(w, r)
		}
	})
}

// Martini compatible handler
func (cors *Cors) HandlerFunc(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		cors.logf("HandlerFunc: Preflight request")
		cors.handlePreflight(w, r)
	} else {
		cors.logf("HandlerFunc: Actual request")
		cors.handleActualRequest(w, r)
	}
}

// Negroni compatible interface
func (cors *Cors) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if r.Method == "OPTIONS" {
		cors.logf("ServeHTTP: Preflight request")
		cors.handlePreflight(w, r)
		// Preflight requests are standalone and should stop the chain as some other
		// middleware may not handle OPTIONS requests correctly. One typical example
		// is authentication middleware ; OPTIONS requests won't carry authentication
		// headers (see #1)
	} else {
		cors.logf("ServeHTTP: Actual request")
		cors.handleActualRequest(w, r)
		next(w, r)
	}
}

// handlePreflight handles pre-flight CORS requests
func (cors *Cors) handlePreflight(w http.ResponseWriter, r *http.Request) {
	options := cors.options
	headers := w.Header()
	origin := r.Header.Get("Origin")

	if r.Method != "OPTIONS" {
		cors.logf("  Preflight aborted: %s!=OPTIONS", r.Method)
		return
	}
	if origin == "" {
		cors.logf("  Preflight aborted: empty origin")
		return
	}
	if !cors.isOriginAllowed(origin) {
		cors.logf("  Preflight aborted: origin '%s' not allowed", origin)
		return
	}

	reqMethod := r.Header.Get("Access-Control-Request-Method")
	if !cors.isMethodAllowed(reqMethod) {
		cors.logf("  Preflight aborted: method '%s' not allowed", reqMethod)
		return
	}
	reqHeaders := parseHeaderList(r.Header.Get("Access-Control-Request-Headers"))
	if !cors.areHeadersAllowed(reqHeaders) {
		cors.logf("  Preflight aborted: headers '%v' not allowed", reqHeaders)
		return
	}
	headers.Set("Access-Control-Allow-Origin", origin)
	headers.Add("Vary", "Origin")
	// Spec says: Since the list of methods can be unbounded, simply returning the method indicated
	// by Access-Control-Request-Method (if supported) can be enough
	headers.Set("Access-Control-Allow-Methods", strings.ToUpper(reqMethod))
	if len(reqHeaders) > 0 {

		// Spec says: Since the list of headers can be unbounded, simply returning supported headers
		// from Access-Control-Request-Headers can be enough
		headers.Set("Access-Control-Allow-Headers", strings.Join(reqHeaders, ", "))
	}
	if options.AllowCredentials {
		headers.Set("Access-Control-Allow-Credentials", "true")
	}
	if options.MaxAge > 0 {
		headers.Set("Access-Control-Max-Age", strconv.Itoa(options.MaxAge))
	}
	cors.logf("  Preflight response headers: %v", headers)
}

// handleActualRequest handles simple cross-origin requests, actual request or redirects
func (cors *Cors) handleActualRequest(w http.ResponseWriter, r *http.Request) {
	options := cors.options
	headers := w.Header()
	origin := r.Header.Get("Origin")

	if r.Method == "OPTIONS" {
		cors.logf("  Actual request no headers added: method == %s", r.Method)
		return
	}
	if origin == "" {
		cors.logf("  Actual request no headers added: missing origin")
		return
	}
	if !cors.isOriginAllowed(origin) {
		cors.logf("  Actual request no headers added: origin '%s' not allowed", origin)
		return
	}

	// Note that spec does define a way to specifically disallow a simple method like GET or
	// POST. Access-Control-Allow-Methods is only used for pre-flight requests and the
	// spec doesn't instruct to check the allowed methods for simple cross-origin requests.
	// We think it's a nice feature to be able to have control on those methods though.
	if !cors.isMethodAllowed(r.Method) {
		if cors.options.Debug {
			cors.logf("  Actual request no headers added: method '%s' not allowed",
				r.Method)
		}

		return
	}
	headers.Set("Access-Control-Allow-Origin", origin)
	headers.Add("Vary", "Origin")
	if len(options.ExposedHeaders) > 0 {
		headers.Set("Access-Control-Expose-Headers", strings.Join(options.ExposedHeaders, ", "))
	}
	if options.AllowCredentials {
		headers.Set("Access-Control-Allow-Credentials", "true")
	}
	cors.logf("  Actual response added headers: %v", headers)
}

// convenience method. checks if debugging is turned on before printing
func (cors *Cors) logf(format string, a ...interface{}) {
	if cors.options.Debug {
		cors.options.log.Printf(format, a...)
	}
}

// isOriginAllowed checks if a given origin is allowed to perform cross-domain requests
// on the endpoint
func (cors *Cors) isOriginAllowed(origin string) bool {
	allowedOrigins := cors.options.AllowedOrigins
	origin = strings.ToLower(origin)
	for _, allowedOrigin := range allowedOrigins {
		switch allowedOrigin {
		case "*":
			return true
		case origin:
			return true
		}
	}
	return false
}

// isMethodAllowed checks if a given method can be used as part of a cross-domain request
// on the endpoing
func (cors *Cors) isMethodAllowed(method string) bool {
	allowedMethods := cors.options.AllowedMethods
	if len(allowedMethods) == 0 {
		// If no method allowed, always return false, even for preflight request
		return false
	}
	method = strings.ToUpper(method)
	if method == "OPTIONS" {
		// Always allow preflight requests
		return true
	}
	for _, allowedMethod := range allowedMethods {
		if allowedMethod == method {
			return true
		}
	}
	return false
}

// areHeadersAllowed checks if a given list of headers are allowed to used within
// a cross-domain request.
func (cors *Cors) areHeadersAllowed(requestedHeaders []string) bool {
	if len(requestedHeaders) == 0 {
		return true
	}
	for _, header := range requestedHeaders {
		found := false
		for _, allowedHeader := range cors.options.AllowedHeaders {
			if allowedHeader == "*" || allowedHeader == header {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
