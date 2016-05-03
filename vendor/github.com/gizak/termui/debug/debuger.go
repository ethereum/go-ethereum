// Copyright 2016 Zack Guo <gizak@icloud.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

package debug

import (
	"fmt"
	"net/http"

	"golang.org/x/net/websocket"
)

type Server struct {
	Port string
	Addr string
	Path string
	Msg  chan string
	chs  []chan string
}

type Client struct {
	Port string
	Addr string
	Path string
	ws   *websocket.Conn
}

var defaultPort = ":8080"

func NewServer() *Server {
	return &Server{
		Port: defaultPort,
		Addr: "localhost",
		Path: "/echo",
		Msg:  make(chan string),
		chs:  make([]chan string, 0),
	}
}

func NewClient() Client {
	return Client{
		Port: defaultPort,
		Addr: "localhost",
		Path: "/echo",
	}
}

func (c Client) ConnectAndListen() error {
	ws, err := websocket.Dial("ws://"+c.Addr+c.Port+c.Path, "", "http://"+c.Addr)
	if err != nil {
		return err
	}
	defer ws.Close()

	var m string
	for {
		err := websocket.Message.Receive(ws, &m)
		if err != nil {
			fmt.Print(err)
			return err
		}
		fmt.Print(m)
	}
}

func (s *Server) ListenAndServe() error {
	http.Handle(s.Path, websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()

		mc := make(chan string)
		s.chs = append(s.chs, mc)

		for m := range mc {
			websocket.Message.Send(ws, m)
		}
	}))

	go func() {
		for msg := range s.Msg {
			for _, c := range s.chs {
				go func(a chan string) {
					a <- msg
				}(c)
			}
		}
	}()

	return http.ListenAndServe(s.Port, nil)
}

func (s *Server) Log(msg string) {
	go func() { s.Msg <- msg }()
}

func (s *Server) Logf(format string, a ...interface{}) {
	s.Log(fmt.Sprintf(format, a...))
}

var DefaultServer = NewServer()
var DefaultClient = NewClient()

func ListenAndServe() error {
	return DefaultServer.ListenAndServe()
}

func ConnectAndListen() error {
	return DefaultClient.ConnectAndListen()
}

func Log(msg string) {
	DefaultServer.Log(msg)
}

func Logf(format string, a ...interface{}) {
	DefaultServer.Logf(format, a...)
}
