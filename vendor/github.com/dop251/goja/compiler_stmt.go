package goja

import (
	"fmt"
	"github.com/dop251/goja/ast"
	"github.com/dop251/goja/file"
	"github.com/dop251/goja/token"
	"strconv"
)

func (c *compiler) compileStatement(v ast.Statement, needResult bool) {
	// log.Printf("compileStatement(): %T", v)

	switch v := v.(type) {
	case *ast.BlockStatement:
		c.compileBlockStatement(v, needResult)
	case *ast.ExpressionStatement:
		c.compileExpressionStatement(v, needResult)
	case *ast.VariableStatement:
		c.compileVariableStatement(v, needResult)
	case *ast.ReturnStatement:
		c.compileReturnStatement(v)
	case *ast.IfStatement:
		c.compileIfStatement(v, needResult)
	case *ast.DoWhileStatement:
		c.compileDoWhileStatement(v, needResult)
	case *ast.ForStatement:
		c.compileForStatement(v, needResult)
	case *ast.ForInStatement:
		c.compileForInStatement(v, needResult)
	case *ast.WhileStatement:
		c.compileWhileStatement(v, needResult)
	case *ast.BranchStatement:
		c.compileBranchStatement(v, needResult)
	case *ast.TryStatement:
		c.compileTryStatement(v)
		if needResult {
			c.emit(loadUndef)
		}
	case *ast.ThrowStatement:
		c.compileThrowStatement(v)
	case *ast.SwitchStatement:
		c.compileSwitchStatement(v, needResult)
	case *ast.LabelledStatement:
		c.compileLabeledStatement(v, needResult)
	case *ast.EmptyStatement:
		c.compileEmptyStatement(needResult)
	case *ast.WithStatement:
		c.compileWithStatement(v, needResult)
	case *ast.DebuggerStatement:
	default:
		panic(fmt.Errorf("Unknown statement type: %T", v))
	}
}

func (c *compiler) compileLabeledStatement(v *ast.LabelledStatement, needResult bool) {
	label := v.Label.Name
	for b := c.block; b != nil; b = b.outer {
		if b.label == label {
			c.throwSyntaxError(int(v.Label.Idx-1), "Label '%s' has already been declared", label)
		}
	}
	switch s := v.Statement.(type) {
	case *ast.ForInStatement:
		c.compileLabeledForInStatement(s, needResult, label)
	case *ast.ForStatement:
		c.compileLabeledForStatement(s, needResult, label)
	case *ast.WhileStatement:
		c.compileLabeledWhileStatement(s, needResult, label)
	case *ast.DoWhileStatement:
		c.compileLabeledDoWhileStatement(s, needResult, label)
	default:
		c.compileGenericLabeledStatement(v.Statement, needResult, label)
	}
}

func (c *compiler) compileTryStatement(v *ast.TryStatement) {
	if c.scope.strict && v.Catch != nil {
		switch v.Catch.Parameter.Name {
		case "arguments", "eval":
			c.throwSyntaxError(int(v.Catch.Parameter.Idx)-1, "Catch variable may not be eval or arguments in strict mode")
		}
	}
	c.block = &block{
		typ:   blockTry,
		outer: c.block,
	}
	lbl := len(c.p.code)
	c.emit(nil)
	c.compileStatement(v.Body, false)
	c.emit(halt)
	lbl2 := len(c.p.code)
	c.emit(nil)
	var catchOffset int
	dynamicCatch := true
	if v.Catch != nil {
		dyn := nearestNonLexical(c.scope).dynamic
		accessed := c.scope.accessed
		c.newScope()
		c.scope.bindName(v.Catch.Parameter.Name)
		c.scope.lexical = true
		start := len(c.p.code)
		c.emit(nil)
		catchOffset = len(c.p.code) - lbl
		c.emit(enterCatch(v.Catch.Parameter.Name))
		c.compileStatement(v.Catch.Body, false)
		dyn1 := c.scope.dynamic
		accessed1 := c.scope.accessed
		c.popScope()
		if !dyn && !dyn1 && !accessed1 {
			c.scope.accessed = accessed
			dynamicCatch = false
			code := c.p.code[start+1:]
			m := make(map[uint32]uint32)
			remap := func(instr uint32) uint32 {
				level := instr >> 24
				idx := instr & 0x00FFFFFF
				if level > 0 {
					level--
					return (level << 24) | idx
				} else {
					// remap
					newIdx, exists := m[idx]
					if !exists {
						exname := " __tmp" + strconv.Itoa(c.scope.lastFreeTmp)
						c.scope.lastFreeTmp++
						newIdx, _ = c.scope.bindName(exname)
						m[idx] = newIdx
					}
					return newIdx
				}
			}
			for pc, instr := range code {
				switch instr := instr.(type) {
				case getLocal:
					code[pc] = getLocal(remap(uint32(instr)))
				case setLocal:
					code[pc] = setLocal(remap(uint32(instr)))
				case setLocalP:
					code[pc] = setLocalP(remap(uint32(instr)))
				}
			}
			if catchVarIdx, exists := m[0]; exists {
				c.p.code[start] = setLocal(catchVarIdx)
				c.p.code[start+1] = pop
				catchOffset--
			} else {
				c.p.code[start+1] = nil
				catchOffset++
			}
		} else {
			c.scope.accessed = true
		}

		/*
			if true/*sc.dynamic/ {
				dynamicCatch = true
				c.scope.accessed = true
				c.newScope()
				c.scope.bindName(v.Catch.Parameter.Name)
				c.scope.lexical = true
				c.emit(enterCatch(v.Catch.Parameter.Name))
				c.compileStatement(v.Catch.Body, false)
				c.popScope()
			} else {
				exname := " __tmp" + strconv.Itoa(c.scope.lastFreeTmp)
				c.scope.lastFreeTmp++
				catchVarIdx, _ := c.scope.bindName(exname)
				c.emit(setLocal(catchVarIdx), pop)
				saved, wasSaved := c.scope.namesMap[v.Catch.Parameter.Name]
				c.scope.namesMap[v.Catch.Parameter.Name] = exname
				c.compileStatement(v.Catch.Body, false)
				if wasSaved {
					c.scope.namesMap[v.Catch.Parameter.Name] = saved
				} else {
					delete(c.scope.namesMap, v.Catch.Parameter.Name)
				}
				c.scope.lastFreeTmp--
			}*/
		c.emit(halt)
	}
	var finallyOffset int
	if v.Finally != nil {
		lbl1 := len(c.p.code)
		c.emit(nil)
		finallyOffset = len(c.p.code) - lbl
		c.compileStatement(v.Finally, false)
		c.emit(halt, retFinally)
		c.p.code[lbl1] = jump(len(c.p.code) - lbl1)
	}
	c.p.code[lbl] = try{catchOffset: int32(catchOffset), finallyOffset: int32(finallyOffset), dynamic: dynamicCatch}
	c.p.code[lbl2] = jump(len(c.p.code) - lbl2)
	c.leaveBlock()
}

func (c *compiler) compileThrowStatement(v *ast.ThrowStatement) {
	//c.p.srcMap = append(c.p.srcMap, srcMapItem{pc: len(c.p.code), srcPos: int(v.Throw) - 1})
	c.compileExpression(v.Argument).emitGetter(true)
	c.emit(throw)
}

func (c *compiler) compileDoWhileStatement(v *ast.DoWhileStatement, needResult bool) {
	c.compileLabeledDoWhileStatement(v, needResult, "")
}

func (c *compiler) compileLabeledDoWhileStatement(v *ast.DoWhileStatement, needResult bool, label string) {
	c.block = &block{
		typ:        blockLoop,
		outer:      c.block,
		label:      label,
		needResult: needResult,
	}

	if needResult {
		c.emit(jump(2))
	}
	start := len(c.p.code)
	if needResult {
		c.emit(pop)
	}
	c.markBlockStart()
	c.compileStatement(v.Body, needResult)
	c.block.cont = len(c.p.code)
	c.emitExpr(c.compileExpression(v.Test), true)
	c.emit(jeq(start - len(c.p.code)))
	c.leaveBlock()
}

func (c *compiler) compileForStatement(v *ast.ForStatement, needResult bool) {
	c.compileLabeledForStatement(v, needResult, "")
}

func (c *compiler) compileLabeledForStatement(v *ast.ForStatement, needResult bool, label string) {
	c.block = &block{
		typ:        blockLoop,
		outer:      c.block,
		label:      label,
		needResult: needResult,
	}

	if v.Initializer != nil {
		c.compileExpression(v.Initializer).emitGetter(false)
	}
	if needResult {
		c.emit(loadUndef) // initial result
	}
	start := len(c.p.code)
	c.markBlockStart()
	var j int
	testConst := false
	if v.Test != nil {
		expr := c.compileExpression(v.Test)
		if expr.constant() {
			r, ex := c.evalConst(expr)
			if ex == nil {
				if r.ToBoolean() {
					testConst = true
				} else {
					// TODO: Properly implement dummy compilation (no garbage in block, scope, etc..)
					/*
						p := c.p
						c.p = &program{}
						c.compileStatement(v.Body, false)
						if v.Update != nil {
							c.compileExpression(v.Update).emitGetter(false)
						}
						c.p = p*/
					goto end
				}
			} else {
				expr.addSrcMap()
				c.emitThrow(ex.val)
				goto end
			}
		} else {
			expr.emitGetter(true)
			j = len(c.p.code)
			c.emit(nil)
		}
	}
	if needResult {
		c.emit(pop) // remove last result
	}
	c.markBlockStart()
	c.compileStatement(v.Body, needResult)
	c.block.cont = len(c.p.code)
	if v.Update != nil {
		c.compileExpression(v.Update).emitGetter(false)
	}
	c.emit(jump(start - len(c.p.code)))
	if v.Test != nil {
		if !testConst {
			c.p.code[j] = jne(len(c.p.code) - j)
		}
	}
end:
	c.leaveBlock()
	c.markBlockStart()
}

func (c *compiler) compileForInStatement(v *ast.ForInStatement, needResult bool) {
	c.compileLabeledForInStatement(v, needResult, "")
}

func (c *compiler) compileLabeledForInStatement(v *ast.ForInStatement, needResult bool, label string) {
	c.block = &block{
		typ:        blockLoop,
		outer:      c.block,
		label:      label,
		needResult: needResult,
	}

	c.compileExpression(v.Source).emitGetter(true)
	c.emit(enumerate)
	if needResult {
		c.emit(loadUndef)
	}
	start := len(c.p.code)
	c.markBlockStart()
	c.block.cont = start
	c.emit(nil)
	c.compileExpression(v.Into).emitSetter(&c.enumGetExpr)
	c.emit(pop)
	if needResult {
		c.emit(pop) // remove last result
	}
	c.markBlockStart()
	c.compileStatement(v.Body, needResult)
	c.emit(jump(start - len(c.p.code)))
	c.p.code[start] = enumNext(len(c.p.code) - start)
	c.leaveBlock()
	c.markBlockStart()
	c.emit(enumPop)
}

func (c *compiler) compileWhileStatement(v *ast.WhileStatement, needResult bool) {
	c.compileLabeledWhileStatement(v, needResult, "")
}

func (c *compiler) compileLabeledWhileStatement(v *ast.WhileStatement, needResult bool, label string) {
	c.block = &block{
		typ:        blockLoop,
		outer:      c.block,
		label:      label,
		needResult: needResult,
	}

	if needResult {
		c.emit(loadUndef)
	}
	start := len(c.p.code)
	c.markBlockStart()
	c.block.cont = start
	expr := c.compileExpression(v.Test)
	testTrue := false
	var j int
	if expr.constant() {
		if t, ex := c.evalConst(expr); ex == nil {
			if t.ToBoolean() {
				testTrue = true
			} else {
				p := c.p
				c.p = &Program{}
				c.compileStatement(v.Body, false)
				c.p = p
				goto end
			}
		} else {
			c.emitThrow(ex.val)
			goto end
		}
	} else {
		expr.emitGetter(true)
		j = len(c.p.code)
		c.emit(nil)
	}
	if needResult {
		c.emit(pop)
	}
	c.markBlockStart()
	c.compileStatement(v.Body, needResult)
	c.emit(jump(start - len(c.p.code)))
	if !testTrue {
		c.p.code[j] = jne(len(c.p.code) - j)
	}
end:
	c.leaveBlock()
	c.markBlockStart()
}

func (c *compiler) compileEmptyStatement(needResult bool) {
	if needResult {
		if len(c.p.code) == c.blockStart {
			// first statement in block, use undefined as result
			c.emit(loadUndef)
		}
	}
}

func (c *compiler) compileBranchStatement(v *ast.BranchStatement, needResult bool) {
	switch v.Token {
	case token.BREAK:
		c.compileBreak(v.Label, v.Idx)
	case token.CONTINUE:
		c.compileContinue(v.Label, v.Idx)
	default:
		panic(fmt.Errorf("Unknown branch statement token: %s", v.Token.String()))
	}
}

func (c *compiler) findBranchBlock(st *ast.BranchStatement) *block {
	switch st.Token {
	case token.BREAK:
		return c.findBreakBlock(st.Label)
	case token.CONTINUE:
		return c.findContinueBlock(st.Label)
	}
	return nil
}

func (c *compiler) findContinueBlock(label *ast.Identifier) (block *block) {
	if label != nil {
		for b := c.block; b != nil; b = b.outer {
			if b.typ == blockLoop && b.label == label.Name {
				block = b
				break
			}
		}
	} else {
		// find the nearest loop
		for b := c.block; b != nil; b = b.outer {
			if b.typ == blockLoop {
				block = b
				break
			}
		}
	}

	return
}

func (c *compiler) findBreakBlock(label *ast.Identifier) (block *block) {
	if label != nil {
		for b := c.block; b != nil; b = b.outer {
			if b.label == label.Name {
				block = b
				break
			}
		}
	} else {
		// find the nearest loop or switch
	L:
		for b := c.block; b != nil; b = b.outer {
			switch b.typ {
			case blockLoop, blockSwitch:
				block = b
				break L
			}
		}
	}

	return
}

func (c *compiler) compileBreak(label *ast.Identifier, idx file.Idx) {
	var block *block
	if label != nil {
		for b := c.block; b != nil; b = b.outer {
			switch b.typ {
			case blockTry:
				c.emit(halt)
			case blockWith:
				c.emit(leaveWith)
			}
			if b.label == label.Name {
				block = b
				break
			}
		}
	} else {
		// find the nearest loop or switch
	L:
		for b := c.block; b != nil; b = b.outer {
			switch b.typ {
			case blockTry:
				c.emit(halt)
			case blockWith:
				c.emit(leaveWith)
			case blockLoop, blockSwitch:
				block = b
				break L
			}
		}
	}

	if block != nil {
		if len(c.p.code) == c.blockStart && block.needResult {
			c.emit(loadUndef)
		}
		block.breaks = append(block.breaks, len(c.p.code))
		c.emit(nil)
	} else {
		c.throwSyntaxError(int(idx)-1, "Undefined label '%s'", label.Name)
	}
}

func (c *compiler) compileContinue(label *ast.Identifier, idx file.Idx) {
	var block *block
	if label != nil {
		for b := c.block; b != nil; b = b.outer {
			if b.typ == blockTry {
				c.emit(halt)
			} else if b.typ == blockLoop && b.label == label.Name {
				block = b
				break
			}
		}
	} else {
		// find the nearest loop
		for b := c.block; b != nil; b = b.outer {
			if b.typ == blockTry {
				c.emit(halt)
			} else if b.typ == blockLoop {
				block = b
				break
			}
		}
	}

	if block != nil {
		if len(c.p.code) == c.blockStart && block.needResult {
			c.emit(loadUndef)
		}
		block.conts = append(block.conts, len(c.p.code))
		c.emit(nil)
	} else {
		c.throwSyntaxError(int(idx)-1, "Undefined label '%s'", label.Name)
	}
}

func (c *compiler) compileIfStatement(v *ast.IfStatement, needResult bool) {
	test := c.compileExpression(v.Test)
	if test.constant() {
		r, ex := c.evalConst(test)
		if ex != nil {
			test.addSrcMap()
			c.emitThrow(ex.val)
			return
		}
		if r.ToBoolean() {
			c.markBlockStart()
			c.compileStatement(v.Consequent, needResult)
			if v.Alternate != nil {
				p := c.p
				c.p = &Program{}
				c.markBlockStart()
				c.compileStatement(v.Alternate, false)
				c.p = p
			}
		} else {
			// TODO: Properly implement dummy compilation (no garbage in block, scope, etc..)
			p := c.p
			c.p = &Program{}
			c.compileStatement(v.Consequent, false)
			c.p = p
			if v.Alternate != nil {
				c.compileStatement(v.Alternate, needResult)
			} else {
				if needResult {
					c.emit(loadUndef)
				}
			}
		}
		return
	}
	test.emitGetter(true)
	jmp := len(c.p.code)
	c.emit(nil)
	c.markBlockStart()
	c.compileStatement(v.Consequent, needResult)
	if v.Alternate != nil {
		jmp1 := len(c.p.code)
		c.emit(nil)
		c.p.code[jmp] = jne(len(c.p.code) - jmp)
		c.markBlockStart()
		c.compileStatement(v.Alternate, needResult)
		c.p.code[jmp1] = jump(len(c.p.code) - jmp1)
		c.markBlockStart()
	} else {
		c.p.code[jmp] = jne(len(c.p.code) - jmp)
		c.markBlockStart()
		if needResult {
			c.emit(loadUndef)
		}
	}
}

func (c *compiler) compileReturnStatement(v *ast.ReturnStatement) {
	if v.Argument != nil {
		c.compileExpression(v.Argument).emitGetter(true)
		//c.emit(checkResolve)
	} else {
		c.emit(loadUndef)
	}
	for b := c.block; b != nil; b = b.outer {
		if b.typ == blockTry {
			c.emit(halt)
		}
	}
	c.emit(ret)
}

func (c *compiler) compileVariableStatement(v *ast.VariableStatement, needResult bool) {
	for _, expr := range v.List {
		c.compileExpression(expr).emitGetter(false)
	}
	if needResult {
		c.emit(loadUndef)
	}
}

func (c *compiler) getFirstNonEmptyStatement(st ast.Statement) ast.Statement {
	switch st := st.(type) {
	case *ast.BlockStatement:
		return c.getFirstNonEmptyStatementList(st.List)
	case *ast.LabelledStatement:
		return c.getFirstNonEmptyStatement(st.Statement)
	}
	return st
}

func (c *compiler) getFirstNonEmptyStatementList(list []ast.Statement) ast.Statement {
	for _, st := range list {
		switch st := st.(type) {
		case *ast.EmptyStatement:
			continue
		case *ast.BlockStatement:
			return c.getFirstNonEmptyStatementList(st.List)
		case *ast.LabelledStatement:
			return c.getFirstNonEmptyStatement(st.Statement)
		}
		return st
	}
	return nil
}

func (c *compiler) compileStatements(list []ast.Statement, needResult bool) {
	if len(list) > 0 {
		cur := list[0]
		for idx := 0; idx < len(list); {
			var next ast.Statement
			// find next non-empty statement
			for idx++; idx < len(list); idx++ {
				if _, empty := list[idx].(*ast.EmptyStatement); !empty {
					next = list[idx]
					break
				}
			}

			if next != nil {
				bs := c.getFirstNonEmptyStatement(next)
				if bs, ok := bs.(*ast.BranchStatement); ok {
					block := c.findBranchBlock(bs)
					if block != nil {
						c.compileStatement(cur, block.needResult)
						cur = next
						continue
					}
				}
				c.compileStatement(cur, false)
				cur = next
			} else {
				c.compileStatement(cur, needResult)
			}
		}
	} else {
		if needResult {
			c.emit(loadUndef)
		}
	}
}

func (c *compiler) compileGenericLabeledStatement(v ast.Statement, needResult bool, label string) {
	c.block = &block{
		typ:        blockBranch,
		outer:      c.block,
		label:      label,
		needResult: needResult,
	}
	c.compileStatement(v, needResult)
	c.leaveBlock()
}

func (c *compiler) compileBlockStatement(v *ast.BlockStatement, needResult bool) {
	c.compileStatements(v.List, needResult)
}

func (c *compiler) compileExpressionStatement(v *ast.ExpressionStatement, needResult bool) {
	expr := c.compileExpression(v.Expression)
	if expr.constant() {
		c.emitConst(expr, needResult)
	} else {
		expr.emitGetter(needResult)
	}
}

func (c *compiler) compileWithStatement(v *ast.WithStatement, needResult bool) {
	if c.scope.strict {
		c.throwSyntaxError(int(v.With)-1, "Strict mode code may not include a with statement")
		return
	}
	c.compileExpression(v.Object).emitGetter(true)
	c.emit(enterWith)
	c.block = &block{
		outer:      c.block,
		typ:        blockWith,
		needResult: needResult,
	}
	c.newScope()
	c.scope.dynamic = true
	c.scope.lexical = true
	c.compileStatement(v.Body, needResult)
	c.emit(leaveWith)
	c.leaveBlock()
	c.popScope()
}

func (c *compiler) compileSwitchStatement(v *ast.SwitchStatement, needResult bool) {
	c.block = &block{
		typ:        blockSwitch,
		outer:      c.block,
		needResult: needResult,
	}

	c.compileExpression(v.Discriminant).emitGetter(true)

	jumps := make([]int, len(v.Body))

	for i, s := range v.Body {
		if s.Test != nil {
			c.emit(dup)
			c.compileExpression(s.Test).emitGetter(true)
			c.emit(op_strict_eq)
			c.emit(jne(3), pop)
			jumps[i] = len(c.p.code)
			c.emit(nil)
		}
	}

	c.emit(pop)
	jumpNoMatch := -1
	if v.Default != -1 {
		if v.Default != 0 {
			jumps[v.Default] = len(c.p.code)
			c.emit(nil)
		}
	} else {
		jumpNoMatch = len(c.p.code)
		c.emit(nil)
	}

	for i, s := range v.Body {
		if s.Test != nil || i != 0 {
			c.p.code[jumps[i]] = jump(len(c.p.code) - jumps[i])
			c.markBlockStart()
		}
		nr := false
		c.markBlockStart()
		if needResult {
			if i < len(v.Body)-1 {
				st := c.getFirstNonEmptyStatementList(v.Body[i+1].Consequent)
				if st, ok := st.(*ast.BranchStatement); ok && st.Token == token.BREAK {
					if c.findBreakBlock(st.Label) != nil {
						stmts := append(s.Consequent, st)
						c.compileStatements(stmts, false)
						continue
					}
				}
			} else {
				nr = true
			}
		}
		c.compileStatements(s.Consequent, nr)
	}
	if jumpNoMatch != -1 {
		if needResult {
			c.emit(jump(2))
		}
		c.p.code[jumpNoMatch] = jump(len(c.p.code) - jumpNoMatch)
		if needResult {
			c.emit(loadUndef)
		}
	}
	c.leaveBlock()
	c.markBlockStart()
}
