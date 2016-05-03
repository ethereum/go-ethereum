package scpd

import (
	"encoding/xml"
	"strings"
)

const (
	SCPDXMLNamespace = "urn:schemas-upnp-org:service-1-0"
)

func cleanWhitespace(s *string) {
	*s = strings.TrimSpace(*s)
}

// SCPD is the service description as described by section 2.5 "Service
// description" in
// http://upnp.org/specs/arch/UPnP-arch-DeviceArchitecture-v1.1.pdf
type SCPD struct {
	XMLName        xml.Name        `xml:"scpd"`
	ConfigId       string          `xml:"configId,attr"`
	SpecVersion    SpecVersion     `xml:"specVersion"`
	Actions        []Action        `xml:"actionList>action"`
	StateVariables []StateVariable `xml:"serviceStateTable>stateVariable"`
}

// Clean attempts to remove stray whitespace etc. in the structure. It seems
// unfortunately common for stray whitespace to be present in SCPD documents,
// this method attempts to make it easy to clean them out.
func (scpd *SCPD) Clean() {
	cleanWhitespace(&scpd.ConfigId)
	for i := range scpd.Actions {
		scpd.Actions[i].clean()
	}
	for i := range scpd.StateVariables {
		scpd.StateVariables[i].clean()
	}
}

func (scpd *SCPD) GetStateVariable(variable string) *StateVariable {
	for i := range scpd.StateVariables {
		v := &scpd.StateVariables[i]
		if v.Name == variable {
			return v
		}
	}
	return nil
}

func (scpd *SCPD) GetAction(action string) *Action {
	for i := range scpd.Actions {
		a := &scpd.Actions[i]
		if a.Name == action {
			return a
		}
	}
	return nil
}

// SpecVersion is part of a SCPD document, describes the version of the
// specification that the data adheres to.
type SpecVersion struct {
	Major int32 `xml:"major"`
	Minor int32 `xml:"minor"`
}

type Action struct {
	Name      string     `xml:"name"`
	Arguments []Argument `xml:"argumentList>argument"`
}

func (action *Action) clean() {
	cleanWhitespace(&action.Name)
	for i := range action.Arguments {
		action.Arguments[i].clean()
	}
}

func (action *Action) InputArguments() []*Argument {
	var result []*Argument
	for i := range action.Arguments {
		arg := &action.Arguments[i]
		if arg.IsInput() {
			result = append(result, arg)
		}
	}
	return result
}

func (action *Action) OutputArguments() []*Argument {
	var result []*Argument
	for i := range action.Arguments {
		arg := &action.Arguments[i]
		if arg.IsOutput() {
			result = append(result, arg)
		}
	}
	return result
}

type Argument struct {
	Name                 string `xml:"name"`
	Direction            string `xml:"direction"`            // in|out
	RelatedStateVariable string `xml:"relatedStateVariable"` // ?
	Retval               string `xml:"retval"`               // ?
}

func (arg *Argument) clean() {
	cleanWhitespace(&arg.Name)
	cleanWhitespace(&arg.Direction)
	cleanWhitespace(&arg.RelatedStateVariable)
	cleanWhitespace(&arg.Retval)
}

func (arg *Argument) IsInput() bool {
	return arg.Direction == "in"
}

func (arg *Argument) IsOutput() bool {
	return arg.Direction == "out"
}

type StateVariable struct {
	Name              string             `xml:"name"`
	SendEvents        string             `xml:"sendEvents,attr"` // yes|no
	Multicast         string             `xml:"multicast,attr"`  // yes|no
	DataType          DataType           `xml:"dataType"`
	DefaultValue      string             `xml:"defaultValue"`
	AllowedValueRange *AllowedValueRange `xml:"allowedValueRange"`
	AllowedValues     []string           `xml:"allowedValueList>allowedValue"`
}

func (v *StateVariable) clean() {
	cleanWhitespace(&v.Name)
	cleanWhitespace(&v.SendEvents)
	cleanWhitespace(&v.Multicast)
	v.DataType.clean()
	cleanWhitespace(&v.DefaultValue)
	if v.AllowedValueRange != nil {
		v.AllowedValueRange.clean()
	}
	for i := range v.AllowedValues {
		cleanWhitespace(&v.AllowedValues[i])
	}
}

type AllowedValueRange struct {
	Minimum string `xml:"minimum"`
	Maximum string `xml:"maximum"`
	Step    string `xml:"step"`
}

func (r *AllowedValueRange) clean() {
	cleanWhitespace(&r.Minimum)
	cleanWhitespace(&r.Maximum)
	cleanWhitespace(&r.Step)
}

type DataType struct {
	Name string `xml:",chardata"`
	Type string `xml:"type,attr"`
}

func (dt *DataType) clean() {
	cleanWhitespace(&dt.Name)
	cleanWhitespace(&dt.Type)
}
