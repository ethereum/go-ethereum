package duktape

import (
	"errors"
	"fmt"
	"time"
)

// DefineTimers defines `setTimeout`, `clearTimeout`, `setInterval`,
// `clearInterval` into global context.
func (d *Context) PushTimers() error {
	d.PushGlobalStash()
	// check if timers already exists
	if !d.HasPropString(-1, "timers") {
		d.PushObject()
		d.PutPropString(-2, "timers") // stash -> [ timers:{} ]
		d.Pop()

		d.PushGlobalGoFunction("setTimeout", setTimeout)
		d.PushGlobalGoFunction("setInterval", setInterval)
		d.PushGlobalGoFunction("clearTimeout", clearTimeout)
		d.PushGlobalGoFunction("clearInterval", clearTimeout)
		return nil
	} else {
		d.Pop()
		return errors.New("Timers are already defined")
	}
}

func (d *Context) FlushTimers() {
	d.PushGlobalStash()
	d.PushObject()
	d.PutPropString(-2, "timers") // stash -> [ timers:{} ]
	d.Pop()
}

func setTimeout(c *Context) int {
	id := c.pushTimer(0)
	timeout := c.ToNumber(1)
	if timeout < 1 {
		timeout = 1
	}
	go func(id float64) {
		<-time.After(time.Duration(timeout) * time.Millisecond)
		c.Lock()
		defer c.Unlock()
		if c.duk_context == nil {
			fmt.Println("[duktape] Warning!\nsetTimeout invokes callback after the context was destroyed.")
			return
		}

		// check if timer still exists
		c.putTimer(id)
		if c.GetType(-1).IsObject() {
			c.Pcall(0 /* nargs */)
		}
		c.dropTimer(id)
	}(id)
	c.PushNumber(id)
	return 1
}

func clearTimeout(c *Context) int {
	if c.GetType(0).IsNumber() {
		c.dropTimer(c.GetNumber(0))
		c.Pop()
	}
	return 0
}

func setInterval(c *Context) int {
	id := c.pushTimer(0)
	timeout := c.ToNumber(1)
	if timeout < 1 {
		timeout = 1
	}
	go func(id float64) {
		ticker := time.NewTicker(time.Duration(timeout) * time.Millisecond)
		for _ = range ticker.C {
			c.Lock()
			// check if duktape context exists
			if c.duk_context == nil {
				c.dropTimer(id)
				c.Pop()
				ticker.Stop()
				fmt.Println("[duktape] Warning!\nsetInterval invokes callback after the context was destroyed.")
				c.Unlock()
				continue
			}

			// check if timer still exists
			c.putTimer(id)
			if c.GetType(-1).IsObject() {
				c.Pcall(0 /* nargs */)
				c.Pop()
			} else {
				c.dropTimer(id)
				c.Pop()
				ticker.Stop()
			}
			c.Unlock()
		}
	}(id)
	c.PushNumber(id)
	return 1
}

func (d *Context) pushTimer(index int) float64 {
	id := d.timerIndex.get()

	d.PushGlobalStash()
	d.GetPropString(-1, "timers")
	d.PushNumber(id)
	d.Dup(index)
	d.PutProp(-3)
	d.Pop2()

	return id
}

func (d *Context) dropTimer(id float64) {
	d.PushGlobalStash()
	d.GetPropString(-1, "timers")
	d.PushNumber(id)
	d.DelProp(-2)
	d.Pop2()
}

func (d *Context) putTimer(id float64) {
	d.PushGlobalStash()           // stash -> [ ..., timers: { <id>: { func: true } } ]
	d.GetPropString(-1, "timers") // stash -> [ ..., timers: { <id>: { func: true } } }, { <id>: { func: true } ]
	d.PushNumber(id)
	d.GetProp(-2) // stash -> [ ..., timers: { <id>: { func: true } } }, { <id>: { func: true }, { func: true } ]
	d.Replace(-3)
	d.Pop()
}
