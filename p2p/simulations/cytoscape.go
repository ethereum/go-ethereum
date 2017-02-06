package simulations

import (
	// "fmt"

	"github.com/ethereum/go-ethereum/event"
)

// TODO: to implement cytoscape global behav
type CyConfig struct {
}

type CyData struct {
	Id     string `json:"id"`
	Source string `json:"source,omitempty"`
	Target string `json:"target,omitempty"`
	Up     bool   `json:"up"`
}

type CyElement struct {
	Data    *CyData `json:"data"`
	Classes string  `json:"classes,omitempty"`
	Group   string  `json:"group"`
	// selected: false, // whether the element is selected (default false)
	// selectable: true, // whether the selection state is mutable (default true)
	// locked: false, // when locked a node's position is immutable (default false)
	// grabbable: true, // whether the node can be grabbed and moved by the user
}

type CyUpdate struct {
	Add     []*CyElement `json:"add"`
	Remove  []string     `json:"remove"`
	Message []string     `json:"message"`
}

func UpdateCy(conf *CyConfig, j *Journal) (*CyUpdate, error) {
	added := []*CyElement{}
	removed := []string{}
	messaged := []string{}
	var el *CyElement
	update := func(e *event.TypeMuxEvent) bool {
		entry := e.Data
		var action string
		if ev, ok := entry.(*NodeEvent); ok {
			el = &CyElement{Group: "nodes", Data: &CyData{Id: ev.node.Id.Label()}}
			action = ev.Action
		} else if ev, ok := entry.(*MsgEvent); ok {
			msg := ev.msg
			id := ConnLabel(msg.One, msg.Other)
			var source, target string
			source = msg.One.Label()
			target = msg.Other.Label()
			el = &CyElement{Group: "msgs", Data: &CyData{Id: id, Source: source, Target: target}}
			action = ev.Action
		} else if ev, ok := entry.(*ConnEvent); ok {
			// mutually exclusive directed edge (caller -> callee)
			conn := ev.conn
			id := ConnLabel(conn.One, conn.Other)
			var source, target string
			if conn.Reverse {
				source = conn.Other.Label()
				target = conn.One.Label()
			} else {
				source = conn.One.Label()
				target = conn.Other.Label()
			}
			el = &CyElement{Group: "edges", Data: &CyData{Id: id, Source: source, Target: target}}
			action = ev.Action
		} else {
			panic("unknown event type")
		}

		switch action {
		case "up":
			el.Data.Up = true
			added = append(added, el)
		case "down":
			el.Data.Up = false
			removed = append(removed, el.Data.Id)
		case "msg":
			el.Data.Up = true
			messaged = append(messaged, el.Data.Id)
		default:
			panic("unknown action")
		}
		return true
	}
	j.Read(update)

	return &CyUpdate{
		Add:     added,
		Remove:  removed,
		Message: messaged,
	}, nil
}
