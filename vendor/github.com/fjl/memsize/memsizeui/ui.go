package memsizeui

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fjl/memsize"
)

type Handler struct {
	init     sync.Once
	mux      http.ServeMux
	mu       sync.Mutex
	reports  map[int]Report
	roots    map[string]interface{}
	reportID int
}

type Report struct {
	ID       int
	Date     time.Time
	Duration time.Duration
	RootName string
	Sizes    memsize.Sizes
}

type templateInfo struct {
	Roots     []string
	Reports   map[int]Report
	PathDepth int
	Data      interface{}
}

func (ti *templateInfo) Link(path ...string) string {
	prefix := strings.Repeat("../", ti.PathDepth)
	return prefix + strings.Join(path, "")
}

func (h *Handler) Add(name string, v interface{}) {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		panic("root must be non-nil pointer")
	}
	h.mu.Lock()
	if h.roots == nil {
		h.roots = make(map[string]interface{})
	}
	h.roots[name] = v
	h.mu.Unlock()
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.init.Do(func() {
		h.reports = make(map[int]Report)
		h.mux.HandleFunc("/", h.handleRoot)
		h.mux.HandleFunc("/scan", h.handleScan)
		h.mux.HandleFunc("/report/", h.handleReport)
	})
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) templateInfo(r *http.Request, data interface{}) *templateInfo {
	h.mu.Lock()
	roots := make([]string, 0, len(h.roots))
	for name := range h.roots {
		roots = append(roots, name)
	}
	h.mu.Unlock()
	sort.Strings(roots)

	return &templateInfo{
		Roots:     roots,
		Reports:   h.reports,
		PathDepth: strings.Count(r.URL.Path, "/") - 1,
		Data:      data,
	}
}

func (h *Handler) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	serveHTML(w, rootTemplate, http.StatusOK, h.templateInfo(r, nil))
}

func (h *Handler) handleScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "invalid HTTP method, want POST", http.StatusMethodNotAllowed)
		return
	}
	ti := h.templateInfo(r, "Unknown root")
	id, ok := h.scan(r.URL.Query().Get("root"))
	if !ok {
		serveHTML(w, notFoundTemplate, http.StatusNotFound, ti)
		return
	}
	w.Header().Add("Location", ti.Link(fmt.Sprintf("report/%d", id)))
	w.WriteHeader(http.StatusSeeOther)
}

func (h *Handler) handleReport(w http.ResponseWriter, r *http.Request) {
	var id int
	fmt.Sscan(strings.TrimPrefix(r.URL.Path, "/report/"), &id)
	h.mu.Lock()
	report, ok := h.reports[id]
	h.mu.Unlock()

	if !ok {
		serveHTML(w, notFoundTemplate, http.StatusNotFound, h.templateInfo(r, "Report not found"))
	} else {
		serveHTML(w, reportTemplate, http.StatusOK, h.templateInfo(r, report))
	}
}

func (h *Handler) scan(root string) (int, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	val, ok := h.roots[root]
	if !ok {
		return 0, false
	}
	id := h.reportID
	start := time.Now()
	sizes := memsize.Scan(val)
	h.reports[id] = Report{
		ID:       id,
		RootName: root,
		Date:     start.Truncate(1 * time.Second),
		Duration: time.Since(start),
		Sizes:    sizes,
	}
	h.reportID++
	return id, true
}

func serveHTML(w http.ResponseWriter, tpl *template.Template, status int, ti *templateInfo) {
	w.Header().Set("content-type", "text/html")
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, ti); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	buf.WriteTo(w)
}
