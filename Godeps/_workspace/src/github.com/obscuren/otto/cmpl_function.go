package otto

// _cmpl_nodeCallFunction
type _cmpl_nodeCallFunction struct {
	node             *_nodeFunctionLiteral
	scopeEnvironment _environment // Can be either Lexical or Variable
}

func new_nodeCallFunction(node *_nodeFunctionLiteral, scopeEnvironment _environment) *_cmpl_nodeCallFunction {
	self := &_cmpl_nodeCallFunction{
		node: node,
	}
	self.scopeEnvironment = scopeEnvironment
	return self
}

func (self _cmpl_nodeCallFunction) Dispatch(function *_object, environment *_functionEnvironment, runtime *_runtime, this Value, argumentList []Value, _ bool) Value {
	return runtime.cmpl_call_nodeFunction(function, environment, self.node, this, argumentList)
}

func (self _cmpl_nodeCallFunction) ScopeEnvironment() _environment {
	return self.scopeEnvironment
}

func (self _cmpl_nodeCallFunction) Source(object *_object) string {
	return self.node.source
}

func (self0 _cmpl_nodeCallFunction) clone(clone *_clone) _callFunction {
	return _cmpl_nodeCallFunction{
		node:             self0.node,
		scopeEnvironment: clone.environment(self0.scopeEnvironment),
	}
}

// ---

func (runtime *_runtime) newNodeFunctionObject(node *_nodeFunctionLiteral, scopeEnvironment _environment) *_object {
	self := runtime.newClassObject("Function")
	self.value = _functionObject{
		call:      new_nodeCallFunction(node, scopeEnvironment),
		construct: defaultConstructFunction,
	}
	self.defineProperty("length", toValue_int(len(node.parameterList)), 0000, false)
	return self
}
