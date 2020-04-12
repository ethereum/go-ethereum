package splunk

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

type Event struct {
	Time       int64       `json:"time"`                 // epoch time in seconds
	Host       string      `json:"host"`                 // hostname
	Source     string      `json:"source,omitempty"`     // optional description of the source of the event; typically the app's name
	SourceType string      `json:"sourcetype,omitempty"` // optional name of a Splunk parsing configuration; this is usually inferred by Splunk
	Index      string      `json:"index,omitempty"`      // optional name of the Splunk index to store the event in; not required if the token has a default index set in Splunk
	Event      interface{} `json:"event"`                // throw any useful key/val pairs here
}

type Client struct {
	HTTPClient *http.Client
	URL        string
	Hostname   string
	Token      string
	Source     string
	SourceType string
	Index      string
}

func NewClient(httpClient *http.Client, URL string, Token string, Source string, SourceType string, Index string) *Client {
	if httpClient == nil {
		tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: false}}
		httpClient = &http.Client{Timeout: time.Second * 20, Transport: tr}
	}
	hostname, _ := os.Hostname()
	c := &Client{
		HTTPClient: httpClient,
		URL:        URL,
		Hostname:   hostname,
		Token:      Token,
		Source:     Source,
		SourceType: SourceType,
		Index:      Index,
	}
	return c
}

func (c *Client) NewEvent(event interface{}, source string, sourcetype string, index string) *Event {
	e := &Event{
		Time:       time.Now().Unix(),
		Host:       c.Hostname,
		Source:     source,
		SourceType: sourcetype,
		Index:      index,
		Event:      event,
	}
	return e
}

func (c *Client) NewEventWithTime(t int64, event interface{}, source string, sourcetype string, index string) *Event {
	e := &Event{
		Time:       t,
		Host:       c.Hostname,
		Source:     source,
		SourceType: sourcetype,
		Index:      index,
		Event:      event,
	}
	return e
}

func (c *Client) Log(event interface{}) error {
	log := c.NewEvent(event, c.Source, c.SourceType, c.Index)
	return c.LogEvent(log)
}

func (c *Client) LogWithTime(t int64, event interface{}) error {
	log := c.NewEventWithTime(t, event, c.Source, c.SourceType, c.Index)
	return c.LogEvent(log)
}

func (c *Client) LogEvent(e *Event) error {
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	return c.doRequest(bytes.NewBuffer(b))
}

func (c *Client) LogEvents(events []*Event) error {
	buf := new(bytes.Buffer)
	for _, e := range events {
		b, err := json.Marshal(e)
		if err != nil {
			return err
		}
		buf.Write(b)
		buf.WriteString("\r\n\r\n")
	}
	return c.doRequest(buf)
}

func (c *Client) Writer() io.Writer {
	return &Writer{
		Client: c,
	}
}

func (c *Client) doRequest(b *bytes.Buffer) error {
	url := c.URL
	req, err := http.NewRequest("POST", url, b)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Splunk "+c.Token)

	res, err := c.HTTPClient.Do(req)
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
