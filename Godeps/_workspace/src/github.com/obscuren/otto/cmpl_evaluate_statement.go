package otto

import (
	"fmt"
	"runtime"

	"github.com/robertkrimen/otto/token"
)

func (self *_runtime) cmpl_evaluate_nodeStatement(node _nodeStatement) Value {
	// Allow interpreter interruption
	// If the Interrupt channel is nil, then
	// we avoid runtime.Gosched() overhead (if any)
	// FIXME: Test this
	if self.Otto.Interrupt != nil {
		runtime.Gosched()
		select {
		case value := <-self.Otto.Interrupt:
			value()
		default:
		}
	}

	switch node := node.(type) {

	case *_nodeBlockStatement:
		// FIXME If result is break, then return the empty value?
		return self.cmpl_evaluate_nodeStatementList(node.list)

	case *_nodeBranchStatement:
		target := node.label
		switch node.branch { // FIXME Maybe node.kind? node.operator?
		case token.BREAK:
			return toValue(newBreakResult(target))
		case token.CONTINUE:
			return toValue(newContinueResult(target))
		}

	case *_nodeDebuggerStatement:
		return Value{} // Nothing happens.

	case *_nodeDoWhileStatement:
		return self.cmpl_evaluate_nodeDoWhileStatement(node)

	case *_nodeEmptyStatement:
		return Value{}

	case *_nodeExpressionStatement:
		return self.cmpl_evaluate_nodeExpression(node.expression)

	case *_nodeForInStatement:
		return self.cmpl_evaluate_nodeForInStatement(node)

	case *_nodeForStatement:
		return self.cmpl_evaluate_nodeForStatement(node)

	case *_nodeIfStatement:
		return self.cmpl_evaluate_nodeIfStatement(node)

	case *_nodeLabelledStatement:
		self.labels = append(self.labels, node.label)
		defer func() {
			if len(self.labels) > 0 {
				self.labels = self.labels[:len(self.labels)-1] // Pop the label
			} else {
				self.labels = nil
			}
		}()
		return self.cmpl_evaluate_nodeStatement(node.statement)

	case *_nodeReturnStatement:
		if node.argument != nil {
			return toValue(newReturnResult(self.GetValue(self.cmpl_evaluate_nodeExpression(node.argument))))
		}
		return toValue(newReturnResult(UndefinedValue()))

	case *_nodeSwitchStatement:
		return self.cmpl_evaluate_nodeSwitchStatement(node)

	case *_nodeThrowStatement:
		value := self.GetValue(self.cmpl_evaluate_nodeExpression(node.argument))
		panic(newException(value))

	case *_nodeTryStatement:
		return self.cmpl_evaluate_nodeTryStatement(node)

	case *_nodeVariableStatement:
		// Variables are already defined, this is initialization only
		for _, variable := range node.list {
			self.cmpl_evaluate_nodeVariableExpression(variable.(*_nodeVariableExpression))
		}
		return Value{}

	case *_nodeWhileStatement:
		return self.cmpl_evaluate_nodeWhileStatement(node)

	case *_nodeWithStatement:
		return self.cmpl_evaluate_nodeWithStatement(node)

	}

	panic(fmt.Errorf("Here be dragons: evaluate_nodeStatement(%T)", node))
}

func (self *_runtime) cmpl_evaluate_nodeStatementList(list []_nodeStatement) Value {
	var result Value
	for _, node := range list {
		value := self.cmpl_evaluate_nodeStatement(node)
		switch value._valueType {
		case valueResult:
			return value
		case valueEmpty:
		default:
			// We have GetValue here to (for example) trigger a
			// ReferenceError (of the not defined variety)
			// Not sure if this is the best way to error out early
			// for such errors or if there is a better way
			// TODO Do we still need this?
			result = self.GetValue(value)
		}
	}
	return result
}

func (self *_runtime) cmpl_evaluate_nodeDoWhileStatement(node *_nodeDoWhileStatement) Value {

	labels := append(self.labels, "")
	self.labels = nil

	test := node.test

	result := Value{}
resultBreak:
	for {
		for _, node := range node.body {
			value := self.cmpl_evaluate_nodeStatement(node)
			switch value._valueType {
			case valueResult:
				switch value.evaluateBreakContinue(labels) {
				case resultReturn:
					return value
				case resultBreak:
					break resultBreak
				case resultContinue:
					goto resultContinue
				}
			case valueEmpty:
			default:
				result = value
			}
		}
	resultContinue:
		if !self.GetValue(self.cmpl_evaluate_nodeExpression(test)).isTrue() {
			// Stahp: do ... while (false)
			break
		}
	}
	return result
}

func (self *_runtime) cmpl_evaluate_nodeForInStatement(node *_nodeForInStatement) Value {

	labels := append(self.labels, "")
	self.labels = nil

	source := self.cmpl_evaluate_nodeExpression(node.source)
	sourceValue := self.GetValue(source)

	switch sourceValue._valueType {
	case valueUndefined, valueNull:
		return emptyValue()
	}

	sourceObject := self.toObject(sourceValue)

	into := node.into
	body := node.body

	result := Value{}
	object := sourceObject
	for object != nil {
		enumerateValue := Value{}
		object.enumerate(false, func(name string) bool {
			into := self.cmpl_evaluate_nodeExpression(into)
			// In the case of: for (var abc in def) ...
			if into.reference() == nil {
				identifier := toString(into)
				// TODO Should be true or false (strictness) depending on context
				into = toValue(getIdentifierReference(self.LexicalEnvironment(), identifier, false))
			}
			self.PutValue(into.reference(), toValue_string(name))
			for _, node := range body {
				value := self.cmpl_evaluate_nodeStatement(node)
				switch value._valueType {
				case valueResult:
					switch value.evaluateBreakContinue(labels) {
					case resultReturn:
						enumerateValue = value
						return false
					case resultBreak:
						object = nil
						return false
					case resultContinue:
						return true
					}
				case valueEmpty:
				default:
					enumerateValue = value
				}
			}
			return true
		})
		if object == nil {
			break
		}
		object = object.prototype
		if !enumerateValue.isEmpty() {
			result = enumerateValue
		}
	}
	return result
}

func (self *_runtime) cmpl_evaluate_nodeForStatement(node *_nodeForStatement) Value {

	labels := append(self.labels, "")
	self.labels = nil

	initializer := node.initializer
	test := node.test
	update := node.update
	body := node.body

	if initializer != nil {
		initialResult := self.cmpl_evaluate_nodeExpression(initializer)
		self.GetValue(initialResult) // Side-effect trigger
	}

	result := Value{}
resultBreak:
	for {
		if test != nil {
			testResult := self.cmpl_evaluate_nodeExpression(test)
			testResultValue := self.GetValue(testResult)
			if toBoolean(testResultValue) == false {
				break
			}
		}
		for _, node := range body {
			value := self.cmpl_evaluate_nodeStatement(node)
			switch value._valueType {
			case valueResult:
				switch value.evaluateBreakContinue(labels) {
				case resultReturn:
					return value
				case resultBreak:
					break resultBreak
				case resultContinue:
					goto resultContinue
				}
			case valueEmpty:
			default:
				result = value
			}
		}
	resultContinue:
		if update != nil {
			updateResult := self.cmpl_evaluate_nodeExpression(update)
			self.GetValue(updateResult) // Side-effect trigger
		}
	}
	return result
}

func (self *_runtime) cmpl_evaluate_nodeIfStatement(node *_nodeIfStatement) Value {
	test := self.cmpl_evaluate_nodeExpression(node.test)
	testValue := self.GetValue(test)
	if toBoolean(testValue) {
		return self.cmpl_evaluate_nodeStatement(node.consequent)
	} else if node.alternate != nil {
		return self.cmpl_evaluate_nodeStatement(node.alternate)
	}

	return Value{}
}

func (self *_runtime) cmpl_evaluate_nodeSwitchStatement(node *_nodeSwitchStatement) Value {

	labels := append(self.labels, "")
	self.labels = nil

	discriminantResult := self.cmpl_evaluate_nodeExpression(node.discriminant)
	target := node.default_

	for index, clause := range node.body {
		test := clause.test
		if test != nil {
			if self.calculateComparison(token.STRICT_EQUAL, discriminantResult, self.cmpl_evaluate_nodeExpression(test)) {
				target = index
				break
			}
		}
	}

	result := Value{}
	if target != -1 {
		for _, clause := range node.body[target:] {
			for _, statement := range clause.consequent {
				value := self.cmpl_evaluate_nodeStatement(statement)
				switch value._valueType {
				case valueResult:
					switch value.evaluateBreak(labels) {
					case resultReturn:
						return value
					case resultBreak:
						return Value{}
					}
				case valueEmpty:
				default:
					result = value
				}
			}
		}
	}

	return result
}

func (self *_runtime) cmpl_evaluate_nodeTryStatement(node *_nodeTryStatement) Value {
	tryCatchValue, exception := self.tryCatchEvaluate(func() Value {
		return self.cmpl_evaluate_nodeStatement(node.body)
	})

	if exception && node.catch != nil {

		lexicalEnvironment := self._executionContext(0).newDeclarativeEnvironment(self)
		defer func() {
			self._executionContext(0).LexicalEnvironment = lexicalEnvironment
		}()
		// TODO If necessary, convert TypeError<runtime> => TypeError
		// That, is, such errors can be thrown despite not being JavaScript "native"
		self.localSet(node.catch.parameter, tryCatchValue)

		// FIXME node.CatchParameter
		// FIXME node.Catch
		tryCatchValue, exception = self.tryCatchEvaluate(func() Value {
			return self.cmpl_evaluate_nodeStatement(node.catch.body)
		})
	}

	if node.finally != nil {
		finallyValue := self.cmpl_evaluate_nodeStatement(node.finally)
		if finallyValue.isResult() {
			return finallyValue
		}
	}

	if exception {
		panic(newException(tryCatchValue))
	}

	return tryCatchValue
}

func (self *_runtime) cmpl_evaluate_nodeWhileStatement(node *_nodeWhileStatement) Value {

	test := node.test
	body := node.body
	labels := append(self.labels, "")
	self.labels = nil

	result := Value{}
resultBreakContinue:
	for {
		if !self.GetValue(self.cmpl_evaluate_nodeExpression(test)).isTrue() {
			// Stahp: while (false) ...
			break
		}
		for _, node := range body {
			value := self.cmpl_evaluate_nodeStatement(node)
			switch value._valueType {
			case valueResult:
				switch value.evaluateBreakContinue(labels) {
				case resultReturn:
					return value
				case resultBreak:
					break resultBreakContinue
				case resultContinue:
					continue resultBreakContinue
				}
			case valueEmpty:
			default:
				result = value
			}
		}
	}
	return result
}

func (self *_runtime) cmpl_evaluate_nodeWithStatement(node *_nodeWithStatement) Value {
	object := self.cmpl_evaluate_nodeExpression(node.object)
	objectValue := self.GetValue(object)
	previousLexicalEnvironment, lexicalEnvironment := self._executionContext(0).newLexicalEnvironment(self.toObject(objectValue))
	lexicalEnvironment.ProvideThis = true
	defer func() {
		self._executionContext(0).LexicalEnvironment = previousLexicalEnvironment
	}()

	return self.cmpl_evaluate_nodeStatement(node.body)
}
