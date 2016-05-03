package parser

import (
	"github.com/robertkrimen/otto/ast"
	"github.com/robertkrimen/otto/token"
)

func (self *_parser) parseBlockStatement() *ast.BlockStatement {
	node := &ast.BlockStatement{}

	// Find comments before the leading brace
	if self.mode&StoreComments != 0 {
		self.commentMap.AddComments(node, self.findComments(false), ast.LEADING)
	}

	node.LeftBrace = self.expect(token.LEFT_BRACE)
	node.List = self.parseStatementList()

	self.consumeComments(node, ast.FINAL)

	node.RightBrace = self.expect(token.RIGHT_BRACE)

	// Find comments after the trailing brace
	if self.mode&StoreComments != 0 {
		self.commentMap.AddComments(node, self.findComments(false), ast.TRAILING)
	}

	return node
}

func (self *_parser) parseEmptyStatement() ast.Statement {
	idx := self.expect(token.SEMICOLON)
	return &ast.EmptyStatement{Semicolon: idx}
}

func (self *_parser) parseStatementList() (list []ast.Statement) {
	for self.token != token.RIGHT_BRACE && self.token != token.EOF {
		if self.token == token.COMMENT {
			self.parseCommentElement()
			continue
		}
		statement := self.parseStatement()
		list = append(list, statement)

		self.addCommentStatements(statement, ast.LEADING)
	}

	return
}

func (self *_parser) parseStatement() ast.Statement {

	if self.token == token.EOF {
		self.errorUnexpectedToken(self.token)
		return &ast.BadStatement{From: self.idx, To: self.idx + 1}
	}

	switch self.token {
	case token.SEMICOLON:
		return self.parseEmptyStatement()
	case token.LEFT_BRACE:
		return self.parseBlockStatement()
	case token.IF:
		return self.parseIfStatement()
	case token.DO:
		return self.parseDoWhileStatement()
	case token.WHILE:
		return self.parseWhileStatement()
	case token.FOR:
		return self.parseForOrForInStatement()
	case token.BREAK:
		return self.parseBreakStatement()
	case token.CONTINUE:
		return self.parseContinueStatement()
	case token.DEBUGGER:
		return self.parseDebuggerStatement()
	case token.WITH:
		return self.parseWithStatement()
	case token.VAR:
		return self.parseVariableStatement()
	case token.FUNCTION:
		self.parseFunction(true)
		// FIXME
		return &ast.EmptyStatement{}
	case token.SWITCH:
		return self.parseSwitchStatement()
	case token.RETURN:
		return self.parseReturnStatement()
	case token.THROW:
		return self.parseThrowStatement()
	case token.TRY:
		return self.parseTryStatement()
	}

	expression := self.parseExpression()

	if identifier, isIdentifier := expression.(*ast.Identifier); isIdentifier && self.token == token.COLON {
		// LabelledStatement
		colon := self.idx
		self.next() // :

		comments := self.findComments(false)

		label := identifier.Name
		for _, value := range self.scope.labels {
			if label == value {
				self.error(identifier.Idx0(), "Label '%s' already exists", label)
			}
		}
		self.scope.labels = append(self.scope.labels, label) // Push the label
		statement := self.parseStatement()
		self.scope.labels = self.scope.labels[:len(self.scope.labels)-1] // Pop the label
		exp := &ast.LabelledStatement{
			Label:     identifier,
			Colon:     colon,
			Statement: statement,
		}

		if self.mode&StoreComments != 0 {
			self.commentMap.AddComments(exp, comments, ast.TRAILING)
		}

		return exp
	}

	self.optionalSemicolon()

	return &ast.ExpressionStatement{
		Expression: expression,
	}
}

func (self *_parser) parseTryStatement() ast.Statement {

	node := &ast.TryStatement{
		Try:  self.expect(token.TRY),
		Body: self.parseBlockStatement(),
	}

	if self.mode&StoreComments != 0 {
		self.commentMap.AddComments(node.Body, self.findComments(true), ast.TRAILING)
	}

	if self.token == token.CATCH {
		catch := self.idx
		self.next()
		self.expect(token.LEFT_PARENTHESIS)
		comments := self.findComments(true)
		if self.token != token.IDENTIFIER {
			self.expect(token.IDENTIFIER)
			self.nextStatement()
			return &ast.BadStatement{From: catch, To: self.idx}
		} else {
			identifier := self.parseIdentifier()

			if self.mode&StoreComments != 0 {
				self.commentMap.AddComments(identifier, comments, ast.LEADING)
			}

			self.expect(token.RIGHT_PARENTHESIS)
			node.Catch = &ast.CatchStatement{
				Catch:     catch,
				Parameter: identifier,
				Body:      self.parseBlockStatement(),
			}
		}

		if self.mode&StoreComments != 0 {
			self.commentMap.AddComments(node.Catch, self.findComments(true), ast.TRAILING)
		}
	}

	if self.token == token.FINALLY {
		self.next()

		comments := self.findComments(true)

		node.Finally = self.parseBlockStatement()

		if self.mode&StoreComments != 0 {
			self.commentMap.AddComments(node.Finally, comments, ast.LEADING)
		}
	}

	if node.Catch == nil && node.Finally == nil {
		self.error(node.Try, "Missing catch or finally after try")
		return &ast.BadStatement{From: node.Try, To: node.Body.Idx1()}
	}

	return node
}

func (self *_parser) parseFunctionParameterList() *ast.ParameterList {
	opening := self.expect(token.LEFT_PARENTHESIS)
	var list []*ast.Identifier
	for self.token != token.RIGHT_PARENTHESIS && self.token != token.EOF {
		comments := self.findComments(true)
		if self.token != token.IDENTIFIER {
			self.expect(token.IDENTIFIER)
		} else {
			identifier := self.parseIdentifier()
			if self.mode&StoreComments != 0 {
				self.commentMap.AddComments(identifier, comments, ast.LEADING)
			}
			list = append(list, identifier)
		}
		if self.token != token.RIGHT_PARENTHESIS {
			self.expect(token.COMMA)
		}
	}
	closing := self.expect(token.RIGHT_PARENTHESIS)

	return &ast.ParameterList{
		Opening: opening,
		List:    list,
		Closing: closing,
	}
}

func (self *_parser) parseParameterList() (list []string) {
	for self.token != token.EOF {
		if self.token != token.IDENTIFIER {
			self.expect(token.IDENTIFIER)
		}
		list = append(list, self.literal)
		self.next()
		if self.token != token.EOF {
			self.expect(token.COMMA)
		}
	}
	return
}

func (self *_parser) parseFunction(declaration bool) *ast.FunctionLiteral {

	node := &ast.FunctionLiteral{
		Function: self.expect(token.FUNCTION),
	}

	var name *ast.Identifier
	if self.token == token.IDENTIFIER {
		name = self.parseIdentifier()
		if declaration {
			self.scope.declare(&ast.FunctionDeclaration{
				Function: node,
			})
		}
	} else if declaration {
		// Use expect error handling
		self.expect(token.IDENTIFIER)
	}
	node.Name = name
	node.ParameterList = self.parseFunctionParameterList()
	self.parseFunctionBlock(node)
	node.Source = self.slice(node.Idx0(), node.Idx1())

	return node
}

func (self *_parser) parseFunctionBlock(node *ast.FunctionLiteral) {
	{
		self.openScope()
		inFunction := self.scope.inFunction
		self.scope.inFunction = true
		defer func() {
			self.scope.inFunction = inFunction
			self.closeScope()
		}()
		node.Body = self.parseBlockStatement()
		node.DeclarationList = self.scope.declarationList
	}
}

func (self *_parser) parseDebuggerStatement() ast.Statement {
	idx := self.expect(token.DEBUGGER)

	comments := self.findComments(true)

	node := &ast.DebuggerStatement{
		Debugger: idx,
	}

	if self.mode&StoreComments != 0 {
		self.commentMap.AddComments(node, comments, ast.TRAILING)
	}

	self.semicolon()

	if !self.skippedLineBreak {
		if self.mode&StoreComments != 0 {
			self.commentMap.AddComments(node, self.findComments(false), ast.TRAILING)
		}
	}

	return node
}

func (self *_parser) parseReturnStatement() ast.Statement {
	idx := self.expect(token.RETURN)

	if !self.scope.inFunction {
		self.error(idx, "Illegal return statement")
		self.nextStatement()
		return &ast.BadStatement{From: idx, To: self.idx}
	}

	node := &ast.ReturnStatement{
		Return: idx,
	}

	if !self.implicitSemicolon && self.token != token.SEMICOLON && self.token != token.RIGHT_BRACE && self.token != token.EOF {
		node.Argument = self.parseExpression()
	}

	self.semicolon()

	return node
}

func (self *_parser) parseThrowStatement() ast.Statement {
	idx := self.expect(token.THROW)

	if self.implicitSemicolon {
		if self.chr == -1 { // Hackish
			self.error(idx, "Unexpected end of input")
		} else {
			self.error(idx, "Illegal newline after throw")
		}
		self.nextStatement()
		return &ast.BadStatement{From: idx, To: self.idx}
	}

	node := &ast.ThrowStatement{
		Argument: self.parseExpression(),
	}

	self.semicolon()

	return node
}

func (self *_parser) parseSwitchStatement() ast.Statement {
	self.expect(token.SWITCH)
	self.expect(token.LEFT_PARENTHESIS)
	node := &ast.SwitchStatement{
		Discriminant: self.parseExpression(),
		Default:      -1,
	}
	self.expect(token.RIGHT_PARENTHESIS)

	self.expect(token.LEFT_BRACE)

	inSwitch := self.scope.inSwitch
	self.scope.inSwitch = true
	defer func() {
		self.scope.inSwitch = inSwitch
	}()

	for index := 0; self.token != token.EOF; index++ {
		if self.token == token.RIGHT_BRACE {
			self.next()
			break
		}

		clause := self.parseCaseStatement()
		if clause.Test == nil {
			if node.Default != -1 {
				self.error(clause.Case, "Already saw a default in switch")
			}
			node.Default = index
		}
		node.Body = append(node.Body, clause)
	}

	return node
}

func (self *_parser) parseWithStatement() ast.Statement {
	self.expect(token.WITH)

	// Find the comments after with
	comments := self.findComments(true)

	self.expect(token.LEFT_PARENTHESIS)

	node := &ast.WithStatement{
		Object: self.parseExpression(),
	}
	self.expect(token.RIGHT_PARENTHESIS)

	// Add the key comments
	if self.mode&StoreComments != 0 {
		self.commentMap.AddComments(node, comments, ast.KEY)
	}

	// Find the leading comments for the body
	comments = self.findComments(true)

	node.Body = self.parseStatement()

	// Add the body comments
	if self.mode&StoreComments != 0 {
		self.commentMap.AddComments(node.Body, comments, ast.LEADING)
	}

	// Move the trailing comments to the with statement
	self.commentMap.MoveComments(node.Body, node, ast.TRAILING)

	return node
}

func (self *_parser) parseCaseStatement() *ast.CaseStatement {

	var comments []*ast.Comment

	node := &ast.CaseStatement{
		Case: self.idx,
	}

	if self.mode&StoreComments != 0 {
		self.commentMap.AddComments(node, self.findComments(true), ast.LEADING)
	}

	// Consume current comments
	self.consumeComments(node, ast.LEADING)

	if self.token == token.DEFAULT {
		self.next()
	} else {
		self.expect(token.CASE)

		comments = self.findComments(true)

		node.Test = self.parseExpression()
		if self.mode&StoreComments != 0 {
			self.commentMap.AddComments(node.Test, comments, ast.LEADING)
		}

		comments = self.findComments(true)
		if self.mode&StoreComments != 0 {
			self.commentMap.AddComments(node.Test, comments, ast.TRAILING)
		}
	}

	self.expect(token.COLON)

	if self.mode&StoreComments != 0 {
		self.commentMap.AddComments(node.Test, self.findComments(false), ast.TRAILING)
	}

	for {
		if self.token == token.EOF ||
			self.token == token.RIGHT_BRACE ||
			self.token == token.CASE ||
			self.token == token.DEFAULT {
			break
		}
		consequent := self.parseStatement()
		node.Consequent = append(node.Consequent, consequent)

		if self.mode&StoreComments != 0 {
			self.commentMap.AddComments(consequent, self.findComments(false), ast.TRAILING)
		}
	}

	return node
}

func (self *_parser) parseIterationStatement() ast.Statement {
	inIteration := self.scope.inIteration
	self.scope.inIteration = true
	defer func() {
		self.scope.inIteration = inIteration
	}()
	return self.parseStatement()
}

func (self *_parser) parseForIn(into ast.Expression) *ast.ForInStatement {

	// Already have consumed "<into> in"

	// Comments after the in, before the expression
	comments := self.findComments(true)

	source := self.parseExpression()
	if self.mode&StoreComments != 0 {
		self.commentMap.AddComments(source, comments, ast.LEADING)
	}

	self.expect(token.RIGHT_PARENTHESIS)

	comments = self.findComments(true)
	body := self.parseIterationStatement()
	if self.mode&StoreComments != 0 {
		self.commentMap.AddComments(body, comments, ast.LEADING)
	}

	forin := &ast.ForInStatement{
		Into:   into,
		Source: source,
		Body:   body,
	}

	self.commentMap.MoveComments(body, forin, ast.TRAILING)

	return forin
}

func (self *_parser) parseFor(initializer ast.Expression) *ast.ForStatement {

	// Already have consumed "<initializer> ;"

	comments := self.findComments(true)

	var test, update ast.Expression

	if self.token != token.SEMICOLON {
		test = self.parseExpression()
		if self.mode&StoreComments != 0 {
			self.commentMap.AddComments(test, comments, ast.LEADING)
		}
	}
	self.expect(token.SEMICOLON)

	comments = self.findComments(true)

	if self.token != token.RIGHT_PARENTHESIS {
		update = self.parseExpression()
		if self.mode&StoreComments != 0 {
			self.commentMap.AddComments(update, comments, ast.LEADING)
		}
	}
	self.expect(token.RIGHT_PARENTHESIS)

	comments = self.findComments(true)

	body := self.parseIterationStatement()
	if self.mode&StoreComments != 0 {
		self.commentMap.AddComments(body, comments, ast.LEADING)
	}

	forstatement := &ast.ForStatement{
		Initializer: initializer,
		Test:        test,
		Update:      update,
		Body:        body,
	}

	self.commentMap.MoveComments(body, forstatement, ast.TRAILING)

	return forstatement
}

func (self *_parser) parseForOrForInStatement() ast.Statement {
	idx := self.expect(token.FOR)
	self.expect(token.LEFT_PARENTHESIS)

	comments := self.findComments(true)

	var left []ast.Expression

	forIn := false
	if self.token != token.SEMICOLON {

		allowIn := self.scope.allowIn
		self.scope.allowIn = false
		if self.token == token.VAR {
			var_ := self.idx
			self.next()
			list := self.parseVariableDeclarationList(var_)
			if len(list) == 1 && self.token == token.IN {
				self.next() // in
				forIn = true
				left = []ast.Expression{list[0]} // There is only one declaration
			} else {
				left = list
			}
		} else {
			left = append(left, self.parseExpression())
			if self.token == token.IN {
				self.next()
				forIn = true
			}
		}
		self.scope.allowIn = allowIn
	}

	if forIn {
		switch left[0].(type) {
		case *ast.Identifier, *ast.DotExpression, *ast.BracketExpression, *ast.VariableExpression:
			// These are all acceptable
		default:
			self.error(idx, "Invalid left-hand side in for-in")
			self.nextStatement()
			return &ast.BadStatement{From: idx, To: self.idx}
		}

		if self.mode&StoreComments != 0 {
			self.commentMap.AddComments(left[0], comments, ast.LEADING)
		}
		return self.parseForIn(left[0])
	}

	self.expect(token.SEMICOLON)
	initializer := &ast.SequenceExpression{Sequence: left}
	if self.mode&StoreComments != 0 {
		self.commentMap.AddComments(initializer, comments, ast.LEADING)
	}
	return self.parseFor(initializer)
}

func (self *_parser) parseVariableStatement() *ast.VariableStatement {

	idx := self.expect(token.VAR)

	list := self.parseVariableDeclarationList(idx)

	statement := &ast.VariableStatement{
		Var:  idx,
		List: list,
	}

	self.commentMap.MoveComments(statement.List[len(statement.List)-1], statement, ast.TRAILING)

	if self.mode&StoreComments != 0 {
		self.commentMap.AddComments(statement, self.findComments(true), ast.TRAILING)
	}

	self.semicolon()

	if self.skippedLineBreak {
		if self.mode&StoreComments != 0 {
			self.commentMap.AddComments(statement, self.findComments(false), ast.TRAILING)
		}
	}

	return statement
}

func (self *_parser) parseDoWhileStatement() ast.Statement {
	inIteration := self.scope.inIteration
	self.scope.inIteration = true
	defer func() {
		self.scope.inIteration = inIteration
	}()

	self.expect(token.DO)

	comments := self.findComments(true)

	node := &ast.DoWhileStatement{}
	if self.mode&StoreComments != 0 {
		self.commentMap.AddComments(node, comments, ast.KEY)
	}
	if self.token == token.LEFT_BRACE {
		node.Body = self.parseBlockStatement()
	} else {
		node.Body = self.parseStatement()
	}

	self.expect(token.WHILE)

	comments = self.findComments(true)

	self.expect(token.LEFT_PARENTHESIS)
	node.Test = self.parseExpression()
	if self.mode&StoreComments != 0 {
		self.commentMap.AddComments(node.Test, comments, ast.LEADING)
	}

	self.expect(token.RIGHT_PARENTHESIS)

	if self.mode&StoreComments != 0 {
		self.commentMap.AddComments(node.Test, self.findComments(false), ast.TRAILING)
	}

	return node
}

func (self *_parser) parseWhileStatement() ast.Statement {
	self.expect(token.WHILE)

	// Comments after while keyword
	comments := self.findComments(true)

	self.expect(token.LEFT_PARENTHESIS)
	node := &ast.WhileStatement{
		Test: self.parseExpression(),
	}

	// Add the while comments
	if self.mode&StoreComments != 0 {
		self.commentMap.AddComments(node, comments, ast.KEY)
	}

	self.expect(token.RIGHT_PARENTHESIS)

	// Finding comments prior to the body
	comments = self.findComments(true)

	node.Body = self.parseIterationStatement()

	// Adding the comments prior to the body
	if self.mode&StoreComments != 0 {
		self.commentMap.AddComments(node.Body, comments, ast.LEADING)
	}

	// Move the trailing comments to the while statement
	self.commentMap.MoveComments(node.Body, node, ast.TRAILING)

	return node
}

func (self *_parser) parseIfStatement() ast.Statement {
	self.expect(token.IF)

	comments := self.findComments(true)

	self.expect(token.LEFT_PARENTHESIS)
	node := &ast.IfStatement{
		Test: self.parseExpression(),
	}

	if self.mode&StoreComments != 0 {
		self.commentMap.AddComments(node, comments, ast.KEY)
	}

	self.expect(token.RIGHT_PARENTHESIS)

	comments = self.findComments(true)

	if self.token == token.LEFT_BRACE {
		node.Consequent = self.parseBlockStatement()
	} else {
		node.Consequent = self.parseStatement()
	}

	if self.mode&StoreComments != 0 {
		self.commentMap.AddComments(node.Consequent, comments, ast.LEADING)
		self.commentMap.AddComments(node.Consequent, self.findComments(true), ast.TRAILING)
	}

	if self.token == token.ELSE {
		self.next()
		comments = self.findComments(true)

		node.Alternate = self.parseStatement()

		if self.mode&StoreComments != 0 {
			self.commentMap.AddComments(node.Alternate, comments, ast.LEADING)
			self.commentMap.AddComments(node.Alternate, self.findComments(false), ast.TRAILING)
		}
	}

	return node
}

func (self *_parser) parseSourceElement() ast.Statement {

	statementComment := self.fetchComments()

	statement := self.parseStatement()

	if self.mode&StoreComments != 0 {
		self.commentMap.AddComments(statement, statementComment, ast.LEADING)
	}

	return statement
}

func (self *_parser) parseCommentElement() {
	literal := self.literal
	idx := self.expect(token.COMMENT)
	self.comments = append(self.comments, &ast.Comment{
		Begin:    idx,
		Text:     literal,
		Position: ast.LEADING,
	})
}

func (self *_parser) parseSourceElements() []ast.Statement {
	body := []ast.Statement(nil)

	for {
		if self.token != token.STRING {
			break
		}

		if self.token == token.COMMENT {
			self.parseCommentElement()
			continue
		}

		body = append(body, self.parseSourceElement())
	}

	for self.token != token.EOF {
		if self.token == token.COMMENT {
			self.parseCommentElement()
			continue
		}
		body = append(body, self.parseSourceElement())
	}

	return body
}

func (self *_parser) parseProgram() *ast.Program {
	self.openScope()
	defer self.closeScope()
	return &ast.Program{
		Body:            self.parseSourceElements(),
		DeclarationList: self.scope.declarationList,
		File:            self.file,
	}
}

func (self *_parser) parseBreakStatement() ast.Statement {
	idx := self.expect(token.BREAK)

	breakComments := self.findComments(true)

	semicolon := self.implicitSemicolon
	if self.token == token.SEMICOLON {
		semicolon = true
		self.next()
	}

	if semicolon || self.token == token.RIGHT_BRACE {
		self.implicitSemicolon = false
		if !self.scope.inIteration && !self.scope.inSwitch {
			goto illegal
		}
		breakStatement := &ast.BranchStatement{
			Idx:   idx,
			Token: token.BREAK,
		}

		if self.mode&StoreComments != 0 {
			self.commentMap.AddComments(breakStatement, breakComments, ast.TRAILING)
		}

		return breakStatement
	}

	if self.token == token.IDENTIFIER {
		identifier := self.parseIdentifier()
		if !self.scope.hasLabel(identifier.Name) {
			self.error(idx, "Undefined label '%s'", identifier.Name)
			return &ast.BadStatement{From: idx, To: identifier.Idx1()}
		}
		self.semicolon()
		return &ast.BranchStatement{
			Idx:   idx,
			Token: token.BREAK,
			Label: identifier,
		}
	}

	self.expect(token.IDENTIFIER)

illegal:
	self.error(idx, "Illegal break statement")
	self.nextStatement()
	return &ast.BadStatement{From: idx, To: self.idx}
}

func (self *_parser) parseContinueStatement() ast.Statement {
	idx := self.expect(token.CONTINUE)
	semicolon := self.implicitSemicolon
	if self.token == token.SEMICOLON {
		semicolon = true
		self.next()
	}

	if semicolon || self.token == token.RIGHT_BRACE {
		self.implicitSemicolon = false
		if !self.scope.inIteration {
			goto illegal
		}
		return &ast.BranchStatement{
			Idx:   idx,
			Token: token.CONTINUE,
		}
	}

	if self.token == token.IDENTIFIER {
		identifier := self.parseIdentifier()
		if !self.scope.hasLabel(identifier.Name) {
			self.error(idx, "Undefined label '%s'", identifier.Name)
			return &ast.BadStatement{From: idx, To: identifier.Idx1()}
		}
		if !self.scope.inIteration {
			goto illegal
		}
		self.semicolon()
		return &ast.BranchStatement{
			Idx:   idx,
			Token: token.CONTINUE,
			Label: identifier,
		}
	}

	self.expect(token.IDENTIFIER)

illegal:
	self.error(idx, "Illegal continue statement")
	self.nextStatement()
	return &ast.BadStatement{From: idx, To: self.idx}
}

// Find the next statement after an error (recover)
func (self *_parser) nextStatement() {
	for {
		switch self.token {
		case token.BREAK, token.CONTINUE,
			token.FOR, token.IF, token.RETURN, token.SWITCH,
			token.VAR, token.DO, token.TRY, token.WITH,
			token.WHILE, token.THROW, token.CATCH, token.FINALLY:
			// Return only if parser made some progress since last
			// sync or if it has not reached 10 next calls without
			// progress. Otherwise consume at least one token to
			// avoid an endless parser loop
			if self.idx == self.recover.idx && self.recover.count < 10 {
				self.recover.count++
				return
			}
			if self.idx > self.recover.idx {
				self.recover.idx = self.idx
				self.recover.count = 0
				return
			}
			// Reaching here indicates a parser bug, likely an
			// incorrect token list in this function, but it only
			// leads to skipping of possibly correct code if a
			// previous error is present, and thus is preferred
			// over a non-terminating parse.
		case token.EOF:
			return
		}
		self.next()
	}
}
