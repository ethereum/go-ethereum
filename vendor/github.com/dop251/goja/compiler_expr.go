package goja

import (
	"fmt"
	"github.com/dop251/goja/ast"
	"github.com/dop251/goja/file"
	"github.com/dop251/goja/token"
	"regexp"
)

var (
	octalRegexp = regexp.MustCompile(`^0[0-7]`)
)

type compiledExpr interface {
	emitGetter(putOnStack bool)
	emitSetter(valueExpr compiledExpr)
	emitUnary(prepare, body func(), postfix, putOnStack bool)
	deleteExpr() compiledExpr
	constant() bool
	addSrcMap()
}

type compiledExprOrRef interface {
	compiledExpr
	emitGetterOrRef()
}

type compiledCallExpr struct {
	baseCompiledExpr
	args   []compiledExpr
	callee compiledExpr
}

type compiledObjectLiteral struct {
	baseCompiledExpr
	expr *ast.ObjectLiteral
}

type compiledArrayLiteral struct {
	baseCompiledExpr
	expr *ast.ArrayLiteral
}

type compiledRegexpLiteral struct {
	baseCompiledExpr
	expr *ast.RegExpLiteral
}

type compiledLiteral struct {
	baseCompiledExpr
	val Value
}

type compiledAssignExpr struct {
	baseCompiledExpr
	left, right compiledExpr
	operator    token.Token
}

type deleteGlobalExpr struct {
	baseCompiledExpr
	name string
}

type deleteVarExpr struct {
	baseCompiledExpr
	name string
}

type deletePropExpr struct {
	baseCompiledExpr
	left compiledExpr
	name string
}

type deleteElemExpr struct {
	baseCompiledExpr
	left, member compiledExpr
}

type constantExpr struct {
	baseCompiledExpr
	val Value
}

type baseCompiledExpr struct {
	c      *compiler
	offset int
}

type compiledIdentifierExpr struct {
	baseCompiledExpr
	name string
}

type compiledFunctionLiteral struct {
	baseCompiledExpr
	expr   *ast.FunctionLiteral
	isExpr bool
}

type compiledBracketExpr struct {
	baseCompiledExpr
	left, member compiledExpr
}

type compiledThisExpr struct {
	baseCompiledExpr
}

type compiledNewExpr struct {
	baseCompiledExpr
	callee compiledExpr
	args   []compiledExpr
}

type compiledSequenceExpr struct {
	baseCompiledExpr
	sequence []compiledExpr
}

type compiledUnaryExpr struct {
	baseCompiledExpr
	operand  compiledExpr
	operator token.Token
	postfix  bool
}

type compiledConditionalExpr struct {
	baseCompiledExpr
	test, consequent, alternate compiledExpr
}

type compiledLogicalOr struct {
	baseCompiledExpr
	left, right compiledExpr
}

type compiledLogicalAnd struct {
	baseCompiledExpr
	left, right compiledExpr
}

type compiledBinaryExpr struct {
	baseCompiledExpr
	left, right compiledExpr
	operator    token.Token
}

type compiledVariableExpr struct {
	baseCompiledExpr
	name        string
	initializer compiledExpr
	expr        *ast.VariableExpression
}

type compiledEnumGetExpr struct {
	baseCompiledExpr
}

type defaultDeleteExpr struct {
	baseCompiledExpr
	expr compiledExpr
}

func (e *defaultDeleteExpr) emitGetter(putOnStack bool) {
	e.expr.emitGetter(false)
	if putOnStack {
		e.c.emit(loadVal(e.c.p.defineLiteralValue(valueTrue)))
	}
}

func (c *compiler) compileExpression(v ast.Expression) compiledExpr {
	// log.Printf("compileExpression: %T", v)
	switch v := v.(type) {
	case nil:
		return nil
	case *ast.AssignExpression:
		return c.compileAssignExpression(v)
	case *ast.NumberLiteral:
		return c.compileNumberLiteral(v)
	case *ast.StringLiteral:
		return c.compileStringLiteral(v)
	case *ast.BooleanLiteral:
		return c.compileBooleanLiteral(v)
	case *ast.NullLiteral:
		r := &compiledLiteral{
			val: _null,
		}
		r.init(c, v.Idx0())
		return r
	case *ast.Identifier:
		return c.compileIdentifierExpression(v)
	case *ast.CallExpression:
		return c.compileCallExpression(v)
	case *ast.ObjectLiteral:
		return c.compileObjectLiteral(v)
	case *ast.ArrayLiteral:
		return c.compileArrayLiteral(v)
	case *ast.RegExpLiteral:
		return c.compileRegexpLiteral(v)
	case *ast.VariableExpression:
		return c.compileVariableExpression(v)
	case *ast.BinaryExpression:
		return c.compileBinaryExpression(v)
	case *ast.UnaryExpression:
		return c.compileUnaryExpression(v)
	case *ast.ConditionalExpression:
		return c.compileConditionalExpression(v)
	case *ast.FunctionLiteral:
		return c.compileFunctionLiteral(v, true)
	case *ast.DotExpression:
		r := &compiledDotExpr{
			left: c.compileExpression(v.Left),
			name: v.Identifier.Name,
		}
		r.init(c, v.Idx0())
		return r
	case *ast.BracketExpression:
		r := &compiledBracketExpr{
			left:   c.compileExpression(v.Left),
			member: c.compileExpression(v.Member),
		}
		r.init(c, v.Idx0())
		return r
	case *ast.ThisExpression:
		r := &compiledThisExpr{}
		r.init(c, v.Idx0())
		return r
	case *ast.SequenceExpression:
		return c.compileSequenceExpression(v)
	case *ast.NewExpression:
		return c.compileNewExpression(v)
	default:
		panic(fmt.Errorf("Unknown expression type: %T", v))
	}
}

func (e *baseCompiledExpr) constant() bool {
	return false
}

func (e *baseCompiledExpr) init(c *compiler, idx file.Idx) {
	e.c = c
	e.offset = int(idx) - 1
}

func (e *baseCompiledExpr) emitSetter(valueExpr compiledExpr) {
	e.c.throwSyntaxError(e.offset, "Not a valid left-value expression")
}

func (e *baseCompiledExpr) deleteExpr() compiledExpr {
	r := &constantExpr{
		val: valueTrue,
	}
	r.init(e.c, file.Idx(e.offset+1))
	return r
}

func (e *baseCompiledExpr) emitUnary(prepare, body func(), postfix bool, putOnStack bool) {
	e.c.throwSyntaxError(e.offset, "Not a valid left-value expression")
}

func (e *baseCompiledExpr) addSrcMap() {
	if e.offset > 0 {
		e.c.p.srcMap = append(e.c.p.srcMap, srcMapItem{pc: len(e.c.p.code), srcPos: e.offset})
	}
}

func (e *constantExpr) emitGetter(putOnStack bool) {
	if putOnStack {
		e.addSrcMap()
		e.c.emit(loadVal(e.c.p.defineLiteralValue(e.val)))
	}
}

func (e *compiledIdentifierExpr) emitGetter(putOnStack bool) {
	e.addSrcMap()
	if idx, found, noDynamics := e.c.scope.lookupName(e.name); noDynamics {
		if found {
			if putOnStack {
				e.c.emit(getLocal(idx))
			}
		} else {
			panic("No dynamics and not found")
		}
	} else {
		if found {
			e.c.emit(getVar{name: e.name, idx: idx})
		} else {
			e.c.emit(getVar1(e.name))
		}
		if !putOnStack {
			e.c.emit(pop)
		}
	}
}

func (e *compiledIdentifierExpr) emitGetterOrRef() {
	e.addSrcMap()
	if idx, found, noDynamics := e.c.scope.lookupName(e.name); noDynamics {
		if found {
			e.c.emit(getLocal(idx))
		} else {
			panic("No dynamics and not found")
		}
	} else {
		if found {
			e.c.emit(getVar{name: e.name, idx: idx, ref: true})
		} else {
			e.c.emit(getVar1Callee(e.name))
		}
	}
}

func (c *compiler) emitVarSetter1(name string, offset int, emitRight func(isRef bool)) {
	if c.scope.strict {
		c.checkIdentifierLName(name, offset)
	}

	if idx, found, noDynamics := c.scope.lookupName(name); noDynamics {
		emitRight(false)
		if found {
			c.emit(setLocal(idx))
		} else {
			if c.scope.strict {
				c.emit(setGlobalStrict(name))
			} else {
				c.emit(setGlobal(name))
			}
		}
	} else {
		if found {
			c.emit(resolveVar{name: name, idx: idx, strict: c.scope.strict})
			emitRight(true)
			c.emit(putValue)
		} else {
			if c.scope.strict {
				c.emit(resolveVar1Strict(name))
			} else {
				c.emit(resolveVar1(name))
			}
			emitRight(true)
			c.emit(putValue)
		}
	}
}

func (c *compiler) emitVarSetter(name string, offset int, valueExpr compiledExpr) {
	c.emitVarSetter1(name, offset, func(bool) {
		c.emitExpr(valueExpr, true)
	})
}

func (e *compiledVariableExpr) emitSetter(valueExpr compiledExpr) {
	e.c.emitVarSetter(e.name, e.offset, valueExpr)
}

func (e *compiledIdentifierExpr) emitSetter(valueExpr compiledExpr) {
	e.c.emitVarSetter(e.name, e.offset, valueExpr)
}

func (e *compiledIdentifierExpr) emitUnary(prepare, body func(), postfix, putOnStack bool) {
	if putOnStack {
		e.c.emitVarSetter1(e.name, e.offset, func(isRef bool) {
			e.c.emit(loadUndef)
			if isRef {
				e.c.emit(getValue)
			} else {
				e.emitGetter(true)
			}
			if prepare != nil {
				prepare()
			}
			if !postfix {
				body()
			}
			e.c.emit(rdupN(1))
			if postfix {
				body()
			}
		})
		e.c.emit(pop)
	} else {
		e.c.emitVarSetter1(e.name, e.offset, func(isRef bool) {
			if isRef {
				e.c.emit(getValue)
			} else {
				e.emitGetter(true)
			}
			body()
		})
		e.c.emit(pop)
	}
}

func (e *compiledIdentifierExpr) deleteExpr() compiledExpr {
	if e.c.scope.strict {
		e.c.throwSyntaxError(e.offset, "Delete of an unqualified identifier in strict mode")
		panic("Unreachable")
	}
	if _, found, noDynamics := e.c.scope.lookupName(e.name); noDynamics {
		if !found {
			r := &deleteGlobalExpr{
				name: e.name,
			}
			r.init(e.c, file.Idx(0))
			return r
		} else {
			r := &constantExpr{
				val: valueFalse,
			}
			r.init(e.c, file.Idx(0))
			return r
		}
	} else {
		r := &deleteVarExpr{
			name: e.name,
		}
		r.init(e.c, file.Idx(e.offset+1))
		return r
	}
}

type compiledDotExpr struct {
	baseCompiledExpr
	left compiledExpr
	name string
}

func (e *compiledDotExpr) emitGetter(putOnStack bool) {
	e.left.emitGetter(true)
	e.addSrcMap()
	e.c.emit(getProp(e.name))
	if !putOnStack {
		e.c.emit(pop)
	}
}

func (e *compiledDotExpr) emitSetter(valueExpr compiledExpr) {
	e.left.emitGetter(true)
	valueExpr.emitGetter(true)
	if e.c.scope.strict {
		e.c.emit(setPropStrict(e.name))
	} else {
		e.c.emit(setProp(e.name))
	}
}

func (e *compiledDotExpr) emitUnary(prepare, body func(), postfix, putOnStack bool) {
	if !putOnStack {
		e.left.emitGetter(true)
		e.c.emit(dup)
		e.c.emit(getProp(e.name))
		body()
		if e.c.scope.strict {
			e.c.emit(setPropStrict(e.name), pop)
		} else {
			e.c.emit(setProp(e.name), pop)
		}
	} else {
		if !postfix {
			e.left.emitGetter(true)
			e.c.emit(dup)
			e.c.emit(getProp(e.name))
			if prepare != nil {
				prepare()
			}
			body()
			if e.c.scope.strict {
				e.c.emit(setPropStrict(e.name))
			} else {
				e.c.emit(setProp(e.name))
			}
		} else {
			e.c.emit(loadUndef)
			e.left.emitGetter(true)
			e.c.emit(dup)
			e.c.emit(getProp(e.name))
			if prepare != nil {
				prepare()
			}
			e.c.emit(rdupN(2))
			body()
			if e.c.scope.strict {
				e.c.emit(setPropStrict(e.name))
			} else {
				e.c.emit(setProp(e.name))
			}
			e.c.emit(pop)
		}
	}
}

func (e *compiledDotExpr) deleteExpr() compiledExpr {
	r := &deletePropExpr{
		left: e.left,
		name: e.name,
	}
	r.init(e.c, file.Idx(0))
	return r
}

func (e *compiledBracketExpr) emitGetter(putOnStack bool) {
	e.left.emitGetter(true)
	e.member.emitGetter(true)
	e.addSrcMap()
	e.c.emit(getElem)
	if !putOnStack {
		e.c.emit(pop)
	}
}

func (e *compiledBracketExpr) emitSetter(valueExpr compiledExpr) {
	e.left.emitGetter(true)
	e.member.emitGetter(true)
	valueExpr.emitGetter(true)
	if e.c.scope.strict {
		e.c.emit(setElemStrict)
	} else {
		e.c.emit(setElem)
	}
}

func (e *compiledBracketExpr) emitUnary(prepare, body func(), postfix, putOnStack bool) {
	if !putOnStack {
		e.left.emitGetter(true)
		e.member.emitGetter(true)
		e.c.emit(dupN(1), dupN(1))
		e.c.emit(getElem)
		body()
		if e.c.scope.strict {
			e.c.emit(setElemStrict, pop)
		} else {
			e.c.emit(setElem, pop)
		}
	} else {
		if !postfix {
			e.left.emitGetter(true)
			e.member.emitGetter(true)
			e.c.emit(dupN(1), dupN(1))
			e.c.emit(getElem)
			if prepare != nil {
				prepare()
			}
			body()
			if e.c.scope.strict {
				e.c.emit(setElemStrict)
			} else {
				e.c.emit(setElem)
			}
		} else {
			e.c.emit(loadUndef)
			e.left.emitGetter(true)
			e.member.emitGetter(true)
			e.c.emit(dupN(1), dupN(1))
			e.c.emit(getElem)
			if prepare != nil {
				prepare()
			}
			e.c.emit(rdupN(3))
			body()
			if e.c.scope.strict {
				e.c.emit(setElemStrict, pop)
			} else {
				e.c.emit(setElem, pop)
			}
		}
	}
}

func (e *compiledBracketExpr) deleteExpr() compiledExpr {
	r := &deleteElemExpr{
		left:   e.left,
		member: e.member,
	}
	r.init(e.c, file.Idx(0))
	return r
}

func (e *deleteElemExpr) emitGetter(putOnStack bool) {
	e.left.emitGetter(true)
	e.member.emitGetter(true)
	e.addSrcMap()
	if e.c.scope.strict {
		e.c.emit(deleteElemStrict)
	} else {
		e.c.emit(deleteElem)
	}
	if !putOnStack {
		e.c.emit(pop)
	}
}

func (e *deletePropExpr) emitGetter(putOnStack bool) {
	e.left.emitGetter(true)
	e.addSrcMap()
	if e.c.scope.strict {
		e.c.emit(deletePropStrict(e.name))
	} else {
		e.c.emit(deleteProp(e.name))
	}
	if !putOnStack {
		e.c.emit(pop)
	}
}

func (e *deleteVarExpr) emitGetter(putOnStack bool) {
	/*if e.c.scope.strict {
		e.c.throwSyntaxError(e.offset, "Delete of an unqualified identifier in strict mode")
		return
	}*/
	e.c.emit(deleteVar(e.name))
	if !putOnStack {
		e.c.emit(pop)
	}
}

func (e *deleteGlobalExpr) emitGetter(putOnStack bool) {
	/*if e.c.scope.strict {
		e.c.throwSyntaxError(e.offset, "Delete of an unqualified identifier in strict mode")
		return
	}*/

	e.c.emit(deleteGlobal(e.name))
	if !putOnStack {
		e.c.emit(pop)
	}
}

func (e *compiledAssignExpr) emitGetter(putOnStack bool) {
	e.addSrcMap()
	switch e.operator {
	case token.ASSIGN:
		e.left.emitSetter(e.right)
	case token.PLUS:
		e.left.emitUnary(nil, func() {
			e.right.emitGetter(true)
			e.c.emit(add)
		}, false, putOnStack)
		return
	case token.MINUS:
		e.left.emitUnary(nil, func() {
			e.right.emitGetter(true)
			e.c.emit(sub)
		}, false, putOnStack)
		return
	case token.MULTIPLY:
		e.left.emitUnary(nil, func() {
			e.right.emitGetter(true)
			e.c.emit(mul)
		}, false, putOnStack)
		return
	case token.SLASH:
		e.left.emitUnary(nil, func() {
			e.right.emitGetter(true)
			e.c.emit(div)
		}, false, putOnStack)
		return
	case token.REMAINDER:
		e.left.emitUnary(nil, func() {
			e.right.emitGetter(true)
			e.c.emit(mod)
		}, false, putOnStack)
		return
	case token.OR:
		e.left.emitUnary(nil, func() {
			e.right.emitGetter(true)
			e.c.emit(or)
		}, false, putOnStack)
		return
	case token.AND:
		e.left.emitUnary(nil, func() {
			e.right.emitGetter(true)
			e.c.emit(and)
		}, false, putOnStack)
		return
	case token.EXCLUSIVE_OR:
		e.left.emitUnary(nil, func() {
			e.right.emitGetter(true)
			e.c.emit(xor)
		}, false, putOnStack)
		return
	case token.SHIFT_LEFT:
		e.left.emitUnary(nil, func() {
			e.right.emitGetter(true)
			e.c.emit(sal)
		}, false, putOnStack)
		return
	case token.SHIFT_RIGHT:
		e.left.emitUnary(nil, func() {
			e.right.emitGetter(true)
			e.c.emit(sar)
		}, false, putOnStack)
		return
	case token.UNSIGNED_SHIFT_RIGHT:
		e.left.emitUnary(nil, func() {
			e.right.emitGetter(true)
			e.c.emit(shr)
		}, false, putOnStack)
		return
	default:
		panic(fmt.Errorf("Unknown assign operator: %s", e.operator.String()))
	}
	if !putOnStack {
		e.c.emit(pop)
	}
}

func (e *compiledLiteral) emitGetter(putOnStack bool) {
	if putOnStack {
		e.addSrcMap()
		e.c.emit(loadVal(e.c.p.defineLiteralValue(e.val)))
	}
}

func (e *compiledLiteral) constant() bool {
	return true
}

func (e *compiledFunctionLiteral) emitGetter(putOnStack bool) {
	e.c.newScope()
	savedBlockStart := e.c.blockStart
	savedPrg := e.c.p
	e.c.p = &Program{
		src: e.c.p.src,
	}
	e.c.blockStart = 0

	if e.expr.Name != nil {
		e.c.p.funcName = e.expr.Name.Name
	}
	block := e.c.block
	e.c.block = nil
	defer func() {
		e.c.block = block
	}()

	if !e.c.scope.strict {
		e.c.scope.strict = e.c.isStrictStatement(e.expr.Body)
	}

	if e.c.scope.strict {
		if e.expr.Name != nil {
			e.c.checkIdentifierLName(e.expr.Name.Name, int(e.expr.Name.Idx)-1)
		}
		for _, item := range e.expr.ParameterList.List {
			e.c.checkIdentifierName(item.Name, int(item.Idx)-1)
			e.c.checkIdentifierLName(item.Name, int(item.Idx)-1)
		}
	}

	length := len(e.expr.ParameterList.List)

	for _, item := range e.expr.ParameterList.List {
		_, unique := e.c.scope.bindNameShadow(item.Name)
		if !unique && e.c.scope.strict {
			e.c.throwSyntaxError(int(item.Idx)-1, "Strict mode function may not have duplicate parameter names (%s)", item.Name)
			return
		}
	}
	paramsCount := len(e.c.scope.names)
	e.c.compileDeclList(e.expr.DeclarationList, true)
	var needCallee bool
	var calleeIdx uint32
	if e.isExpr && e.expr.Name != nil {
		if idx, ok := e.c.scope.bindName(e.expr.Name.Name); ok {
			calleeIdx = idx
			needCallee = true
		}
	}
	lenBefore := len(e.c.scope.names)
	namesBefore := make([]string, 0, lenBefore)
	for key, _ := range e.c.scope.names {
		namesBefore = append(namesBefore, key)
	}
	maxPreambleLen := 2
	e.c.p.code = make([]instruction, maxPreambleLen)
	if needCallee {
		e.c.emit(loadCallee, setLocalP(calleeIdx))
	}

	e.c.compileFunctions(e.expr.DeclarationList)
	e.c.markBlockStart()
	e.c.compileStatement(e.expr.Body, false)

	if e.c.blockStart >= len(e.c.p.code)-1 || e.c.p.code[len(e.c.p.code)-1] != ret {
		e.c.emit(loadUndef, ret)
	}

	if !e.c.scope.dynamic && !e.c.scope.accessed {
		// log.Printf("Function can use inline stash")
		l := 0
		if !e.c.scope.strict && e.c.scope.thisNeeded {
			l = 2
			e.c.p.code = e.c.p.code[maxPreambleLen-2:]
			e.c.p.code[1] = boxThis
		} else {
			l = 1
			e.c.p.code = e.c.p.code[maxPreambleLen-1:]
		}
		e.c.convertFunctionToStashless(e.c.p.code, paramsCount)
		for i, _ := range e.c.p.srcMap {
			e.c.p.srcMap[i].pc -= maxPreambleLen - l
		}
	} else {
		l := 1 + len(e.c.scope.names)
		if e.c.scope.argsNeeded {
			l += 2
		}
		if !e.c.scope.strict && e.c.scope.thisNeeded {
			l++
		}

		code := make([]instruction, l+len(e.c.p.code)-maxPreambleLen)
		code[0] = enterFunc(length)
		for name, nameIdx := range e.c.scope.names {
			code[nameIdx+1] = bindName(name)
		}
		pos := 1 + len(e.c.scope.names)

		if !e.c.scope.strict && e.c.scope.thisNeeded {
			code[pos] = boxThis
			pos++
		}

		if e.c.scope.argsNeeded {
			if e.c.scope.strict {
				code[pos] = createArgsStrict(length)
			} else {
				code[pos] = createArgs(length)
			}
			pos++
			idx, exists := e.c.scope.names["arguments"]
			if !exists {
				panic("No arguments")
			}
			code[pos] = setLocalP(idx)
			pos++
		}

		copy(code[l:], e.c.p.code[maxPreambleLen:])
		e.c.p.code = code
		for i, _ := range e.c.p.srcMap {
			e.c.p.srcMap[i].pc += l - maxPreambleLen
		}
	}

	strict := e.c.scope.strict
	p := e.c.p
	// e.c.p.dumpCode()
	e.c.popScope()
	e.c.p = savedPrg
	e.c.blockStart = savedBlockStart
	name := ""
	if e.expr.Name != nil {
		name = e.expr.Name.Name
	}
	e.c.emit(&newFunc{prg: p, length: uint32(length), name: name, srcStart: uint32(e.expr.Idx0() - 1), srcEnd: uint32(e.expr.Idx1() - 1), strict: strict})
	if !putOnStack {
		e.c.emit(pop)
	}
}

func (c *compiler) compileFunctionLiteral(v *ast.FunctionLiteral, isExpr bool) compiledExpr {
	if v.Name != nil && c.scope.strict {
		c.checkIdentifierLName(v.Name.Name, int(v.Name.Idx)-1)
	}
	r := &compiledFunctionLiteral{
		expr:   v,
		isExpr: isExpr,
	}
	r.init(c, v.Idx0())
	return r
}

func nearestNonLexical(s *scope) *scope {
	for ; s != nil && s.lexical; s = s.outer {
	}
	return s
}

func (e *compiledThisExpr) emitGetter(putOnStack bool) {
	if putOnStack {
		e.addSrcMap()
		if e.c.scope.eval || e.c.scope.isFunction() {
			nearestNonLexical(e.c.scope).thisNeeded = true
			e.c.emit(loadStack(0))
		} else {
			e.c.emit(loadGlobalObject)
		}
	}
}

/*
func (e *compiledThisExpr) deleteExpr() compiledExpr {
	r := &compiledLiteral{
		val: valueTrue,
	}
	r.init(e.c, 0)
	return r
}
*/

func (e *compiledNewExpr) emitGetter(putOnStack bool) {
	e.callee.emitGetter(true)
	for _, expr := range e.args {
		expr.emitGetter(true)
	}
	e.addSrcMap()
	e.c.emit(_new(len(e.args)))
	if !putOnStack {
		e.c.emit(pop)
	}
}

func (c *compiler) compileNewExpression(v *ast.NewExpression) compiledExpr {
	args := make([]compiledExpr, len(v.ArgumentList))
	for i, expr := range v.ArgumentList {
		args[i] = c.compileExpression(expr)
	}
	r := &compiledNewExpr{
		callee: c.compileExpression(v.Callee),
		args:   args,
	}
	r.init(c, v.Idx0())
	return r
}

func (e *compiledSequenceExpr) emitGetter(putOnStack bool) {
	if len(e.sequence) > 0 {
		for i := 0; i < len(e.sequence)-1; i++ {
			e.sequence[i].emitGetter(false)
		}
		e.sequence[len(e.sequence)-1].emitGetter(putOnStack)
	}
}

func (c *compiler) compileSequenceExpression(v *ast.SequenceExpression) compiledExpr {
	s := make([]compiledExpr, len(v.Sequence))
	for i, expr := range v.Sequence {
		s[i] = c.compileExpression(expr)
	}
	r := &compiledSequenceExpr{
		sequence: s,
	}
	var idx file.Idx
	if len(v.Sequence) > 0 {
		idx = v.Idx0()
	}
	r.init(c, idx)
	return r
}

func (c *compiler) emitThrow(v Value) {
	if o, ok := v.(*Object); ok {
		t := o.self.getStr("name").String()
		switch t {
		case "TypeError":
			c.emit(getVar1(t))
			msg := o.self.getStr("message")
			if msg != nil {
				c.emit(loadVal(c.p.defineLiteralValue(msg)))
				c.emit(_new(1))
			} else {
				c.emit(_new(0))
			}
			c.emit(throw)
			return
		}
	}
	panic(fmt.Errorf("Unknown exception type thrown while evaliating constant expression: %s", v.String()))
}

func (c *compiler) emitConst(expr compiledExpr, putOnStack bool) {
	v, ex := c.evalConst(expr)
	if ex == nil {
		if putOnStack {
			c.emit(loadVal(c.p.defineLiteralValue(v)))
		}
	} else {
		c.emitThrow(ex.val)
	}
}

func (c *compiler) emitExpr(expr compiledExpr, putOnStack bool) {
	if expr.constant() {
		c.emitConst(expr, putOnStack)
	} else {
		expr.emitGetter(putOnStack)
	}
}

func (c *compiler) evalConst(expr compiledExpr) (Value, *Exception) {
	if expr, ok := expr.(*compiledLiteral); ok {
		return expr.val, nil
	}
	if c.evalVM == nil {
		c.evalVM = New().vm
	}
	var savedPrg *Program
	createdPrg := false
	if c.evalVM.prg == nil {
		c.evalVM.prg = &Program{}
		savedPrg = c.p
		c.p = c.evalVM.prg
		createdPrg = true
	}
	savedPc := len(c.p.code)
	expr.emitGetter(true)
	c.emit(halt)
	c.evalVM.pc = savedPc
	ex := c.evalVM.runTry()
	if createdPrg {
		c.evalVM.prg = nil
		c.evalVM.pc = 0
		c.p = savedPrg
	} else {
		c.evalVM.prg.code = c.evalVM.prg.code[:savedPc]
		c.p.code = c.evalVM.prg.code
	}
	if ex == nil {
		return c.evalVM.pop(), nil
	}
	return nil, ex
}

func (e *compiledUnaryExpr) constant() bool {
	return e.operand.constant()
}

func (e *compiledUnaryExpr) emitGetter(putOnStack bool) {
	var prepare, body func()

	toNumber := func() {
		e.c.emit(toNumber)
	}

	switch e.operator {
	case token.NOT:
		e.operand.emitGetter(true)
		e.c.emit(not)
		goto end
	case token.BITWISE_NOT:
		e.operand.emitGetter(true)
		e.c.emit(bnot)
		goto end
	case token.TYPEOF:
		if o, ok := e.operand.(compiledExprOrRef); ok {
			o.emitGetterOrRef()
		} else {
			e.operand.emitGetter(true)
		}
		e.c.emit(typeof)
		goto end
	case token.DELETE:
		e.operand.deleteExpr().emitGetter(putOnStack)
		return
	case token.MINUS:
		e.c.emitExpr(e.operand, true)
		e.c.emit(neg)
		goto end
	case token.PLUS:
		e.c.emitExpr(e.operand, true)
		e.c.emit(plus)
		goto end
	case token.INCREMENT:
		prepare = toNumber
		body = func() {
			e.c.emit(inc)
		}
	case token.DECREMENT:
		prepare = toNumber
		body = func() {
			e.c.emit(dec)
		}
	case token.VOID:
		e.c.emitExpr(e.operand, false)
		if putOnStack {
			e.c.emit(loadUndef)
		}
		return
	default:
		panic(fmt.Errorf("Unknown unary operator: %s", e.operator.String()))
	}

	e.operand.emitUnary(prepare, body, e.postfix, putOnStack)
	return

end:
	if !putOnStack {
		e.c.emit(pop)
	}
}

func (c *compiler) compileUnaryExpression(v *ast.UnaryExpression) compiledExpr {
	r := &compiledUnaryExpr{
		operand:  c.compileExpression(v.Operand),
		operator: v.Operator,
		postfix:  v.Postfix,
	}
	r.init(c, v.Idx0())
	return r
}

func (e *compiledConditionalExpr) emitGetter(putOnStack bool) {
	e.test.emitGetter(true)
	j := len(e.c.p.code)
	e.c.emit(nil)
	e.consequent.emitGetter(putOnStack)
	j1 := len(e.c.p.code)
	e.c.emit(nil)
	e.c.p.code[j] = jne(len(e.c.p.code) - j)
	e.alternate.emitGetter(putOnStack)
	e.c.p.code[j1] = jump(len(e.c.p.code) - j1)
}

func (c *compiler) compileConditionalExpression(v *ast.ConditionalExpression) compiledExpr {
	r := &compiledConditionalExpr{
		test:       c.compileExpression(v.Test),
		consequent: c.compileExpression(v.Consequent),
		alternate:  c.compileExpression(v.Alternate),
	}
	r.init(c, v.Idx0())
	return r
}

func (e *compiledLogicalOr) constant() bool {
	if e.left.constant() {
		if v, ex := e.c.evalConst(e.left); ex == nil {
			if v.ToBoolean() {
				return true
			}
			return e.right.constant()
		} else {
			return true
		}
	}

	return false
}

func (e *compiledLogicalOr) emitGetter(putOnStack bool) {
	if e.left.constant() {
		if v, ex := e.c.evalConst(e.left); ex == nil {
			if !v.ToBoolean() {
				e.c.emitExpr(e.right, putOnStack)
			} else {
				if putOnStack {
					e.c.emit(loadVal(e.c.p.defineLiteralValue(v)))
				}
			}
		} else {
			e.c.emitThrow(ex.val)
		}
		return
	}
	e.c.emitExpr(e.left, true)
	e.c.markBlockStart()
	j := len(e.c.p.code)
	e.addSrcMap()
	e.c.emit(nil)
	e.c.emit(pop)
	e.c.emitExpr(e.right, true)
	e.c.p.code[j] = jeq1(len(e.c.p.code) - j)
	if !putOnStack {
		e.c.emit(pop)
	}
}

func (e *compiledLogicalAnd) constant() bool {
	if e.left.constant() {
		if v, ex := e.c.evalConst(e.left); ex == nil {
			if !v.ToBoolean() {
				return true
			} else {
				return e.right.constant()
			}
		} else {
			return true
		}
	}

	return false
}

func (e *compiledLogicalAnd) emitGetter(putOnStack bool) {
	var j int
	if e.left.constant() {
		if v, ex := e.c.evalConst(e.left); ex == nil {
			if !v.ToBoolean() {
				e.c.emit(loadVal(e.c.p.defineLiteralValue(v)))
			} else {
				e.c.emitExpr(e.right, putOnStack)
			}
		} else {
			e.c.emitThrow(ex.val)
		}
		return
	}
	e.left.emitGetter(true)
	e.c.markBlockStart()
	j = len(e.c.p.code)
	e.addSrcMap()
	e.c.emit(nil)
	e.c.emit(pop)
	e.c.emitExpr(e.right, true)
	e.c.p.code[j] = jneq1(len(e.c.p.code) - j)
	if !putOnStack {
		e.c.emit(pop)
	}
}

func (e *compiledBinaryExpr) constant() bool {
	return e.left.constant() && e.right.constant()
}

func (e *compiledBinaryExpr) emitGetter(putOnStack bool) {
	e.c.emitExpr(e.left, true)
	e.c.emitExpr(e.right, true)
	e.addSrcMap()

	switch e.operator {
	case token.LESS:
		e.c.emit(op_lt)
	case token.GREATER:
		e.c.emit(op_gt)
	case token.LESS_OR_EQUAL:
		e.c.emit(op_lte)
	case token.GREATER_OR_EQUAL:
		e.c.emit(op_gte)
	case token.EQUAL:
		e.c.emit(op_eq)
	case token.NOT_EQUAL:
		e.c.emit(op_neq)
	case token.STRICT_EQUAL:
		e.c.emit(op_strict_eq)
	case token.STRICT_NOT_EQUAL:
		e.c.emit(op_strict_neq)
	case token.PLUS:
		e.c.emit(add)
	case token.MINUS:
		e.c.emit(sub)
	case token.MULTIPLY:
		e.c.emit(mul)
	case token.SLASH:
		e.c.emit(div)
	case token.REMAINDER:
		e.c.emit(mod)
	case token.AND:
		e.c.emit(and)
	case token.OR:
		e.c.emit(or)
	case token.EXCLUSIVE_OR:
		e.c.emit(xor)
	case token.INSTANCEOF:
		e.c.emit(op_instanceof)
	case token.IN:
		e.c.emit(op_in)
	case token.SHIFT_LEFT:
		e.c.emit(sal)
	case token.SHIFT_RIGHT:
		e.c.emit(sar)
	case token.UNSIGNED_SHIFT_RIGHT:
		e.c.emit(shr)
	default:
		panic(fmt.Errorf("Unknown operator: %s", e.operator.String()))
	}

	if !putOnStack {
		e.c.emit(pop)
	}
}

func (c *compiler) compileBinaryExpression(v *ast.BinaryExpression) compiledExpr {

	switch v.Operator {
	case token.LOGICAL_OR:
		return c.compileLogicalOr(v.Left, v.Right, v.Idx0())
	case token.LOGICAL_AND:
		return c.compileLogicalAnd(v.Left, v.Right, v.Idx0())
	}

	r := &compiledBinaryExpr{
		left:     c.compileExpression(v.Left),
		right:    c.compileExpression(v.Right),
		operator: v.Operator,
	}
	r.init(c, v.Idx0())
	return r
}

func (c *compiler) compileLogicalOr(left, right ast.Expression, idx file.Idx) compiledExpr {
	r := &compiledLogicalOr{
		left:  c.compileExpression(left),
		right: c.compileExpression(right),
	}
	r.init(c, idx)
	return r
}

func (c *compiler) compileLogicalAnd(left, right ast.Expression, idx file.Idx) compiledExpr {
	r := &compiledLogicalAnd{
		left:  c.compileExpression(left),
		right: c.compileExpression(right),
	}
	r.init(c, idx)
	return r
}

func (e *compiledVariableExpr) emitGetter(putOnStack bool) {
	if e.initializer != nil {
		idExpr := &compiledIdentifierExpr{
			name: e.name,
		}
		idExpr.init(e.c, file.Idx(0))
		idExpr.emitSetter(e.initializer)
		if !putOnStack {
			e.c.emit(pop)
		}
	} else {
		if putOnStack {
			e.c.emit(loadUndef)
		}
	}
}

func (c *compiler) compileVariableExpression(v *ast.VariableExpression) compiledExpr {
	r := &compiledVariableExpr{
		name:        v.Name,
		initializer: c.compileExpression(v.Initializer),
	}
	r.init(c, v.Idx0())
	return r
}

func (e *compiledObjectLiteral) emitGetter(putOnStack bool) {
	e.addSrcMap()
	e.c.emit(newObject)
	for _, prop := range e.expr.Value {
		e.c.compileExpression(prop.Value).emitGetter(true)
		switch prop.Kind {
		case "value":
			if prop.Key == "__proto__" {
				e.c.emit(setProto)
			} else {
				e.c.emit(setProp1(prop.Key))
			}
		case "get":
			e.c.emit(setPropGetter(prop.Key))
		case "set":
			e.c.emit(setPropSetter(prop.Key))
		default:
			panic(fmt.Errorf("Unknown property kind: %s", prop.Kind))
		}
	}
	if !putOnStack {
		e.c.emit(pop)
	}
}

func (c *compiler) compileObjectLiteral(v *ast.ObjectLiteral) compiledExpr {
	r := &compiledObjectLiteral{
		expr: v,
	}
	r.init(c, v.Idx0())
	return r
}

func (e *compiledArrayLiteral) emitGetter(putOnStack bool) {
	e.addSrcMap()
	for _, v := range e.expr.Value {
		if v != nil {
			e.c.compileExpression(v).emitGetter(true)
		} else {
			e.c.emit(loadNil)
		}
	}
	e.c.emit(newArray(len(e.expr.Value)))
	if !putOnStack {
		e.c.emit(pop)
	}
}

func (c *compiler) compileArrayLiteral(v *ast.ArrayLiteral) compiledExpr {
	r := &compiledArrayLiteral{
		expr: v,
	}
	r.init(c, v.Idx0())
	return r
}

func (e *compiledRegexpLiteral) emitGetter(putOnStack bool) {
	if putOnStack {
		pattern, global, ignoreCase, multiline, err := compileRegexp(e.expr.Pattern, e.expr.Flags)
		if err != nil {
			e.c.throwSyntaxError(e.offset, err.Error())
		}

		e.c.emit(&newRegexp{pattern: pattern,
			src:        newStringValue(e.expr.Pattern),
			global:     global,
			ignoreCase: ignoreCase,
			multiline:  multiline,
		})
	}
}

func (c *compiler) compileRegexpLiteral(v *ast.RegExpLiteral) compiledExpr {
	r := &compiledRegexpLiteral{
		expr: v,
	}
	r.init(c, v.Idx0())
	return r
}

func (e *compiledCallExpr) emitGetter(putOnStack bool) {
	var calleeName string
	switch callee := e.callee.(type) {
	case *compiledDotExpr:
		callee.left.emitGetter(true)
		e.c.emit(dup)
		e.c.emit(getPropCallee(callee.name))
	case *compiledBracketExpr:
		callee.left.emitGetter(true)
		e.c.emit(dup)
		callee.member.emitGetter(true)
		e.c.emit(getElemCallee)
	case *compiledIdentifierExpr:
		e.c.emit(loadUndef)
		calleeName = callee.name
		callee.emitGetterOrRef()
	default:
		e.c.emit(loadUndef)
		callee.emitGetter(true)
	}

	for _, expr := range e.args {
		expr.emitGetter(true)
	}

	e.addSrcMap()
	if calleeName == "eval" {
		e.c.scope.dynamic = true
		e.c.scope.thisNeeded = true
		if e.c.scope.lexical {
			e.c.scope.outer.dynamic = true
		}
		e.c.scope.accessed = true
		if e.c.scope.strict {
			e.c.emit(callEvalStrict(len(e.args)))
		} else {
			e.c.emit(callEval(len(e.args)))
		}
	} else {
		e.c.emit(call(len(e.args)))
	}

	if !putOnStack {
		e.c.emit(pop)
	}
}

func (e *compiledCallExpr) deleteExpr() compiledExpr {
	r := &defaultDeleteExpr{
		expr: e,
	}
	r.init(e.c, file.Idx(e.offset+1))
	return r
}

func (c *compiler) compileCallExpression(v *ast.CallExpression) compiledExpr {

	args := make([]compiledExpr, len(v.ArgumentList))
	for i, argExpr := range v.ArgumentList {
		args[i] = c.compileExpression(argExpr)
	}

	r := &compiledCallExpr{
		args:   args,
		callee: c.compileExpression(v.Callee),
	}
	r.init(c, v.LeftParenthesis)
	return r
}

func (c *compiler) compileIdentifierExpression(v *ast.Identifier) compiledExpr {
	if c.scope.strict {
		c.checkIdentifierName(v.Name, int(v.Idx)-1)
	}

	r := &compiledIdentifierExpr{
		name: v.Name,
	}
	r.offset = int(v.Idx) - 1
	r.init(c, v.Idx0())
	return r
}

func (c *compiler) compileNumberLiteral(v *ast.NumberLiteral) compiledExpr {
	if c.scope.strict && octalRegexp.MatchString(v.Literal) {
		c.throwSyntaxError(int(v.Idx)-1, "Octal literals are not allowed in strict mode")
		panic("Unreachable")
	}
	var val Value
	switch num := v.Value.(type) {
	case int64:
		val = intToValue(num)
	case float64:
		val = floatToValue(num)
	default:
		panic(fmt.Errorf("Unsupported number literal type: %T", v.Value))
	}
	r := &compiledLiteral{
		val: val,
	}
	r.init(c, v.Idx0())
	return r
}

func (c *compiler) compileStringLiteral(v *ast.StringLiteral) compiledExpr {
	r := &compiledLiteral{
		val: newStringValue(v.Value),
	}
	r.init(c, v.Idx0())
	return r
}

func (c *compiler) compileBooleanLiteral(v *ast.BooleanLiteral) compiledExpr {
	var val Value
	if v.Value {
		val = valueTrue
	} else {
		val = valueFalse
	}

	r := &compiledLiteral{
		val: val,
	}
	r.init(c, v.Idx0())
	return r
}

func (c *compiler) compileAssignExpression(v *ast.AssignExpression) compiledExpr {
	// log.Printf("compileAssignExpression(): %+v", v)

	r := &compiledAssignExpr{
		left:     c.compileExpression(v.Left),
		right:    c.compileExpression(v.Right),
		operator: v.Operator,
	}
	r.init(c, v.Idx0())
	return r
}

func (e *compiledEnumGetExpr) emitGetter(putOnStack bool) {
	e.c.emit(enumGet)
	if !putOnStack {
		e.c.emit(pop)
	}
}
