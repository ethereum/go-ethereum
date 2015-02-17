package otto

type _executionContext struct {
	LexicalEnvironment  _environment
	VariableEnvironment _environment
	this                *_object
	eval                bool // Replace this with kind?
}

func newExecutionContext(lexical _environment, variable _environment, this *_object) *_executionContext {
	return &_executionContext{
		LexicalEnvironment:  lexical,
		VariableEnvironment: variable,
		this:                this,
	}
}

func (self *_executionContext) getValue(name string) Value {
	strict := false
	return self.LexicalEnvironment.GetValue(name, strict)
}

func (self *_executionContext) setValue(name string, value Value, throw bool) {
	self.LexicalEnvironment.SetValue(name, value, throw)
}

func (self *_executionContext) newLexicalEnvironment(object *_object) (_environment, *_objectEnvironment) {
	// Get runtime from the object (for now)
	runtime := object.runtime
	previousLexical := self.LexicalEnvironment
	newLexical := runtime.newObjectEnvironment(object, self.LexicalEnvironment)
	self.LexicalEnvironment = newLexical
	return previousLexical, newLexical
}

func (self *_executionContext) newDeclarativeEnvironment(runtime *_runtime) _environment {
	previousLexical := self.LexicalEnvironment
	self.LexicalEnvironment = runtime.newDeclarativeEnvironment(self.LexicalEnvironment)
	return previousLexical
}
