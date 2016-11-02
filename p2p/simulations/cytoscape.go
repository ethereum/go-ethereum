package simulations

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger/glog"
)

type CyData struct {
	Id     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	On     bool   `json:"on"`
}

type CyElement struct {
	Data    *CyData `json:"data"`
	Classes string  `json:"classes"`
	Group   string  `json:"group"`
	// selected: false, // whether the element is selected (default false)
	// selectable: true, // whether the selection state is mutable (default true)
	// locked: false, // when locked a node's position is immutable (default false)
	// grabbable: true, // whether the node can be grabbed and moved by the user
}

// type ConnEntry struct {
// }

// type MsgEntry struct {
// }

type Entry struct {
	Action string      `json:"action"`
	Type   string      `json:"type"`
	Object interface{} `json:"object"`
}

func (self *Entry) Stirng() string {
	return fmt.Sprintf("<Action: %v, Type: %v, Data: %v>\n", self.Action, self.Type, self.Object)
}

type CyUpdate struct {
	Add    []*CyElement `json:"add"`
	Remove []string     `json:"remove"`
}

func UpdateCy(j *Journal) *CyUpdate {
	added := []*CyElement{}
	removed := []string{}
	var el *CyElement
	update := func(ev *event.Event) bool {
		entry := ev.Data.(*Entry)
		glog.V(6).Infof("journal entry of %v: %v", ev.Time, entry)
		switch entry.Type {
		case "Node":
			el = &CyElement{Group: "nodes", Data: &CyData{Id: entry.Object.(*SimNode).ID.String()[0:lablen]}}
		case "Conn":
			// mutually exclusive directed edge (caller -> callee)
			source := entry.Object.(*SimConn).Caller.String()[0:lablen]
			target := entry.Object.(*SimConn).Callee.String()[0:lablen]
			first := source
			second := target
			if bytes.Compare([]byte(first), []byte(second)) > 1 {
				first = target
				second = source
			}
			id := fmt.Sprintf("%v-%v", first, second)
			el = &CyElement{Group: "edges", Data: &CyData{Id: id, Source: source, Target: target}}
		case "Know":
			// independent directed edge (peer0 registers peer1)
			source := entry.Object.(*Know).Subject.String()[0:lablen]
			target := entry.Object.(*Know).Object.String()[0:lablen]
			id := fmt.Sprintf("%v-%v-%v", source, target, "know")
			el = &CyElement{Group: "edges", Data: &CyData{Id: id, Source: source, Target: target}}
		}
		switch entry.Action {
		case "Add":
			added = append(added, el)
		case "Remove":
			removed = append(removed, el.Data.Id)
		case "On":
			el.Data.On = true
			added = append(added, el)
		case "Off":
			el.Data.On = false
			removed = append(removed, el.Data.Id)
		}
		return true
	}
	glog.V(6).Infof("journal read")
	j.Read(update)
	glog.V(6).Infof("journal read done")

	return &CyUpdate{
		Add:    added,
		Remove: removed,
	}
}
