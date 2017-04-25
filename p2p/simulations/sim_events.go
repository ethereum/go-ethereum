package simulations

import (
	"fmt"

	"github.com/ethereum/go-ethereum/event"
)

// TODO: to implement simulation global behav
type SimConfig struct {
}

type SimData struct {
	Id     string `json:"id"`
	Source string `json:"source,omitempty"`
	Target string `json:"target,omitempty"`
	Up     bool   `json:"up"`
}

type SimElement struct {
	Data    *SimData `json:"data"`
	Classes string  `json:"classes,omitempty"`
	Group   string  `json:"group"`
	// selected: false, // whether the element is selected (default false)
	// selectable: true, // whether the selection state is mutable (default true)
	// locked: false, // when locked a node's position is immutable (default false)
	// grabbable: true, // whether the node can be grabbed and moved by the user
}

type SimUpdate struct {
	Add     []*SimElement `json:"add"`
	Remove  []*SimElement `json:"remove"`
	Message []*SimElement `json:"message"`
}

func NewSimUpdate(e *event.TypeMuxEvent) (*SimUpdate, error) {
	var update SimUpdate
	var el *SimElement
	entry := e.Data
	var action string
	if ev, ok := entry.(*NodeEvent); ok {
		el = &SimElement{Group: "nodes", Data: &SimData{Id: ev.node.Id.String()}}
		action = ev.Action
	} else if ev, ok := entry.(*MsgEvent); ok {
		msg := ev.msg
		id := ConnLabel(msg.One, msg.Other)
		var source, target string
		source = msg.One.String()
		target = msg.Other.String()
		el = &SimElement{Group: "msgs", Data: &SimData{Id: id, Source: source, Target: target}}
		action = ev.Action
	} else if ev, ok := entry.(*ConnEvent); ok {
		// mutually exclusive directed edge (caller -> callee)
		conn := ev.conn
		id := ConnLabel(conn.One, conn.Other)
		var source, target string
		if conn.Reverse {
			source = conn.Other.String()
			target = conn.One.String()
		} else {
			source = conn.One.String()
			target = conn.Other.String()
		}
		el = &SimElement{Group: "edges", Data: &SimData{Id: id, Source: source, Target: target}}
		action = ev.Action
	} else {
		return nil, fmt.Errorf("unknown event type: %T", entry)
	}

	switch action {
	case "up":
		el.Data.Up = true
		update.Add = append(update.Add, el)
	case "down":
		el.Data.Up = false
		update.Remove = append(update.Remove, el)
	case "msg":
		el.Data.Up = true
		update.Message = append(update.Message, el)
	default:
		return nil, fmt.Errorf("unknown action: %q", action)
	}

	return &update, nil
}

func UpdateSim(conf *SimConfig, j *Journal) (*SimUpdate, error) {
	var update SimUpdate
	j.Read(func(e *event.TypeMuxEvent) bool {
		u, err := NewSimUpdate(e)
		if err != nil {
			panic(err.Error())
		}
		update.Add = append(update.Add, u.Add...)
		update.Remove = append(update.Remove, u.Remove...)
		update.Message = append(update.Message, u.Message...)
		return true
	})
	return &update, nil
}
