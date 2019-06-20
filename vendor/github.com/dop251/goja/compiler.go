package goja

import (
	"fmt"
	"github.com/dop251/goja/ast"
	"github.com/dop251/goja/file"
	"sort"
	"strconv"
)

const (
	blockLoop = iota
	blockTry
	blockBranch
	blockSwitch
	blockWith
)

type CompilerError struct {
	Message string
	File    *SrcFile
	Offset  int
}

type CompilerSyntaxError struct {
	CompilerError
}

type CompilerReferenceError struct {
	CompilerError
}

type srcMapItem struct {
	pc     int
	srcPos int
}

type Program struct {
	code   []instruction
	values []Value

	funcName string
	src      *SrcFile
	srcMap   []srcMapItem
}

type compiler struct {
	p          *Program
	scope      *scope
	block      *block
	blockStart int

	enumGetExpr compiledEnumGetExpr

	evalVM *vm
}

type scope struct {
	names      map[string]uint32
	outer      *scope
	strict     bool
	eval       bool
	lexical    bool
	dynamic    bool
	accessed   bool
	argsNeeded bool
	thisNeeded bool

	namesMap    map[string]string
	lastFreeTmp int
}

type block struct {
	typ        int
	label      string
	needResult bool
	cont       int
	breaks     []int
	conts      []int
	outer      *block
}

func (c *compiler) leaveBlock() {
	lbl := len(c.p.code)
	for _, item := range c.block.breaks {
		c.p.code[item] = jump(lbl - item)
	}
	if c.block.typ == blockLoop {
		for _, item := range c.block.conts {
			c.p.code[item] = jump(c.block.cont - item)
		}
	}
	c.block = c.block.outer
}

func (e *CompilerSyntaxError) Error() string {
	if e.File != nil {
		return fmt.Sprintf("SyntaxError: %s at %s", e.Message, e.File.Position(e.Offset))
	}
	return fmt.Sprintf("SyntaxError: %s", e.Message)
}

func (e *CompilerReferenceError) Error() string {
	return fmt.Sprintf("ReferenceError: %s", e.Message)
}

func (c *compiler) newScope() {
	strict := false
	if c.scope != nil {
		strict = c.scope.strict
	}
	c.scope = &scope{
		outer:    c.scope,
		names:    make(map[string]uint32),
		strict:   strict,
		namesMap: make(map[string]string),
	}
}

func (c *compiler) popScope() {
	c.scope = c.scope.outer
}

func newCompiler() *compiler {
	c := &compiler{
		p: &Program{},
	}

	c.enumGetExpr.init(c, file.Idx(0))

	c.newScope()
	c.scope.dynamic = true
	return c
}

func (p *Program) defineLiteralValue(val Value) uint32 {
	for idx, v := range p.values {
		if v.SameAs(val) {
			return uint32(idx)
		}
	}
	idx := uint32(len(p.values))
	p.values = append(p.values, val)
	return idx
}

func (p *Program) dumpCode(logger func(format string, args ...interface{})) {
	p._dumpCode("", logger)
}

func (p *Program) _dumpCode(indent string, logger func(format string, args ...interface{})) {
	logger("values: %+v", p.values)
	for pc, ins := range p.code {
		logger("%s %d: %T(%v)", indent, pc, ins, ins)
		if f, ok := ins.(*newFunc); ok {
			f.prg._dumpCode(indent+">", logger)
		}
	}
}

func (p *Program) sourceOffset(pc int) int {
	i := sort.Search(len(p.srcMap), func(idx int) bool {
		return p.srcMap[idx].pc > pc
	}) - 1
	if i >= 0 {
		return p.srcMap[i].srcPos
	}

	return 0
}

func (s *scope) isFunction() bool {
	if !s.lexical {
		return s.outer != nil
	}
	return s.outer.isFunction()
}

func (s *scope) lookupName(name string) (idx uint32, found, noDynamics bool) {
	var level uint32 = 0
	noDynamics = true
	for curScope := s; curScope != nil; curScope = curScope.outer {
		if curScope != s {
			curScope.accessed = true
		}
		if curScope.dynamic {
			noDynamics = false
		} else {
			var mapped string
			if m, exists := curScope.namesMap[name]; exists {
				mapped = m
			} else {
				mapped = name
			}
			if i, exists := curScope.names[mapped]; exists {
				idx = i | (level << 24)
				found = true
				return
			}
		}
		if name == "arguments" && !s.lexical && s.isFunction() {
			s.argsNeeded = true
			s.accessed = true
			idx, _ = s.bindName(name)
			found = true
			return
		}
		level++
	}
	return
}

func (s *scope) bindName(name string) (uint32, bool) {
	if s.lexical {
		return s.outer.bindName(name)
	}

	if idx, exists := s.names[name]; exists {
		return idx, false
	}
	idx := uint32(len(s.names))
	s.names[name] = idx
	return idx, true
}

func (s *scope) bindNameShadow(name string) (uint32, bool) {
	if s.lexical {
		return s.outer.bindName(name)
	}

	unique := true

	if idx, exists := s.names[name]; exists {
		unique = false
		// shadow the var
		delete(s.names, name)
		n := strconv.Itoa(int(idx))
		s.names[n] = idx
	}
	idx := uint32(len(s.names))
	s.names[name] = idx
	return idx, unique
}

func (c *compiler) markBlockStart() {
	c.blockStart = len(c.p.code)
}

func (c *compiler) compile(in *ast.Program) {
	c.p.src = NewSrcFile(in.File.Name(), in.File.Source(), in.SourceMap)

	if len(in.Body) > 0 {
		if !c.scope.strict {
			c.scope.strict = c.isStrict(in.Body)
		}
	}

	c.compileDeclList(in.DeclarationList, false)
	c.compileFunctions(in.DeclarationList)

	c.markBlockStart()
	c.compileStatements(in.Body, true)

	c.p.code = append(c.p.code, halt)
	code := c.p.code
	c.p.code = make([]instruction, 0, len(code)+len(c.scope.names)+2)
	if c.scope.eval {
		if !c.scope.strict {
			c.emit(jne(2), newStash)
		} else {
			c.emit(pop, newStash)
		}
	}
	l := len(c.p.code)
	c.p.code = c.p.code[:l+len(c.scope.names)]
	for name, nameIdx := range c.scope.names {
		c.p.code[l+int(nameIdx)] = bindName(name)
	}

	c.p.code = append(c.p.code, code...)
	for i, _ := range c.p.srcMap {
		c.p.srcMap[i].pc += len(c.scope.names)
	}

}

func (c *compiler) compileDeclList(v []ast.Declaration, inFunc bool) {
	for _, value := range v {
		switch value := value.(type) {
		case *ast.FunctionDeclaration:
			c.compileFunctionDecl(value)
		case *ast.VariableDeclaration:
			c.compileVarDecl(value, inFunc)
		default:
			panic(fmt.Errorf("Unsupported declaration: %T", value))
		}
	}
}

func (c *compiler) compileFunctions(v []ast.Declaration) {
	for _, value := range v {
		if value, ok := value.(*ast.FunctionDeclaration); ok {
			c.compileFunction(value)
		}
	}
}

func (c *compiler) compileVarDecl(v *ast.VariableDeclaration, inFunc bool) {
	for _, item := range v.List {
		if c.scope.strict {
			c.checkIdentifierLName(item.Name, int(item.Idx)-1)
			c.checkIdentifierName(item.Name, int(item.Idx)-1)
		}
		if !inFunc || item.Name != "arguments" {
			idx, ok := c.scope.bindName(item.Name)
			_ = idx
			//log.Printf("Define var: %s: %x", item.Name, idx)
			if !ok {
				// TODO: error
			}
		}
	}
}

func (c *compiler) addDecls() []instruction {
	code := make([]instruction, len(c.scope.names))
	for name, nameIdx := range c.scope.names {
		code[nameIdx] = bindName(name)
	}
	return code
}

func (c *compiler) convertInstrToStashless(instr uint32, args int) (newIdx int, convert bool) {
	level := instr >> 24
	idx := instr & 0x00FFFFFF
	if level > 0 {
		level--
		newIdx = int((level << 24) | idx)
	} else {
		iidx := int(idx)
		if iidx < args {
			newIdx = -iidx - 1
		} else {
			newIdx = iidx - args + 1
		}
		convert = true
	}
	return
}

func (c *compiler) convertFunctionToStashless(code []instruction, args int) {
	code[0] = enterFuncStashless{stackSize: uint32(len(c.scope.names) - args), args: uint32(args)}
	for pc := 1; pc < len(code); pc++ {
		instr := code[pc]
		if instr == ret {
			code[pc] = retStashless
		}
		switch instr := instr.(type) {
		case getLocal:
			if newIdx, convert := c.convertInstrToStashless(uint32(instr), args); convert {
				code[pc] = loadStack(newIdx)
			} else {
				code[pc] = getLocal(newIdx)
			}
		case setLocal:
			if newIdx, convert := c.convertInstrToStashless(uint32(instr), args); convert {
				code[pc] = storeStack(newIdx)
			} else {
				code[pc] = setLocal(newIdx)
			}
		case setLocalP:
			if newIdx, convert := c.convertInstrToStashless(uint32(instr), args); convert {
				code[pc] = storeStackP(newIdx)
			} else {
				code[pc] = setLocalP(newIdx)
			}
		case getVar:
			level := instr.idx >> 24
			idx := instr.idx & 0x00FFFFFF
			level--
			instr.idx = level<<24 | idx
			code[pc] = instr
		case setVar:
			level := instr.idx >> 24
			idx := instr.idx & 0x00FFFFFF
			level--
			instr.idx = level<<24 | idx
			code[pc] = instr
		}
	}
}

func (c *compiler) compileFunctionDecl(v *ast.FunctionDeclaration) {
	idx, ok := c.scope.bindName(v.Function.Name.Name)
	if !ok {
		// TODO: error
	}
	_ = idx
	// log.Printf("Define function: %s: %x", v.Function.Name.Name, idx)
}

func (c *compiler) compileFunction(v *ast.FunctionDeclaration) {
	e := &compiledIdentifierExpr{
		name: v.Function.Name.Name,
	}
	e.init(c, v.Function.Idx0())
	e.emitSetter(c.compileFunctionLiteral(v.Function, false))
	c.emit(pop)
}

func (c *compiler) emit(instructions ...instruction) {
	c.p.code = append(c.p.code, instructions...)
}

func (c *compiler) throwSyntaxError(offset int, format string, args ...interface{}) {
	panic(&CompilerSyntaxError{
		CompilerError: CompilerError{
			File:    c.p.src,
			Offset:  offset,
			Message: fmt.Sprintf(format, args...),
		},
	})
}

func (c *compiler) isStrict(list []ast.Statement) bool {
	for _, st := range list {
		if st, ok := st.(*ast.ExpressionStatement); ok {
			if e, ok := st.Expression.(*ast.StringLiteral); ok {
				if e.Literal == `"use strict"` || e.Literal == `'use strict'` {
					return true
				}
			} else {
				break
			}
		} else {
			break
		}
	}
	return false
}

func (c *compiler) isStrictStatement(s ast.Statement) bool {
	if s, ok := s.(*ast.BlockStatement); ok {
		return c.isStrict(s.List)
	}
	return false
}

func (c *compiler) checkIdentifierName(name string, offset int) {
	switch name {
	case "implements", "interface", "let", "package", "private", "protected", "public", "static", "yield":
		c.throwSyntaxError(offset, "Unexpected strict mode reserved word")
	}
}

func (c *compiler) checkIdentifierLName(name string, offset int) {
	switch name {
	case "eval", "arguments":
		c.throwSyntaxError(offset, "Assignment to eval or arguments is not allowed in strict mode")
	}
}
