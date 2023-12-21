package tracer

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

type Action interface {
	Type() string
	Children() []Action
	Parent() Action
	Depth() int
	Log()
	Has(string) bool
	Context() common.Address
	Code() common.Address

	AddChildren(Action)
}

type Call struct {
	ParentValue Action `json:"-"`
	DepthValue  int    `json:"depth,omitempty"`

	TypeValue     string   `json:"type,omitempty"`
	CallType      string   `json:"callType,omitempty"`
	ChildrenValue []Action `json:"children,omitempty"`

	ContextValue     common.Address `json:"context,omitempty"`
	CodeValue        common.Address `json:"code,omitempty"`
	ForwardedContext common.Address `json:"forwardedContext,omitempty"`
	ForwardedCode    common.Address `json:"forwardedCode,omitempty"`

	From   common.Address `json:"from,omitempty"`
	To     common.Address `json:"to,omitempty"`
	Value  string         `json:"value,omitempty"`
	In     []byte         `json:"-"`
	Out    []byte         `json:"-"`
	InHex  string         `json:"in,omitempty"`
	OutHex string         `json:"out,omitempty"`
}

func (c *Call) Type() string {
	return c.CallType
}

func (c *Call) Children() []Action {
	return c.ChildrenValue
}

func (c *Call) Context() common.Address {
	return c.ContextValue
}

func (c *Call) Code() common.Address {
	return c.CodeValue
}

func (c *Call) Depth() int {
	return c.DepthValue
}

func (c *Call) Parent() Action {
	return c.ParentValue
}

func (c *Call) AddChildren(a Action) {
	c.ChildrenValue = append(c.ChildrenValue, a)
}

func (c *Call) Log() {
	fmt.Printf("%s- %s %s to %s (%s:%s) (%d,%d) (%d)\n", strings.Repeat(" ", c.DepthValue), c.Type(), c.From.String(), c.To.String(), c.Context().String(), c.Code().String(), len(c.In), len(c.Out), len(c.ChildrenValue))
	for _, subcall := range c.ChildrenValue {
		subcall.Log()
	}
}

func (c *Call) Has(typ string) bool {
	if c.Type() == typ {
		return true
	}
	for _, chld := range c.ChildrenValue {
		if chld.Has(typ) {
			return true
		}
	}
	return false
}

type Event struct {
	ParentValue Action `json:"-"`
	DepthValue  int    `json:"depth,omitempty"`

	TypeValue string `json:"type,omitempty"`
	LogType   string `json:"logType,omitempty"`

	ContextValue common.Address `json:"context,omitempty"`
	CodeValue    common.Address `json:"code,omitempty"`

	Data    []byte         `json:"-"`
	DataHex string         `json:"data,omitempty"`
	Topics  []common.Hash  `json:"topics,omitempty"`
	From    common.Address `json:"from,omitempty"`
}

func (c *Event) Type() string {
	return c.LogType
}

func (c *Event) Children() []Action {
	return []Action{}
}

func (c *Event) Context() common.Address {
	return c.ContextValue
}

func (c *Event) Code() common.Address {
	return c.CodeValue
}

func (c *Event) Depth() int {
	return c.DepthValue
}

func (c *Event) Parent() Action {
	return c.ParentValue
}

func (c *Event) AddChildren(a Action) {
}

func (c *Event) Log() {
	fmt.Printf("%s- %s (%s:%s) \n", strings.Repeat(" ", c.DepthValue), c.Type(), c.Context().String(), c.Code().String())
}

func (c *Event) Has(typ string) bool {
	return c.Type() == typ
}

type Revert struct {
	ParentValue Action `json:"-"`
	DepthValue  int    `json:"depth,omitempty"`

	TypeValue string `json:"type,omitempty"`
	ErrorType string `json:"errorType,omitempty"`

	ContextValue common.Address `json:"context,omitempty"`
	CodeValue    common.Address `json:"code,omitempty"`

	Data    []byte         `json:"-"`
	DataHex string         `json:"data,omitempty"`
	From    common.Address `json:"from,omitempty"`
}

func (r *Revert) Type() string {
	return r.ErrorType
}

func (r *Revert) Children() []Action {
	return []Action{}
}

func (r *Revert) Context() common.Address {
	return r.ContextValue
}

func (r *Revert) Code() common.Address {
	return r.CodeValue
}

func (r *Revert) Depth() int {
	return r.DepthValue
}

func (r *Revert) Parent() Action {
	return r.ParentValue
}

func (r *Revert) AddChildren(a Action) {
}

func (r *Revert) Log() {
	fmt.Printf("%s- %s (%s:%s) %x\n", strings.Repeat(" ", r.DepthValue), r.Type(), r.Context().String(), r.Code().String(), r.Data)
}

func (r *Revert) Has(typ string) bool {
	return r.Type() == typ
}
