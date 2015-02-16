package otto

func (runtime *_runtime) newErrorObject(message Value) *_object {
	self := runtime.newClassObject("Error")
	if message.IsDefined() {
		self.defineProperty("message", toValue_string(toString(message)), 0111, false)
	}
	return self
}
