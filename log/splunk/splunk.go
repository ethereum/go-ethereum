package splunk

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/go-stack/stack"
)

// Handler posts data to Splunk HTTP Event Collector.
type Handler struct {
	HTTPClient      *http.Client
	URL             string
	Hostname        string
	Token           string
	Source          string
	SourceType      string
	Index           string
	SkipTLSVerify   bool
	OriginalHandler log.Handler
	errors          chan error
	once            sync.Once
}

func (h *Handler) init() {
	if h.HTTPClient == nil {
		tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: h.SkipTLSVerify}}
		h.HTTPClient = &http.Client{Timeout: time.Second * 20, Transport: tr}
	}
	if h.Hostname == "" {
		h.Hostname, _ = os.Hostname()
	}
	if h.OriginalHandler != nil {
		h.errors = make(chan error, 10)

		go func() {
			for {
				err := <-h.errors

				_ = h.OriginalHandler.Log(&log.Record{
					Time: time.Now(),
					Lvl:  log.LvlWarn,
					Msg:  fmt.Sprintf("Error sending info to Splunk: %v", err),
					Ctx:  nil,
					Call: stack.Caller(0),
				})
			}
		}()
	}
}

type eventdata struct {
	Msg   string            `json:"message"`
	Level string            `json:"level"`
	Ctxt  map[string]string `json:"context"`
}

type event struct {
	Time       int64     `json:"time"`                 // epoch time in seconds
	Host       string    `json:"host"`                 // hostname
	Source     string    `json:"source,omitempty"`     // optional description of the source of the event; typically the app's name
	SourceType string    `json:"sourcetype,omitempty"` // optional name of a Splunk parsing configuration; this is usually inferred by Splunk
	Index      string    `json:"index,omitempty"`      // optional name of the Splunk index to store the event in; not required if the token has a default index set in Splunk
	Event      eventdata `json:"event"`                // throw any useful key/val pairs here
}

func (h *Handler) Log(r *log.Record) error {
	h.once.Do(h.init)

	context := make(map[string]string)
	for index := 0; index < len(r.Ctx); index++ {
		key := fmt.Sprintf("%v", r.Ctx[index])
		index++
		if r.Ctx[index] != nil {
			value := fmt.Sprintf("%v", r.Ctx[index])
			context[key] = value
		}
	}
	e := &event{
		Time:       r.Time.Unix(),
		Host:       h.Hostname,
		Source:     h.Source,
		SourceType: h.SourceType,
		Index:      h.Index,
		Event:      eventdata{Msg: r.Msg, Level: r.Lvl.String(), Ctxt: context},
	}
	b, err := json.Marshal(e)
	if err != nil {
		h.errors <- err
		return err
	}
	err = h.doRequest(bytes.NewBuffer(b))
	if err != nil {
		h.errors <- err
	}
	return err
}

func (h *Handler) doRequest(b *bytes.Buffer) error {
	url := h.URL
	req, err := http.NewRequest("POST", url, b)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Splunk "+h.Token)

	res, err := h.HTTPClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	switch res.StatusCode {
	case 200:
		io.Copy(ioutil.Discard, res.Body)
		return nil
	default:
		buf := new(bytes.Buffer)
		buf.ReadFrom(res.Body)
		responseBody := buf.String()
		err = errors.New(responseBody)
	}
	return err
}
