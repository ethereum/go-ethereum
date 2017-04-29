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
	Classes string   `json:"classes,omitempty"`
	Group   string   `json:"group"`
	Control bool     `json:"control"`
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

	switch entry.(type) {
	case *NodeControlEvent, *NodeEvent:
		var data *SimData
		var control bool
		nce, ok := entry.(*NodeControlEvent)
		if ok {
			data = &SimData{Id: nce.Node.Id.String()}
			data.Up = nce.Up
			control = true
		} else {
			ne := entry.(*NodeEvent)
			data = &SimData{Id: ne.Node.Id.String()}
			data.Up = ne.Up
			control = false
		}
		el = &SimElement{Group: "nodes", Data: data}
		el.Control = control
		if el.Data.Up {
			update.Add = append(update.Add, el)
		} else {
			update.Remove = append(update.Remove, el)
		}
	case *MsgControlEvent, *MsgEvent:
		var control bool
		var msg *Msg
		mce, ok := entry.(*MsgControlEvent)
		if ok {
			msg = mce.Message
			control = true
		} else {
			me := entry.(*MsgEvent)
			msg = me.Message
			control = false
		}
		id := ConnLabel(msg.One, msg.Other)
		var source, target string
		source = msg.One.String()
		target = msg.Other.String()
		el = &SimElement{Group: "msgs", Data: &SimData{Id: id, Source: source, Target: target}}
		el.Data.Up = true
		el.Control = control
		update.Message = append(update.Message, el)
	case *ConnControlEvent, *ConnEvent:
		var control bool
		var conn *Conn
		var up bool
		cce, ok := entry.(*ConnControlEvent)
		if ok {
			conn = cce.Connection
			up = cce.Up
			control = true
		} else {
			ce := entry.(*ConnEvent)
			conn = ce.Connection
			up = ce.Up
			control = false
		}
		// mutually exclusive directed edge (caller -> callee)
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
		el.Control = control
		el.Data.Up = up
		if up {
			update.Add = append(update.Add, el)
		} else {
			update.Remove = append(update.Remove, el)
		}
	default:
		return nil, fmt.Errorf("unknown event type: %T", entry)
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
