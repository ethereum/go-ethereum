package duktape

// Must returns existing *Context or throw panic.
// It is highly recommended to use Must all the time.
func (d *Context) Must() *Context {
	if d.duk_context == nil {
		panic("[duktape] Context does not exists!\nYou cannot call any contexts methods after `DestroyHeap()` was called.")
	}
	return d
}
