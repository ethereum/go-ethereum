package txpool

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"time"
)

const (
	pongWait      = 10 * time.Second
	pingPeriod    = 5 * time.Second
	writeDeadline = 10 * time.Second
)

var upgraderConn = &websocket.Upgrader{}

type Client struct {
	clientBroadcast *Broadcast
	websocketConn   *websocket.Conn
	sendMessage     chan []byte
}

func (c *Client) readPump() {
	defer func() {
		c.clientBroadcast.unregisterClient <- c
		c.websocketConn.Close()
	}()

	err := c.websocketConn.SetReadDeadline(time.Now().Add(pongWait))
	if err != nil {
		return
	}

	c.websocketConn.SetPongHandler(func(string) error {
		fmt.Println("Received Pong")
		err = c.websocketConn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err = c.websocketConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Println("Error", err)
			}
			log.Println("Error", err)
			break
		}
	}
}

func (c *Client) writePump() {
	newTicker := time.NewTicker(pingPeriod)
	defer func() {
		newTicker.Stop()
		c.websocketConn.Close()
	}()
	for {
		select {
		case messageData, stateData := <-c.sendMessage:
			err := c.websocketConn.SetWriteDeadline(time.Now().Add(pingPeriod))
			if err != nil {
				return
			}
			if !stateData {
				log.Println("Close the channel")
				c.websocketConn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			c.websocketConn.WriteMessage(websocket.BinaryMessage, messageData)

			err = c.websocketConn.SetWriteDeadline(time.Time{})
			if err != nil {
				return
			}
		case <-newTicker.C:
			log.Println("Send Ping")
			err := c.websocketConn.SetWriteDeadline(time.Now().Add(writeDeadline))
			if err != nil {
				return
			}

			c.websocketConn.WriteMessage(websocket.PingMessage, nil)
			err = c.websocketConn.SetWriteDeadline(time.Time{})
			if err != nil {
				return
			}
		}
	}
}

func serveWs(broadcast *Broadcast, w http.ResponseWriter, r *http.Request) {
	websocketConn, err := upgraderConn.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	newClient := &Client{
		clientBroadcast: broadcast,
		websocketConn:   websocketConn,
		sendMessage:     make(chan []byte, 256),
	}
	newClient.clientBroadcast.registerClient <- newClient
	go newClient.writePump()
	go newClient.readPump()
}

func messageHandler(broadcast *Broadcast, b []byte) {
	broadcast.broadcastMessage <- b
}
