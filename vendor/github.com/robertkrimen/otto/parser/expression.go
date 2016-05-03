package parser

import (
	"regexp"

	"github.com/robertkrimen/otto/ast"
	"github.com/robertkrimen/otto/file"
	"github.com/robertkrimen/otto/token"
)

func (self *_parser) parseIdentifier() *ast.Identifier {
	literal := self.literal
	idx := self.idx
	self.next()
	comments := self.findComments(false)
	exp := &ast.Identifier{
		Name: literal,
		Idx:  idx,
	}

	self.commentMap.AddComments(exp, comments, ast.TRAILING)
	return exp
}

func (self *_parser) parsePrimaryExpression() ast.Expression {
	literal := self.literal
	idx := self.idx
	switch self.token {
	case token.IDENTIFIER:
		self.next()
		if len(literal) > 1 {
			tkn, strict := token.IsKeyword(literal)
			if tkn == token.KEYWORD {
				if !strict {
					self.error(idx, "Unexpected reserved word")
				}
			}
		}
		return &ast.Identifier{
			Name: literal,
			Idx:  idx,
		}
	case token.NULL:
		self.next()
		return &ast.NullLiteral{
			Idx:     idx,
			Literal: literal,
		}
	case token.BOOLEAN:
		self.next()
		value := false
		switch literal {
		case "true":
			value = true
		case "false":
			value = false
		default:
			self.error(idx, "Illegal boolean literal")
		}
		return &ast.BooleanLiteral{
			Idx:     idx,
			Literal: literal,
			Value:   value,
		}
	case token.STRING:
		self.next()
		value, err := parseStringLiteral(literal[1 : len(literal)-1])
		if err != nil {
			self.error(idx, err.Error())
		}
		return &ast.StringLiteral{
			Idx:     idx,
			Literal: literal,
			Value:   value,
		}
	case token.NUMBER:
		self.next()
		value, err := parseNumberLiteral(literal)
		if err != nil {
			self.error(idx, err.Error())
			value = 0
		}
		return &ast.NumberLiteral{
			Idx:     idx,
			Literal: literal,
			Value:   value,
		}
	case token.SLASH, token.QUOTIENT_ASSIGN:
		return self.parseRegExpLiteral()
	case token.LEFT_BRACE:
		return self.parseObjectLiteral()
	case token.LEFT_BRACKET:
		return self.parseArrayLiteral()
	case token.LEFT_PARENTHESIS:
		self.expect(token.LEFT_PARENTHESIS)
		expression := self.parseExpression()
		self.expect(token.RIGHT_PARENTHESIS)
		return expression
	case token.THIS:
		self.next()
		return &ast.ThisExpression{
			Idx: idx,
		}
	case token.FUNCTION:
		return self.parseFunction(false)
	}

	self.errorUnexpectedToken(self.token)
	self.nextStatement()
	return &ast.BadExpression{From: idx, To: self.idx}
}

func (self *_parser) parseRegExpLiteral() *ast.RegExpLiteral {

	offset := self.chrOffset - 1 // Opening slash already gotten
	if self.token == token.QUOTIENT_ASSIGN {
		offset -= 1 // =
	}
	idx := self.idxOf(offset)

	pattern, err := self.scanString(offset)
	endOffset := self.chrOffset

	self.next()
	if err == nil {
		pattern = pattern[1 : len(pattern)-1]
	}

	flags := ""
	if self.token == token.IDENTIFIER { // gim

		flags = self.literal
		self.next()
		endOffset = self.chrOffset - 1
	}

	var value string
	// TODO 15.10
	{
		// Test during parsing that this is a valid regular expression
		// Sorry, (?=) and (?!) are invalid (for now)
		pattern, err := TransformRegExp(pattern)
		if err != nil {
			if pattern == "" || self.mode&IgnoreRegExpErrors == 0 {
				self.error(idx, "Invalid regular expression: %s", err.Error())
			}
		} else {
			_, err = regexp.Compile(pattern)
			if err != nil {
				// We should not get here, ParseRegExp should catch any errors
				self.error(idx, "Invalid regular expression: %s", err.Error()[22:]) // Skip redundant "parse regexp error"
			} else {
				value = pattern
			}
		}
	}

	literal := self.str[offset:endOffset]

	return &ast.RegExpLiteral{
		Idx:     idx,
		Literal: literal,
		Pattern: pattern,
		Flags:   flags,
		Value:   value,
	}
}

func (self *_parser) parseVariableDeclaration(declarationList *[]*ast.VariableExpression) ast.Expression {

	if self.token != token.IDENTIFIER {
		idx := self.expect(token.IDENTIFIER)
		self.nextStatement()
		return &ast.BadExpression{From: idx, To: self.idx}
	}

	literal := self.literal
	idx := self.idx
	self.next()
	node := &ast.VariableExpression{
		Name: literal,
		Idx:  idx,
	}

	if declarationList != nil {
		*declarationList = append(*declarationList, node)
	}

	if self.token == token.ASSIGN {
		self.next()
		node.Initializer = self.parseAssignmentExpression()
	}

	return node
}

func (self *_parser) parseVariableDeclarationList(var_ file.Idx) []ast.Expression {

	var declarationList []*ast.VariableExpression // Avoid bad expressions
	var list []ast.Expression

	for {
		comments := self.findComments(false)

		decl := self.parseVariableDeclaration(&declarationList)
		if self.mode&StoreComments != 0 {
			self.commentMap.AddComments(decl, comments, ast.LEADING)
			self.commentMap.AddComments(decl, self.findComments(false), ast.TRAILING)
		}

		list = append(list, decl)
		if self.token != token.COMMA {
			break
		}
		self.next()

	}

	self.scope.declare(&ast.VariableDeclaration{
		Var:  var_,
		List: declarationList,
	})

	return list
}

func (self *_parser) parseObjectPropertyKey() (string, string, []*ast.Comment) {
	idx, tkn, literal := self.idx, self.token, self.literal
	value := ""
	self.next()

	comments := self.findComments(false)

	switch tkn {
	case token.IDENTIFIER:
		value = literal
	case token.NUMBER:
		var err error
		_, err = parseNumberLiteral(literal)
		if err != nil {
			self.error(idx, err.Error())
		} else {
			value = literal
		}
	case token.STRING:
		var err error
		value, err = parseStringLiteral(literal[1 : len(literal)-1])
		if err != nil {
			self.error(idx, err.Error())
		}
	default:
		// null, false, class, etc.
		if matchIdentifier.MatchString(literal) {
			value = literal
		}
	}
	return literal, value, comments
}

func (self *_parser) parseObjectProperty() ast.Property {
	literal, value, comments := self.parseObjectPropertyKey()
	if literal == "get" && self.token != token.COLON {
		idx := self.idx
		_, value, _ := self.parseObjectPropertyKey()
		parameterList := self.parseFunctionParameterList()

		node := &ast.FunctionLiteral{
			Function:      idx,
			ParameterList: parameterList,
		}
		self.parseFunctionBlock(node)
		return ast.Property{
			Key:   value,
			Kind:  "get",
			Value: node,
		}
	} else if literal == "set" && self.token != token.COLON {
		idx := self.idx
		_, value, _ := self.parseObjectPropertyKey()
		parameterList := self.parseFunctionParameterList()

		node := &ast.FunctionLiteral{
			Function:      idx,
			ParameterList: parameterList,
		}
		self.parseFunctionBlock(node)
		return ast.Property{
			Key:   value,
			Kind:  "set",
			Value: node,
		}
	}

	self.expect(token.COLON)
	comments2 := self.findComments(false)

	exp := ast.Property{
		Key:   value,
		Kind:  "value",
		Value: self.parseAssignmentExpression(),
	}

	if self.mode&StoreComments != 0 {
		self.commentMap.AddComments(exp.Value, comments, ast.KEY)
		self.commentMap.AddComments(exp.Value, comments2, ast.COLON)
	}
	return exp
}

func (self *_parser) parseObjectLiteral() ast.Expression {
	var value []ast.Property
	idx0 := self.expect(token.LEFT_BRACE)

	var comments2 []*ast.Comment
	for self.token != token.RIGHT_BRACE && self.token != token.EOF {

		// Leading comments for object literal
		comments := self.findComments(false)
		property := self.parseObjectProperty()
		if self.mode&StoreComments != 0 {
			self.commentMap.AddComments(property.Value, comments, ast.LEADING)
			self.commentMap.AddComments(property.Value, comments2, ast.LEADING)
		}
		value = append(value, property)
		if self.token == token.COMMA {
			self.next()

			// Find leading comments after trailing comma
			comments2 = self.findComments(false)
			continue
		}
	}
	idx1 := self.expect(token.RIGHT_BRACE)

	exp := &ast.ObjectLiteral{
		LeftBrace:  idx0,
		RightBrace: idx1,
		Value:      value,
	}

	if self.mode&StoreComments != 0 {
		self.commentMap.AddComments(exp, comments2, ast.FINAL)
	}
	self.consumeComments(exp, ast.FINAL)

	return exp
}

func (self *_parser) parseArrayLiteral() ast.Expression {
	idx0 := self.expect(token.LEFT_BRACKET)
	var comments2 []*ast.Comment
	var comments []*ast.Comment
	var value []ast.Expression
	for self.token != token.RIGHT_BRACKET && self.token != token.EOF {
		// Find leading comments for both empty and non-empty expressions
		comments = self.findComments(false)

		if self.token == token.COMMA {
			self.next()

			// This kind of comment requires a special empty expression node.
			empty := &ast.EmptyExpression{self.idx, self.idx}

			if self.mode&StoreComments != 0 {
				self.commentMap.AddComments(empty, comments, ast.LEADING)
				self.commentMap.AddComments(empty, comments2, ast.LEADING)
			}

			value = append(value, empty)

			// This comment belongs to the following expression, or trailing
			comments2 = self.findComments(false)

			continue
		}

		exp := self.parseAssignmentExpression()
		if self.mode&StoreComments != 0 {
			self.commentMap.AddComments(exp, comments, ast.LEADING)
			self.commentMap.AddComments(exp, comments2, ast.LEADING)
		}

		value = append(value, exp)
		if self.token != token.RIGHT_BRACKET {
			self.expect(token.COMMA)
		}

		// This comment belongs to the following expression, or trailing
		comments2 = self.findComments(false)
	}
	idx1 := self.expect(token.RIGHT_BRACKET)

	array := &ast.ArrayLiteral{
		LeftBracket:  idx0,
		RightBracket: idx1,
		Value:        value,
	}

	// This is where comments after a possible trailing comma are added
	if self.mode&StoreComments != 0 {
		self.commentMap.AddComments(array, comments2, ast.FINAL)
	}

	return array
}

func (self *_parser) parseArgumentList() (argumentList []ast.Expression, idx0, idx1 file.Idx) {
	idx0 = self.expect(token.LEFT_PARENTHESIS)
	if self.token != token.RIGHT_PARENTHESIS {
		for {
			comments := self.findComments(false)
			exp := self.parseAssignmentExpression()
			if self.mode&StoreComments != 0 {
				self.commentMap.AddComments(exp, comments, ast.LEADING)
			}
			argumentList = append(argumentList, exp)
			if self.token != token.COMMA {
				break
			}
			self.next()
		}
	}
	idx1 = self.expect(token.RIGHT_PARENTHESIS)
	return
}

func (self *_parser) parseCallExpression(left ast.Expression) ast.Expression {
	argumentList, idx0, idx1 := self.parseArgumentList()
	exp := &ast.CallExpression{
		Callee:           left,
		LeftParenthesis:  idx0,
		ArgumentList:     argumentList,
		RightParenthesis: idx1,
	}

	if self.mode&StoreComments != 0 {
		self.commentMap.AddComments(exp, self.findComments(false), ast.TRAILING)
	}
	return exp
}

func (self *_parser) parseDotMember(left ast.Expression) ast.Expression {
	period := self.expect(token.PERIOD)

	literal := self.literal
	idx := self.idx

	if !matchIdentifier.MatchString(literal) {
		self.expect(token.IDENTIFIER)
		self.nextStatement()
		return &ast.BadExpression{From: period, To: self.idx}
	}

	self.next()

	return &ast.DotExpression{
		Left: left,
		Identifier: ast.Identifier{
			Idx:  idx,
			Name: literal,
		},
	}
}

func (self *_parser) parseBracketMember(left ast.Expression) ast.Expression {
	idx0 := self.expect(token.LEFT_BRACKET)
	member := self.parseExpression()
	idx1 := self.expect(token.RIGHT_BRACKET)
	return &ast.BracketExpression{
		LeftBracket:  idx0,
		Left:         left,
		Member:       member,
		RightBracket: idx1,
	}
}

func (self *_parser) parseNewExpression() ast.Expression {
	idx := self.expect(token.NEW)
	callee := self.parseLeftHandSideExpression()
	node := &ast.NewExpression{
		New:    idx,
		Callee: callee,
	}
	if self.token == token.LEFT_PARENTHESIS {
		argumentList, idx0, idx1 := self.parseArgumentList()
		node.ArgumentList = argumentList
		node.LeftParenthesis = idx0
		node.RightParenthesis = idx1
	}

	if self.mode&StoreComments != 0 {
		self.commentMap.AddComments(node, self.findComments(false), ast.TRAILING)
	}

	return node
}

func (self *_parser) parseLeftHandSideExpression() ast.Expression {

	var left ast.Expression
	if self.token == token.NEW {
		left = self.parseNewExpression()
	} else {
		left = self.parsePrimaryExpression()
	}

	if self.mode&StoreComments != 0 {
		self.commentMap.AddComments(left, self.findComments(false), ast.TRAILING)
	}

	for {
		if self.token == token.PERIOD {
			left = self.parseDotMember(left)
		} else if self.token == token.LEFT_BRACKET {
			left = self.parseBracketMember(left)
		} else {
			break
		}
	}

	return left
}

func (self *_parser) parseLeftHandSideExpressionAllowCall() ast.Expression {

	allowIn := self.scope.allowIn
	self.scope.allowIn = true
	defer func() {
		self.scope.allowIn = allowIn
	}()

	var left ast.Expression
	if self.token == token.NEW {
		left = self.parseNewExpression()
	} else {
		left = self.parsePrimaryExpression()
	}

	if self.mode&StoreComments != 0 {
		self.commentMap.AddComments(left, self.findComments(false), ast.TRAILING)
	}

	for {
		if self.token == token.PERIOD {
			left = self.parseDotMember(left)
		} else if self.token == token.LEFT_BRACKET {
			left = self.parseBracketMember(left)
		} else if self.token == token.LEFT_PARENTHESIS {
			left = self.parseCallExpression(left)
		} else {
			break
		}
	}

	return left
}

func (self *_parser) parsePostfixExpression() ast.Expression {
	operand := self.parseLeftHandSideExpressionAllowCall()

	switch self.token {
	case token.INCREMENT, token.DECREMENT:
		// Make sure there is no line terminator here
		if self.implicitSemicolon {
			break
		}
		tkn := self.token
		idx := self.idx
		self.next()
		switch operand.(type) {
		case *ast.Identifier, *ast.DotExpression, *ast.BracketExpression:
		default:
			self.error(idx, "Invalid left-hand side in assignment")
			self.nextStatement()
			return &ast.BadExpression{From: idx, To: self.idx}
		}
		exp := &ast.UnaryExpression{
			Operator: tkn,
			Idx:      idx,
			Operand:  operand,
			Postfix:  true,
		}

		if self.mode&StoreComments != 0 {
			self.commentMap.AddComments(exp, self.findComments(false), ast.TRAILING)
		}

		return exp
	}

	return operand
}

func (self *_parser) parseUnaryExpression() ast.Expression {

	switch self.token {
	case token.PLUS, token.MINUS, token.NOT, token.BITWISE_NOT:
		fallthrough
	case token.DELETE, token.VOID, token.TYPEOF:
		tkn := self.token
		idx := self.idx
		self.next()

		comments := self.findComments(false)

		exp := &ast.UnaryExpression{
			Operator: tkn,
			Idx:      idx,
			Operand:  self.parseUnaryExpression(),
		}

		if self.mode&StoreComments != 0 {
			self.commentMap.AddComments(exp.Operand, comments, ast.LEADING)
		}
		return exp
	case token.INCREMENT, token.DECREMENT:
		tkn := self.token
		idx := self.idx
		self.next()

		comments := self.findComments(false)

		operand := self.parseUnaryExpression()
		if self.mode&StoreComments != 0 {
			self.commentMap.AddComments(operand, comments, ast.LEADING)
		}
		switch operand.(type) {
		case *ast.Identifier, *ast.DotExpression, *ast.BracketExpression:
		default:
			self.error(idx, "Invalid left-hand side in assignment")
			self.nextStatement()
			return &ast.BadExpression{From: idx, To: self.idx}
		}
		return &ast.UnaryExpression{
			Operator: tkn,
			Idx:      idx,
			Operand:  operand,
		}
	}

	return self.parsePostfixExpression()
}

func (self *_parser) parseMultiplicativeExpression() ast.Expression {
	next := self.parseUnaryExpression
	left := next()

	for self.token == token.MULTIPLY || self.token == token.SLASH ||
		self.token == token.REMAINDER {
		tkn := self.token
		self.next()

		comments := self.findComments(false)

		left = &ast.BinaryExpression{
			Operator: tkn,
			Left:     left,
			Right:    next(),
		}

		if self.mode&StoreComments != 0 {
			self.commentMap.AddComments(left.(*ast.BinaryExpression).Right, comments, ast.LEADING)
		}
	}

	return left
}

func (self *_parser) parseAdditiveExpression() ast.Expression {
	next := self.parseMultiplicativeExpression
	left := next()

	for self.token == token.PLUS || self.token == token.MINUS {
		tkn := self.token
		self.next()

		comments := self.findComments(false)

		left = &ast.BinaryExpression{
			Operator: tkn,
			Left:     left,
			Right:    next(),
		}

		if self.mode&StoreComments != 0 {
			self.commentMap.AddComments(left.(*ast.BinaryExpression).Right, comments, ast.LEADING)
		}
	}

	return left
}

func (self *_parser) parseShiftExpression() ast.Expression {
	next := self.parseAdditiveExpression
	left := next()

	for self.token == token.SHIFT_LEFT || self.token == token.SHIFT_RIGHT ||
		self.token == token.UNSIGNED_SHIFT_RIGHT {
		tkn := self.token
		self.next()

		comments := self.findComments(false)

		left = &ast.BinaryExpression{
			Operator: tkn,
			Left:     left,
			Right:    next(),
		}

		if self.mode&StoreComments != 0 {
			self.commentMap.AddComments(left.(*ast.BinaryExpression).Right, comments, ast.LEADING)
		}
	}

	return left
}

func (self *_parser) parseRelationalExpression() ast.Expression {
	next := self.parseShiftExpression
	left := next()

	allowIn := self.scope.allowIn
	self.scope.allowIn = true
	defer func() {
		self.scope.allowIn = allowIn
	}()

	switch self.token {
	case token.LESS, token.LESS_OR_EQUAL, token.GREATER, token.GREATER_OR_EQUAL:
		tkn := self.token
		self.next()

		comments := self.findComments(false)

		exp := &ast.BinaryExpression{
			Operator:   tkn,
			Left:       left,
			Right:      self.parseRelationalExpression(),
			Comparison: true,
		}

		if self.mode&StoreComments != 0 {
			self.commentMap.AddComments(exp.Right, comments, ast.LEADING)
		}
		return exp
	case token.INSTANCEOF:
		tkn := self.token
		self.next()

		comments := self.findComments(false)

		exp := &ast.BinaryExpression{
			Operator: tkn,
			Left:     left,
			Right:    self.parseRelationalExpression(),
		}

		if self.mode&StoreComments != 0 {
			self.commentMap.AddComments(exp.Right, comments, ast.LEADING)
		}
		return exp
	case token.IN:
		if !allowIn {
			return left
		}
		tkn := self.token
		self.next()

		comments := self.findComments(false)

		exp := &ast.BinaryExpression{
			Operator: tkn,
			Left:     left,
			Right:    self.parseRelationalExpression(),
		}

		if self.mode&StoreComments != 0 {
			self.commentMap.AddComments(exp.Right, comments, ast.LEADING)
		}
		return exp
	}

	return left
}

func (self *_parser) parseEqualityExpression() ast.Expression {
	next := self.parseRelationalExpression
	left := next()

	for self.token == token.EQUAL || self.token == token.NOT_EQUAL ||
		self.token == token.STRICT_EQUAL || self.token == token.STRICT_NOT_EQUAL {
		tkn := self.token
		self.next()

		comments := self.findComments(false)

		left = &ast.BinaryExpression{
			Operator:   tkn,
			Left:       left,
			Right:      next(),
			Comparison: true,
		}

		if self.mode&StoreComments != 0 {
			self.commentMap.AddComments(left.(*ast.BinaryExpression).Right, comments, ast.LEADING)
		}
	}

	return left
}

func (self *_parser) parseBitwiseAndExpression() ast.Expression {
	next := self.parseEqualityExpression
	left := next()

	for self.token == token.AND {
		tkn := self.token
		self.next()

		comments := self.findComments(false)

		left = &ast.BinaryExpression{
			Operator: tkn,
			Left:     left,
			Right:    next(),
		}

		if self.mode&StoreComments != 0 {
			self.commentMap.AddComments(left.(*ast.BinaryExpression).Right, comments, ast.LEADING)
		}
	}

	return left
}

func (self *_parser) parseBitwiseExclusiveOrExpression() ast.Expression {
	next := self.parseBitwiseAndExpression
	left := next()

	for self.token == token.EXCLUSIVE_OR {
		tkn := self.token
		self.next()

		comments := self.findComments(false)

		left = &ast.BinaryExpression{
			Operator: tkn,
			Left:     left,
			Right:    next(),
		}

		if self.mode&StoreComments != 0 {
			self.commentMap.AddComments(left.(*ast.BinaryExpression).Right, comments, ast.LEADING)
		}
	}

	return left
}

func (self *_parser) parseBitwiseOrExpression() ast.Expression {
	next := self.parseBitwiseExclusiveOrExpression
	left := next()

	for self.token == token.OR {
		tkn := self.token
		self.next()

		comments := self.findComments(false)

		left = &ast.BinaryExpression{
			Operator: tkn,
			Left:     left,
			Right:    next(),
		}

		if self.mode&StoreComments != 0 {
			self.commentMap.AddComments(left.(*ast.BinaryExpression).Right, comments, ast.LEADING)
		}
	}

	return left
}

func (self *_parser) parseLogicalAndExpression() ast.Expression {
	next := self.parseBitwiseOrExpression
	left := next()

	for self.token == token.LOGICAL_AND {
		tkn := self.token
		self.next()

		comments := self.findComments(false)

		left = &ast.BinaryExpression{
			Operator: tkn,
			Left:     left,
			Right:    next(),
		}

		if self.mode&StoreComments != 0 {
			self.commentMap.AddComments(left.(*ast.BinaryExpression).Right, comments, ast.LEADING)
		}
	}

	return left
}

func (self *_parser) parseLogicalOrExpression() ast.Expression {
	next := self.parseLogicalAndExpression
	left := next()

	for self.token == token.LOGICAL_OR {
		tkn := self.token
		self.next()

		comments := self.findComments(false)

		left = &ast.BinaryExpression{
			Operator: tkn,
			Left:     left,
			Right:    next(),
		}

		if self.mode&StoreComments != 0 {
			self.commentMap.AddComments(left.(*ast.BinaryExpression).Right, comments, ast.LEADING)
		}
	}

	return left
}

func (self *_parser) parseConditionlExpression() ast.Expression {
	left := self.parseLogicalOrExpression()

	if self.token == token.QUESTION_MARK {
		self.next()

		// Comments before the consequence
		comments1 := self.findComments(false)

		consequent := self.parseAssignmentExpression()
		if self.mode&StoreComments != 0 {
			self.commentMap.AddComments(consequent, comments1, ast.LEADING)
		}

		self.expect(token.COLON)

		// Comments before the alternate
		comments2 := self.findComments(false)
		exp := &ast.ConditionalExpression{
			Test:       left,
			Consequent: consequent,
			Alternate:  self.parseAssignmentExpression(),
		}

		if self.mode&StoreComments != 0 {
			self.commentMap.AddComments(exp.Alternate, comments2, ast.LEADING)
		}
		return exp
	}

	return left
}

func (self *_parser) parseAssignmentExpression() ast.Expression {
	left := self.parseConditionlExpression()
	var operator token.Token
	switch self.token {
	case token.ASSIGN:
		operator = self.token
	case token.ADD_ASSIGN:
		operator = token.PLUS
	case token.SUBTRACT_ASSIGN:
		operator = token.MINUS
	case token.MULTIPLY_ASSIGN:
		operator = token.MULTIPLY
	case token.QUOTIENT_ASSIGN:
		operator = token.SLASH
	case token.REMAINDER_ASSIGN:
		operator = token.REMAINDER
	case token.AND_ASSIGN:
		operator = token.AND
	case token.AND_NOT_ASSIGN:
		operator = token.AND_NOT
	case token.OR_ASSIGN:
		operator = token.OR
	case token.EXCLUSIVE_OR_ASSIGN:
		operator = token.EXCLUSIVE_OR
	case token.SHIFT_LEFT_ASSIGN:
		operator = token.SHIFT_LEFT
	case token.SHIFT_RIGHT_ASSIGN:
		operator = token.SHIFT_RIGHT
	case token.UNSIGNED_SHIFT_RIGHT_ASSIGN:
		operator = token.UNSIGNED_SHIFT_RIGHT
	}

	if operator != 0 {
		idx := self.idx
		self.next()
		switch left.(type) {
		case *ast.Identifier, *ast.DotExpression, *ast.BracketExpression:
		default:
			self.error(left.Idx0(), "Invalid left-hand side in assignment")
			self.nextStatement()
			return &ast.BadExpression{From: idx, To: self.idx}
		}

		comments := self.findComments(false)

		exp := &ast.AssignExpression{
			Left:     left,
			Operator: operator,
			Right:    self.parseAssignmentExpression(),
		}

		if self.mode&StoreComments != 0 {
			self.commentMap.AddComments(exp.Right, comments, ast.LEADING)
		}

		return exp
	}

	return left
}

func (self *_parser) parseExpression() ast.Expression {

	comments := self.findComments(false)
	statementComments := self.fetchComments()

	next := self.parseAssignmentExpression
	left := next()

	if self.token == token.COMMA {
		sequence := []ast.Expression{left}
		for {
			if self.token != token.COMMA {
				break
			}
			self.next()
			sequence = append(sequence, next())
		}
		return &ast.SequenceExpression{
			Sequence: sequence,
		}
	}

	if self.mode&StoreComments != 0 {
		self.commentMap.AddComments(left, comments, ast.LEADING)
		self.commentMap.AddComments(left, statementComments, ast.LEADING)
	}

	return left
}
