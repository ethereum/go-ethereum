package otto

func (rt *_runtime) newErrorObject(name string, message Value) *_object {
	self := rt.newClassObject("Error")
	if message.IsDefined() {
		msg := message.string()
		self.defineProperty("message", toValue_string(msg), 0111, false)
		self.value = newError(rt, name, msg)
	} else {
		self.value = newError(rt, name)
	}
	return self
}
