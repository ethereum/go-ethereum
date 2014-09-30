package websocket

import (
	"net/http"

	"github.com/ethereum/eth-go/ethlog"

	ws "code.google.com/p/go.net/websocket"
)

var logger = ethlog.NewLogger("WS")

// Chat server.
type Server struct {
	httpServ  string
	pattern   string
	messages  []*Message
	clients   map[int]*Client
	addCh     chan *Client
	delCh     chan *Client
	sendAllCh chan string
	doneCh    chan bool
	errCh     chan error
	msgFunc   MsgFunc
}

// Create new chat server.
func NewServer(pattern, httpServ string) *Server {
	clients := make(map[int]*Client)
	addCh := make(chan *Client)
	delCh := make(chan *Client)
	sendAllCh := make(chan string)
	doneCh := make(chan bool)
	errCh := make(chan error)

	return &Server{
		httpServ,
		pattern,
		nil,
		clients,
		addCh,
		delCh,
		sendAllCh,
		doneCh,
		errCh,
		nil,
	}
}

func (s *Server) Add(c *Client) {
	s.addCh <- c
}

func (s *Server) Del(c *Client) {
	s.delCh <- c
}

func (s *Server) SendAll(msg string) {
	s.sendAllCh <- msg
}

func (s *Server) Done() {
	s.doneCh <- true
}

func (s *Server) Err(err error) {
	s.errCh <- err
}

func (s *Server) servHTTP() {
	logger.Debugln("Serving http", s.httpServ)
	err := http.ListenAndServe(s.httpServ, nil)

	logger.Warnln(err)
}

func (s *Server) MessageFunc(f MsgFunc) {
	s.msgFunc = f
}

// Listen and serve.
// It serves client connection and broadcast request.
func (s *Server) Listen() {
	logger.Debugln("Listening server...")

	// ws handler
	onConnected := func(ws *ws.Conn) {
		defer func() {
			err := ws.Close()
			if err != nil {
				s.errCh <- err
			}
		}()

		client := NewClient(ws, s)
		client.onMessage = s.msgFunc
		s.Add(client)
		client.Listen()
	}
	// Disable Origin check. Request don't need to come necessarily from origin.
	http.HandleFunc(s.pattern, func(w http.ResponseWriter, req *http.Request) {
		s := ws.Server{Handler: ws.Handler(onConnected)}
		s.ServeHTTP(w, req)
	})
	logger.Debugln("Created handler")

	go s.servHTTP()

	for {
		select {

		// Add new a client
		case c := <-s.addCh:
			s.clients[c.id] = c

		// del a client
		case c := <-s.delCh:
			delete(s.clients, c.id)

		case err := <-s.errCh:
			logger.Debugln("Error:", err.Error())

		case <-s.doneCh:
			return
		}
	}
}
