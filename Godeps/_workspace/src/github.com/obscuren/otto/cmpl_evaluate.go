package otto

import (
	"strconv"
)

func (self *_runtime) cmpl_evaluate_nodeProgram(node *_nodeProgram) Value {
	self.cmpl_functionDeclaration(node.functionList)
	self.cmpl_variableDeclaration(node.varList)
	return self.cmpl_evaluate_nodeStatementList(node.body)
}

func (self *_runtime) cmpl_call_nodeFunction(function *_object, environment *_functionEnvironment, node *_nodeFunctionLiteral, this Value, argumentList []Value) Value {

	indexOfParameterName := make([]string, len(argumentList))
	// function(abc, def, ghi)
	// indexOfParameterName[0] = "abc"
	// indexOfParameterName[1] = "def"
	// indexOfParameterName[2] = "ghi"
	// ...

	argumentsFound := false
	for index, name := range node.parameterList {
		if name == "arguments" {
			argumentsFound = true
		}
		value := UndefinedValue()
		if index < len(argumentList) {
			value = argumentList[index]
			indexOfParameterName[index] = name
		}
		self.localSet(name, value)
	}

	if !argumentsFound {
		arguments := self.newArgumentsObject(indexOfParameterName, environment, len(argumentList))
		arguments.defineProperty("callee", toValue_object(function), 0101, false)
		environment.arguments = arguments
		self.localSet("arguments", toValue_object(arguments))
		for index, _ := range argumentList {
			if index < len(node.parameterList) {
				continue
			}
			indexAsString := strconv.FormatInt(int64(index), 10)
			arguments.defineProperty(indexAsString, argumentList[index], 0111, false)
		}
	}

	self.cmpl_functionDeclaration(node.functionList)
	self.cmpl_variableDeclaration(node.varList)

	result := self.cmpl_evaluate_nodeStatement(node.body)
	if result.isResult() {
		return result
	}

	return UndefinedValue()
}

func (self *_runtime) cmpl_functionDeclaration(list []*_nodeFunctionLiteral) {
	executionContext := self._executionContext(0)
	eval := executionContext.eval
	environment := executionContext.VariableEnvironment

	for _, function := range list {
		name := function.name
		value := self.cmpl_evaluate_nodeExpression(function)
		if !environment.HasBinding(name) {
			environment.CreateMutableBinding(name, eval == true)
		}
		// TODO 10.5.5.e
		environment.SetMutableBinding(name, value, false) // TODO strict
	}
}

func (self *_runtime) cmpl_variableDeclaration(list []string) {
	executionContext := self._executionContext(0)
	eval := executionContext.eval
	environment := executionContext.VariableEnvironment

	for _, name := range list {
		if !environment.HasBinding(name) {
			environment.CreateMutableBinding(name, eval == true)
			environment.SetMutableBinding(name, UndefinedValue(), false) // TODO strict
		}
	}
}
