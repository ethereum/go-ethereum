package simulations

import (
	"fmt"
	"time"
)

type EventType string

const (
	EventTypeNode EventType = "node"
	EventTypeConn EventType = "conn"
	EventTypeMsg  EventType = "msg"
)

type Event struct {
	Type    EventType `json:"type"`
	Time    time.Time `json:"time"`
	Control bool      `json:"control"`

	Node *Node `json:"node,omitempty"`
	Conn *Conn `json:"conn,omitempty"`
	Msg  *Msg  `json:"msg,omitempty"`
}

func NewEvent(v interface{}) *Event {
	event := &Event{Time: time.Now()}
	switch v := v.(type) {
	case *Node:
		event.Type = EventTypeNode
		event.Node = v
	case *Conn:
		event.Type = EventTypeConn
		event.Conn = v
	case *Msg:
		event.Type = EventTypeMsg
		event.Msg = v
	default:
		panic(fmt.Sprintf("invalid event type: %T", v))
	}
	return event
}

func ControlEvent(v interface{}) *Event {
	event := NewEvent(v)
	event.Control = true
	return event
}

func (e *Event) String() string {
	switch e.Type {
	case EventTypeNode:
		return fmt.Sprintf("<node-event> id: %s up: %t", e.Node.ID().Label(), e.Node.Up)
	case EventTypeConn:
		return fmt.Sprintf("<conn-event> nodes: %s->%s up: %t", e.Conn.One.Label(), e.Conn.Other.Label(), e.Conn.Up)
	case EventTypeMsg:
		return fmt.Sprintf("<msg-event> nodes: %s->%s code: %d, received: %t", e.Msg.One.Label(), e.Msg.Other.Label(), e.Msg.Code, e.Msg.Received)
	default:
		return ""
	}
}
