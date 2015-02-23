package otto

import (
	"fmt"
	"regexp"

	"github.com/robertkrimen/otto/ast"
	"github.com/robertkrimen/otto/token"
)

var trueLiteral = &_nodeLiteral{value: toValue_bool(true)}
var falseLiteral = &_nodeLiteral{value: toValue_bool(false)}
var nullLiteral = &_nodeLiteral{value: NullValue()}
var emptyStatement = &_nodeEmptyStatement{}

func parseExpression(x ast.Expression) _nodeExpression {
	if x == nil {
		return nil
	}

	switch x := x.(type) {

	case *ast.ArrayLiteral:
		y := &_nodeArrayLiteral{
			value: make([]_nodeExpression, len(x.Value)),
		}
		for i, value := range x.Value {
			y.value[i] = parseExpression(value)
		}
		return y

	case *ast.AssignExpression:
		return &_nodeAssignExpression{
			operator: x.Operator,
			left:     parseExpression(x.Left),
			right:    parseExpression(x.Right),
		}

	case *ast.BinaryExpression:
		return &_nodeBinaryExpression{
			operator:   x.Operator,
			left:       parseExpression(x.Left),
			right:      parseExpression(x.Right),
			comparison: x.Comparison,
		}

	case *ast.BooleanLiteral:
		if x.Value {
			return trueLiteral
		}
		return falseLiteral

	case *ast.BracketExpression:
		return &_nodeBracketExpression{
			left:   parseExpression(x.Left),
			member: parseExpression(x.Member),
		}

	case *ast.CallExpression:
		y := &_nodeCallExpression{
			callee:       parseExpression(x.Callee),
			argumentList: make([]_nodeExpression, len(x.ArgumentList)),
		}
		for i, value := range x.ArgumentList {
			y.argumentList[i] = parseExpression(value)
		}
		return y

	case *ast.ConditionalExpression:
		return &_nodeConditionalExpression{
			test:       parseExpression(x.Test),
			consequent: parseExpression(x.Consequent),
			alternate:  parseExpression(x.Alternate),
		}

	case *ast.DotExpression:
		return &_nodeDotExpression{
			left:       parseExpression(x.Left),
			identifier: x.Identifier.Name,
		}

	case *ast.FunctionLiteral:
		name := ""
		if x.Name != nil {
			name = x.Name.Name
		}
		y := &_nodeFunctionLiteral{
			name:   name,
			body:   parseStatement(x.Body),
			source: x.Source,
		}
		if x.ParameterList != nil {
			list := x.ParameterList.List
			y.parameterList = make([]string, len(list))
			for i, value := range list {
				y.parameterList[i] = value.Name
			}
		}
		for _, value := range x.DeclarationList {
			switch value := value.(type) {
			case *ast.FunctionDeclaration:
				y.functionList = append(y.functionList, parseExpression(value.Function).(*_nodeFunctionLiteral))
			case *ast.VariableDeclaration:
				for _, value := range value.List {
					y.varList = append(y.varList, value.Name)
				}
			default:
				panic(fmt.Errorf("Here be dragons: parseProgram.declaration(%T)", value))
			}
		}
		return y

	case *ast.Identifier:
		return &_nodeIdentifier{
			name: x.Name,
		}

	case *ast.NewExpression:
		y := &_nodeNewExpression{
			callee:       parseExpression(x.Callee),
			argumentList: make([]_nodeExpression, len(x.ArgumentList)),
		}
		for i, value := range x.ArgumentList {
			y.argumentList[i] = parseExpression(value)
		}
		return y

	case *ast.NullLiteral:
		return nullLiteral

	case *ast.NumberLiteral:
		return &_nodeLiteral{
			value: toValue(x.Value),
		}

	case *ast.ObjectLiteral:
		y := &_nodeObjectLiteral{
			value: make([]_nodeProperty, len(x.Value)),
		}
		for i, value := range x.Value {
			y.value[i] = _nodeProperty{
				key:   value.Key,
				kind:  value.Kind,
				value: parseExpression(value.Value),
			}
		}
		return y

	case *ast.RegExpLiteral:
		return &_nodeRegExpLiteral{
			flags:   x.Flags,
			pattern: x.Pattern,
		}

	case *ast.SequenceExpression:
		y := &_nodeSequenceExpression{
			sequence: make([]_nodeExpression, len(x.Sequence)),
		}
		for i, value := range x.Sequence {
			y.sequence[i] = parseExpression(value)
		}
		return y

	case *ast.StringLiteral:
		return &_nodeLiteral{
			value: toValue_string(x.Value),
		}

	case *ast.ThisExpression:
		return &_nodeThisExpression{}

	case *ast.UnaryExpression:
		return &_nodeUnaryExpression{
			operator: x.Operator,
			operand:  parseExpression(x.Operand),
			postfix:  x.Postfix,
		}

	case *ast.VariableExpression:
		return &_nodeVariableExpression{
			name:        x.Name,
			initializer: parseExpression(x.Initializer),
		}

	}

	panic(fmt.Errorf("Here be dragons: parseExpression(%T)", x))
}

func parseStatement(x ast.Statement) _nodeStatement {
	if x == nil {
		return nil
	}

	switch x := x.(type) {

	case *ast.BlockStatement:
		y := &_nodeBlockStatement{
			list: make([]_nodeStatement, len(x.List)),
		}
		for i, value := range x.List {
			y.list[i] = parseStatement(value)
		}
		return y

	case *ast.BranchStatement:
		y := &_nodeBranchStatement{
			branch: x.Token,
		}
		if x.Label != nil {
			y.label = x.Label.Name
		}
		return y

	case *ast.DebuggerStatement:
		return &_nodeDebuggerStatement{}

	case *ast.DoWhileStatement:
		y := &_nodeDoWhileStatement{
			test: parseExpression(x.Test),
		}
		body := parseStatement(x.Body)
		if block, ok := body.(*_nodeBlockStatement); ok {
			y.body = block.list
		} else {
			y.body = append(y.body, body)
		}
		return y

	case *ast.EmptyStatement:
		return emptyStatement

	case *ast.ExpressionStatement:
		return &_nodeExpressionStatement{
			expression: parseExpression(x.Expression),
		}

	case *ast.ForInStatement:
		y := &_nodeForInStatement{
			into:   parseExpression(x.Into),
			source: parseExpression(x.Source),
		}
		body := parseStatement(x.Body)
		if block, ok := body.(*_nodeBlockStatement); ok {
			y.body = block.list
		} else {
			y.body = append(y.body, body)
		}
		return y

	case *ast.ForStatement:
		y := &_nodeForStatement{
			initializer: parseExpression(x.Initializer),
			update:      parseExpression(x.Update),
			test:        parseExpression(x.Test),
		}
		body := parseStatement(x.Body)
		if block, ok := body.(*_nodeBlockStatement); ok {
			y.body = block.list
		} else {
			y.body = append(y.body, body)
		}
		return y

	case *ast.IfStatement:
		return &_nodeIfStatement{
			test:       parseExpression(x.Test),
			consequent: parseStatement(x.Consequent),
			alternate:  parseStatement(x.Alternate),
		}

	case *ast.LabelledStatement:
		return &_nodeLabelledStatement{
			label:     x.Label.Name,
			statement: parseStatement(x.Statement),
		}

	case *ast.ReturnStatement:
		return &_nodeReturnStatement{
			argument: parseExpression(x.Argument),
		}

	case *ast.SwitchStatement:
		y := &_nodeSwitchStatement{
			discriminant: parseExpression(x.Discriminant),
			default_:     x.Default,
			body:         make([]*_nodeCaseStatement, len(x.Body)),
		}
		for i, p := range x.Body {
			q := &_nodeCaseStatement{
				test:       parseExpression(p.Test),
				consequent: make([]_nodeStatement, len(p.Consequent)),
			}
			for j, value := range p.Consequent {
				q.consequent[j] = parseStatement(value)
			}
			y.body[i] = q
		}
		return y

	case *ast.ThrowStatement:
		return &_nodeThrowStatement{
			argument: parseExpression(x.Argument),
		}

	case *ast.TryStatement:
		y := &_nodeTryStatement{
			body:    parseStatement(x.Body),
			finally: parseStatement(x.Finally),
		}
		if x.Catch != nil {
			y.catch = &_nodeCatchStatement{
				parameter: x.Catch.Parameter.Name,
				body:      parseStatement(x.Catch.Body),
			}
		}
		return y

	case *ast.VariableStatement:
		y := &_nodeVariableStatement{
			list: make([]_nodeExpression, len(x.List)),
		}
		for i, value := range x.List {
			y.list[i] = parseExpression(value)
		}
		return y

	case *ast.WhileStatement:
		y := &_nodeWhileStatement{
			test: parseExpression(x.Test),
		}
		body := parseStatement(x.Body)
		if block, ok := body.(*_nodeBlockStatement); ok {
			y.body = block.list
		} else {
			y.body = append(y.body, body)
		}
		return y

	case *ast.WithStatement:
		return &_nodeWithStatement{
			object: parseExpression(x.Object),
			body:   parseStatement(x.Body),
		}

	}

	panic(fmt.Errorf("Here be dragons: parseStatement(%T)", x))
}

func cmpl_parse(x *ast.Program) *_nodeProgram {
	y := &_nodeProgram{
		body: make([]_nodeStatement, len(x.Body)),
	}
	for i, value := range x.Body {
		y.body[i] = parseStatement(value)
	}
	for _, value := range x.DeclarationList {
		switch value := value.(type) {
		case *ast.FunctionDeclaration:
			y.functionList = append(y.functionList, parseExpression(value.Function).(*_nodeFunctionLiteral))
		case *ast.VariableDeclaration:
			for _, value := range value.List {
				y.varList = append(y.varList, value.Name)
			}
		default:
			panic(fmt.Errorf("Here be dragons: parseProgram.DeclarationList(%T)", value))
		}
	}
	return y
}

type _nodeProgram struct {
	body []_nodeStatement

	varList      []string
	functionList []*_nodeFunctionLiteral

	variableList []_nodeDeclaration
}

type _nodeDeclaration struct {
	name       string
	definition _node
}

type _node interface {
}

type (
	_nodeExpression interface {
		_node
		_expressionNode()
	}

	_nodeArrayLiteral struct {
		value []_nodeExpression
	}

	_nodeAssignExpression struct {
		operator token.Token
		left     _nodeExpression
		right    _nodeExpression
	}

	_nodeBinaryExpression struct {
		operator   token.Token
		left       _nodeExpression
		right      _nodeExpression
		comparison bool
	}

	_nodeBracketExpression struct {
		left   _nodeExpression
		member _nodeExpression
	}

	_nodeCallExpression struct {
		callee       _nodeExpression
		argumentList []_nodeExpression
	}

	_nodeConditionalExpression struct {
		test       _nodeExpression
		consequent _nodeExpression
		alternate  _nodeExpression
	}

	_nodeDotExpression struct {
		left       _nodeExpression
		identifier string
	}

	_nodeFunctionLiteral struct {
		name          string
		body          _nodeStatement
		source        string
		parameterList []string
		varList       []string
		functionList  []*_nodeFunctionLiteral
	}

	_nodeIdentifier struct {
		name string
	}

	_nodeLiteral struct {
		value Value
	}

	_nodeNewExpression struct {
		callee       _nodeExpression
		argumentList []_nodeExpression
	}

	_nodeObjectLiteral struct {
		value []_nodeProperty
	}

	_nodeProperty struct {
		key   string
		kind  string
		value _nodeExpression
	}

	_nodeRegExpLiteral struct {
		flags   string
		pattern string // Value?
		regexp  *regexp.Regexp
	}

	_nodeSequenceExpression struct {
		sequence []_nodeExpression
	}

	_nodeThisExpression struct {
	}

	_nodeUnaryExpression struct {
		operator token.Token
		operand  _nodeExpression
		postfix  bool
	}

	_nodeVariableExpression struct {
		name        string
		initializer _nodeExpression
	}
)

type (
	_nodeStatement interface {
		_node
		_statementNode()
	}

	_nodeBlockStatement struct {
		list []_nodeStatement
	}

	_nodeBranchStatement struct {
		branch token.Token
		label  string
	}

	_nodeCaseStatement struct {
		test       _nodeExpression
		consequent []_nodeStatement
	}

	_nodeCatchStatement struct {
		parameter string
		body      _nodeStatement
	}

	_nodeDebuggerStatement struct {
	}

	_nodeDoWhileStatement struct {
		test _nodeExpression
		body []_nodeStatement
	}

	_nodeEmptyStatement struct {
	}

	_nodeExpressionStatement struct {
		expression _nodeExpression
	}

	_nodeForInStatement struct {
		into   _nodeExpression
		source _nodeExpression
		body   []_nodeStatement
	}

	_nodeForStatement struct {
		initializer _nodeExpression
		update      _nodeExpression
		test        _nodeExpression
		body        []_nodeStatement
	}

	_nodeIfStatement struct {
		test       _nodeExpression
		consequent _nodeStatement
		alternate  _nodeStatement
	}

	_nodeLabelledStatement struct {
		label     string
		statement _nodeStatement
	}

	_nodeReturnStatement struct {
		argument _nodeExpression
	}

	_nodeSwitchStatement struct {
		discriminant _nodeExpression
		default_     int
		body         []*_nodeCaseStatement
	}

	_nodeThrowStatement struct {
		argument _nodeExpression
	}

	_nodeTryStatement struct {
		body    _nodeStatement
		catch   *_nodeCatchStatement
		finally _nodeStatement
	}

	_nodeVariableStatement struct {
		list []_nodeExpression
	}

	_nodeWhileStatement struct {
		test _nodeExpression
		body []_nodeStatement
	}

	_nodeWithStatement struct {
		object _nodeExpression
		body   _nodeStatement
	}
)

// _expressionNode

func (*_nodeArrayLiteral) _expressionNode()          {}
func (*_nodeAssignExpression) _expressionNode()      {}
func (*_nodeBinaryExpression) _expressionNode()      {}
func (*_nodeBracketExpression) _expressionNode()     {}
func (*_nodeCallExpression) _expressionNode()        {}
func (*_nodeConditionalExpression) _expressionNode() {}
func (*_nodeDotExpression) _expressionNode()         {}
func (*_nodeFunctionLiteral) _expressionNode()       {}
func (*_nodeIdentifier) _expressionNode()            {}
func (*_nodeLiteral) _expressionNode()               {}
func (*_nodeNewExpression) _expressionNode()         {}
func (*_nodeObjectLiteral) _expressionNode()         {}
func (*_nodeRegExpLiteral) _expressionNode()         {}
func (*_nodeSequenceExpression) _expressionNode()    {}
func (*_nodeThisExpression) _expressionNode()        {}
func (*_nodeUnaryExpression) _expressionNode()       {}
func (*_nodeVariableExpression) _expressionNode()    {}

// _statementNode

func (*_nodeBlockStatement) _statementNode()      {}
func (*_nodeBranchStatement) _statementNode()     {}
func (*_nodeCaseStatement) _statementNode()       {}
func (*_nodeCatchStatement) _statementNode()      {}
func (*_nodeDebuggerStatement) _statementNode()   {}
func (*_nodeDoWhileStatement) _statementNode()    {}
func (*_nodeEmptyStatement) _statementNode()      {}
func (*_nodeExpressionStatement) _statementNode() {}
func (*_nodeForInStatement) _statementNode()      {}
func (*_nodeForStatement) _statementNode()        {}
func (*_nodeIfStatement) _statementNode()         {}
func (*_nodeLabelledStatement) _statementNode()   {}
func (*_nodeReturnStatement) _statementNode()     {}
func (*_nodeSwitchStatement) _statementNode()     {}
func (*_nodeThrowStatement) _statementNode()      {}
func (*_nodeTryStatement) _statementNode()        {}
func (*_nodeVariableStatement) _statementNode()   {}
func (*_nodeWhileStatement) _statementNode()      {}
func (*_nodeWithStatement) _statementNode()       {}
