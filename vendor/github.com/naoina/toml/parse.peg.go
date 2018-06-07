package toml

import (
	"fmt"
	"math"
	"sort"
	"strconv"
)

const endSymbol rune = 1114112

/* The rule types inferred from the grammar are below. */
type pegRule uint8

const (
	ruleUnknown pegRule = iota
	ruleTOML
	ruleExpression
	rulenewline
	rulews
	rulewsnl
	rulecomment
	rulekeyval
	rulekey
	rulebareKey
	rulequotedKey
	ruleval
	ruletable
	rulestdTable
	rulearrayTable
	ruleinlineTable
	ruleinlineTableKeyValues
	ruletableKey
	ruletableKeyComp
	ruletableKeySep
	ruleinlineTableValSep
	ruleinteger
	ruleint
	rulefloat
	rulefrac
	ruleexp
	rulestring
	rulebasicString
	rulebasicChar
	ruleescaped
	rulebasicUnescaped
	ruleescape
	rulemlBasicString
	rulemlBasicBody
	ruleliteralString
	ruleliteralChar
	rulemlLiteralString
	rulemlLiteralBody
	rulemlLiteralChar
	rulehexdigit
	rulehexQuad
	ruleboolean
	ruledateFullYear
	ruledateMonth
	ruledateMDay
	ruletimeHour
	ruletimeMinute
	ruletimeSecond
	ruletimeSecfrac
	ruletimeNumoffset
	ruletimeOffset
	rulepartialTime
	rulefullDate
	rulefullTime
	ruledatetime
	ruledigit
	ruledigitDual
	ruledigitQuad
	rulearray
	rulearrayValues
	rulearraySep
	ruleAction0
	rulePegText
	ruleAction1
	ruleAction2
	ruleAction3
	ruleAction4
	ruleAction5
	ruleAction6
	ruleAction7
	ruleAction8
	ruleAction9
	ruleAction10
	ruleAction11
	ruleAction12
	ruleAction13
	ruleAction14
	ruleAction15
	ruleAction16
	ruleAction17
	ruleAction18
	ruleAction19
	ruleAction20
	ruleAction21
	ruleAction22
	ruleAction23
	ruleAction24
	ruleAction25
)

var rul3s = [...]string{
	"Unknown",
	"TOML",
	"Expression",
	"newline",
	"ws",
	"wsnl",
	"comment",
	"keyval",
	"key",
	"bareKey",
	"quotedKey",
	"val",
	"table",
	"stdTable",
	"arrayTable",
	"inlineTable",
	"inlineTableKeyValues",
	"tableKey",
	"tableKeyComp",
	"tableKeySep",
	"inlineTableValSep",
	"integer",
	"int",
	"float",
	"frac",
	"exp",
	"string",
	"basicString",
	"basicChar",
	"escaped",
	"basicUnescaped",
	"escape",
	"mlBasicString",
	"mlBasicBody",
	"literalString",
	"literalChar",
	"mlLiteralString",
	"mlLiteralBody",
	"mlLiteralChar",
	"hexdigit",
	"hexQuad",
	"boolean",
	"dateFullYear",
	"dateMonth",
	"dateMDay",
	"timeHour",
	"timeMinute",
	"timeSecond",
	"timeSecfrac",
	"timeNumoffset",
	"timeOffset",
	"partialTime",
	"fullDate",
	"fullTime",
	"datetime",
	"digit",
	"digitDual",
	"digitQuad",
	"array",
	"arrayValues",
	"arraySep",
	"Action0",
	"PegText",
	"Action1",
	"Action2",
	"Action3",
	"Action4",
	"Action5",
	"Action6",
	"Action7",
	"Action8",
	"Action9",
	"Action10",
	"Action11",
	"Action12",
	"Action13",
	"Action14",
	"Action15",
	"Action16",
	"Action17",
	"Action18",
	"Action19",
	"Action20",
	"Action21",
	"Action22",
	"Action23",
	"Action24",
	"Action25",
}

type token32 struct {
	pegRule
	begin, end uint32
}

func (t *token32) String() string {
	return fmt.Sprintf("\x1B[34m%v\x1B[m %v %v", rul3s[t.pegRule], t.begin, t.end)
}

type node32 struct {
	token32
	up, next *node32
}

func (node *node32) print(pretty bool, buffer string) {
	var print func(node *node32, depth int)
	print = func(node *node32, depth int) {
		for node != nil {
			for c := 0; c < depth; c++ {
				fmt.Printf(" ")
			}
			rule := rul3s[node.pegRule]
			quote := strconv.Quote(string(([]rune(buffer)[node.begin:node.end])))
			if !pretty {
				fmt.Printf("%v %v\n", rule, quote)
			} else {
				fmt.Printf("\x1B[34m%v\x1B[m %v\n", rule, quote)
			}
			if node.up != nil {
				print(node.up, depth+1)
			}
			node = node.next
		}
	}
	print(node, 0)
}

func (node *node32) Print(buffer string) {
	node.print(false, buffer)
}

func (node *node32) PrettyPrint(buffer string) {
	node.print(true, buffer)
}

type tokens32 struct {
	tree []token32
}

func (t *tokens32) Trim(length uint32) {
	t.tree = t.tree[:length]
}

func (t *tokens32) Print() {
	for _, token := range t.tree {
		fmt.Println(token.String())
	}
}

func (t *tokens32) AST() *node32 {
	type element struct {
		node *node32
		down *element
	}
	tokens := t.Tokens()
	var stack *element
	for _, token := range tokens {
		if token.begin == token.end {
			continue
		}
		node := &node32{token32: token}
		for stack != nil && stack.node.begin >= token.begin && stack.node.end <= token.end {
			stack.node.next = node.up
			node.up = stack.node
			stack = stack.down
		}
		stack = &element{node: node, down: stack}
	}
	if stack != nil {
		return stack.node
	}
	return nil
}

func (t *tokens32) PrintSyntaxTree(buffer string) {
	t.AST().Print(buffer)
}

func (t *tokens32) PrettyPrintSyntaxTree(buffer string) {
	t.AST().PrettyPrint(buffer)
}

func (t *tokens32) Add(rule pegRule, begin, end, index uint32) {
	if tree := t.tree; int(index) >= len(tree) {
		expanded := make([]token32, 2*len(tree))
		copy(expanded, tree)
		t.tree = expanded
	}
	t.tree[index] = token32{
		pegRule: rule,
		begin:   begin,
		end:     end,
	}
}

func (t *tokens32) Tokens() []token32 {
	return t.tree
}

type tomlParser struct {
	toml

	Buffer string
	buffer []rune
	rules  [88]func() bool
	parse  func(rule ...int) error
	reset  func()
	Pretty bool
	tokens32
}

func (p *tomlParser) Parse(rule ...int) error {
	return p.parse(rule...)
}

func (p *tomlParser) Reset() {
	p.reset()
}

type textPosition struct {
	line, symbol int
}

type textPositionMap map[int]textPosition

func translatePositions(buffer []rune, positions []int) textPositionMap {
	length, translations, j, line, symbol := len(positions), make(textPositionMap, len(positions)), 0, 1, 0
	sort.Ints(positions)

search:
	for i, c := range buffer {
		if c == '\n' {
			line, symbol = line+1, 0
		} else {
			symbol++
		}
		if i == positions[j] {
			translations[positions[j]] = textPosition{line, symbol}
			for j++; j < length; j++ {
				if i != positions[j] {
					continue search
				}
			}
			break search
		}
	}

	return translations
}

type parseError struct {
	p   *tomlParser
	max token32
}

func (e *parseError) Error() string {
	tokens, error := []token32{e.max}, "\n"
	positions, p := make([]int, 2*len(tokens)), 0
	for _, token := range tokens {
		positions[p], p = int(token.begin), p+1
		positions[p], p = int(token.end), p+1
	}
	translations := translatePositions(e.p.buffer, positions)
	format := "parse error near %v (line %v symbol %v - line %v symbol %v):\n%v\n"
	if e.p.Pretty {
		format = "parse error near \x1B[34m%v\x1B[m (line %v symbol %v - line %v symbol %v):\n%v\n"
	}
	for _, token := range tokens {
		begin, end := int(token.begin), int(token.end)
		error += fmt.Sprintf(format,
			rul3s[token.pegRule],
			translations[begin].line, translations[begin].symbol,
			translations[end].line, translations[end].symbol,
			strconv.Quote(string(e.p.buffer[begin:end])))
	}

	return error
}

func (p *tomlParser) PrintSyntaxTree() {
	if p.Pretty {
		p.tokens32.PrettyPrintSyntaxTree(p.Buffer)
	} else {
		p.tokens32.PrintSyntaxTree(p.Buffer)
	}
}

func (p *tomlParser) Execute() {
	buffer, _buffer, text, begin, end := p.Buffer, p.buffer, "", 0, 0
	for _, token := range p.Tokens() {
		switch token.pegRule {

		case rulePegText:
			begin, end = int(token.begin), int(token.end)
			text = string(_buffer[begin:end])

		case ruleAction0:
			_ = buffer
		case ruleAction1:
			p.SetTableString(begin, end)
		case ruleAction2:
			p.AddLineCount(end - begin)
		case ruleAction3:
			p.AddLineCount(end - begin)
		case ruleAction4:
			p.AddKeyValue()
		case ruleAction5:
			p.SetKey(p.buffer, begin, end)
		case ruleAction6:
			p.SetKey(p.buffer, begin, end)
		case ruleAction7:
			p.SetTime(begin, end)
		case ruleAction8:
			p.SetFloat64(begin, end)
		case ruleAction9:
			p.SetInt64(begin, end)
		case ruleAction10:
			p.SetString(begin, end)
		case ruleAction11:
			p.SetBool(begin, end)
		case ruleAction12:
			p.SetArray(begin, end)
		case ruleAction13:
			p.SetTable(p.buffer, begin, end)
		case ruleAction14:
			p.SetArrayTable(p.buffer, begin, end)
		case ruleAction15:
			p.StartInlineTable()
		case ruleAction16:
			p.EndInlineTable()
		case ruleAction17:
			p.AddTableKey()
		case ruleAction18:
			p.SetBasicString(p.buffer, begin, end)
		case ruleAction19:
			p.SetMultilineString()
		case ruleAction20:
			p.AddMultilineBasicBody(p.buffer, begin, end)
		case ruleAction21:
			p.SetLiteralString(p.buffer, begin, end)
		case ruleAction22:
			p.SetMultilineLiteralString(p.buffer, begin, end)
		case ruleAction23:
			p.StartArray()
		case ruleAction24:
			p.AddArrayVal()
		case ruleAction25:
			p.AddArrayVal()

		}
	}
	_, _, _, _, _ = buffer, _buffer, text, begin, end
}

func (p *tomlParser) Init() {
	var (
		max                  token32
		position, tokenIndex uint32
		buffer               []rune
	)
	p.reset = func() {
		max = token32{}
		position, tokenIndex = 0, 0

		p.buffer = []rune(p.Buffer)
		if len(p.buffer) == 0 || p.buffer[len(p.buffer)-1] != endSymbol {
			p.buffer = append(p.buffer, endSymbol)
		}
		buffer = p.buffer
	}
	p.reset()

	_rules := p.rules
	tree := tokens32{tree: make([]token32, math.MaxInt16)}
	p.parse = func(rule ...int) error {
		r := 1
		if len(rule) > 0 {
			r = rule[0]
		}
		matches := p.rules[r]()
		p.tokens32 = tree
		if matches {
			p.Trim(tokenIndex)
			return nil
		}
		return &parseError{p, max}
	}

	add := func(rule pegRule, begin uint32) {
		tree.Add(rule, begin, position, tokenIndex)
		tokenIndex++
		if begin != position && position > max.end {
			max = token32{rule, begin, position}
		}
	}

	matchDot := func() bool {
		if buffer[position] != endSymbol {
			position++
			return true
		}
		return false
	}

	/*matchChar := func(c byte) bool {
		if buffer[position] == c {
			position++
			return true
		}
		return false
	}*/

	/*matchRange := func(lower byte, upper byte) bool {
		if c := buffer[position]; c >= lower && c <= upper {
			position++
			return true
		}
		return false
	}*/

	_rules = [...]func() bool{
		nil,
		/* 0 TOML <- <(Expression (newline Expression)* newline? !. Action0)> */
		func() bool {
			position0, tokenIndex0 := position, tokenIndex
			{
				position1 := position
				if !_rules[ruleExpression]() {
					goto l0
				}
			l2:
				{
					position3, tokenIndex3 := position, tokenIndex
					if !_rules[rulenewline]() {
						goto l3
					}
					if !_rules[ruleExpression]() {
						goto l3
					}
					goto l2
				l3:
					position, tokenIndex = position3, tokenIndex3
				}
				{
					position4, tokenIndex4 := position, tokenIndex
					if !_rules[rulenewline]() {
						goto l4
					}
					goto l5
				l4:
					position, tokenIndex = position4, tokenIndex4
				}
			l5:
				{
					position6, tokenIndex6 := position, tokenIndex
					if !matchDot() {
						goto l6
					}
					goto l0
				l6:
					position, tokenIndex = position6, tokenIndex6
				}
				{
					add(ruleAction0, position)
				}
				add(ruleTOML, position1)
			}
			return true
		l0:
			position, tokenIndex = position0, tokenIndex0
			return false
		},
		/* 1 Expression <- <((<(ws table ws comment? (wsnl keyval ws comment?)*)> Action1) / (ws keyval ws comment?) / (ws comment?) / ws)> */
		func() bool {
			position8, tokenIndex8 := position, tokenIndex
			{
				position9 := position
				{
					position10, tokenIndex10 := position, tokenIndex
					{
						position12 := position
						if !_rules[rulews]() {
							goto l11
						}
						{
							position13 := position
							{
								position14, tokenIndex14 := position, tokenIndex
								{
									position16 := position
									if buffer[position] != rune('[') {
										goto l15
									}
									position++
									if !_rules[rulews]() {
										goto l15
									}
									{
										position17 := position
										if !_rules[ruletableKey]() {
											goto l15
										}
										add(rulePegText, position17)
									}
									if !_rules[rulews]() {
										goto l15
									}
									if buffer[position] != rune(']') {
										goto l15
									}
									position++
									{
										add(ruleAction13, position)
									}
									add(rulestdTable, position16)
								}
								goto l14
							l15:
								position, tokenIndex = position14, tokenIndex14
								{
									position19 := position
									if buffer[position] != rune('[') {
										goto l11
									}
									position++
									if buffer[position] != rune('[') {
										goto l11
									}
									position++
									if !_rules[rulews]() {
										goto l11
									}
									{
										position20 := position
										if !_rules[ruletableKey]() {
											goto l11
										}
										add(rulePegText, position20)
									}
									if !_rules[rulews]() {
										goto l11
									}
									if buffer[position] != rune(']') {
										goto l11
									}
									position++
									if buffer[position] != rune(']') {
										goto l11
									}
									position++
									{
										add(ruleAction14, position)
									}
									add(rulearrayTable, position19)
								}
							}
						l14:
							add(ruletable, position13)
						}
						if !_rules[rulews]() {
							goto l11
						}
						{
							position22, tokenIndex22 := position, tokenIndex
							if !_rules[rulecomment]() {
								goto l22
							}
							goto l23
						l22:
							position, tokenIndex = position22, tokenIndex22
						}
					l23:
					l24:
						{
							position25, tokenIndex25 := position, tokenIndex
							if !_rules[rulewsnl]() {
								goto l25
							}
							if !_rules[rulekeyval]() {
								goto l25
							}
							if !_rules[rulews]() {
								goto l25
							}
							{
								position26, tokenIndex26 := position, tokenIndex
								if !_rules[rulecomment]() {
									goto l26
								}
								goto l27
							l26:
								position, tokenIndex = position26, tokenIndex26
							}
						l27:
							goto l24
						l25:
							position, tokenIndex = position25, tokenIndex25
						}
						add(rulePegText, position12)
					}
					{
						add(ruleAction1, position)
					}
					goto l10
				l11:
					position, tokenIndex = position10, tokenIndex10
					if !_rules[rulews]() {
						goto l29
					}
					if !_rules[rulekeyval]() {
						goto l29
					}
					if !_rules[rulews]() {
						goto l29
					}
					{
						position30, tokenIndex30 := position, tokenIndex
						if !_rules[rulecomment]() {
							goto l30
						}
						goto l31
					l30:
						position, tokenIndex = position30, tokenIndex30
					}
				l31:
					goto l10
				l29:
					position, tokenIndex = position10, tokenIndex10
					if !_rules[rulews]() {
						goto l32
					}
					{
						position33, tokenIndex33 := position, tokenIndex
						if !_rules[rulecomment]() {
							goto l33
						}
						goto l34
					l33:
						position, tokenIndex = position33, tokenIndex33
					}
				l34:
					goto l10
				l32:
					position, tokenIndex = position10, tokenIndex10
					if !_rules[rulews]() {
						goto l8
					}
				}
			l10:
				add(ruleExpression, position9)
			}
			return true
		l8:
			position, tokenIndex = position8, tokenIndex8
			return false
		},
		/* 2 newline <- <(<('\r' / '\n')+> Action2)> */
		func() bool {
			position35, tokenIndex35 := position, tokenIndex
			{
				position36 := position
				{
					position37 := position
					{
						position40, tokenIndex40 := position, tokenIndex
						if buffer[position] != rune('\r') {
							goto l41
						}
						position++
						goto l40
					l41:
						position, tokenIndex = position40, tokenIndex40
						if buffer[position] != rune('\n') {
							goto l35
						}
						position++
					}
				l40:
				l38:
					{
						position39, tokenIndex39 := position, tokenIndex
						{
							position42, tokenIndex42 := position, tokenIndex
							if buffer[position] != rune('\r') {
								goto l43
							}
							position++
							goto l42
						l43:
							position, tokenIndex = position42, tokenIndex42
							if buffer[position] != rune('\n') {
								goto l39
							}
							position++
						}
					l42:
						goto l38
					l39:
						position, tokenIndex = position39, tokenIndex39
					}
					add(rulePegText, position37)
				}
				{
					add(ruleAction2, position)
				}
				add(rulenewline, position36)
			}
			return true
		l35:
			position, tokenIndex = position35, tokenIndex35
			return false
		},
		/* 3 ws <- <(' ' / '\t')*> */
		func() bool {
			{
				position46 := position
			l47:
				{
					position48, tokenIndex48 := position, tokenIndex
					{
						position49, tokenIndex49 := position, tokenIndex
						if buffer[position] != rune(' ') {
							goto l50
						}
						position++
						goto l49
					l50:
						position, tokenIndex = position49, tokenIndex49
						if buffer[position] != rune('\t') {
							goto l48
						}
						position++
					}
				l49:
					goto l47
				l48:
					position, tokenIndex = position48, tokenIndex48
				}
				add(rulews, position46)
			}
			return true
		},
		/* 4 wsnl <- <((&('\t') '\t') | (&(' ') ' ') | (&('\n' | '\r') (<('\r' / '\n')> Action3)))*> */
		func() bool {
			{
				position52 := position
			l53:
				{
					position54, tokenIndex54 := position, tokenIndex
					{
						switch buffer[position] {
						case '\t':
							if buffer[position] != rune('\t') {
								goto l54
							}
							position++
							break
						case ' ':
							if buffer[position] != rune(' ') {
								goto l54
							}
							position++
							break
						default:
							{
								position56 := position
								{
									position57, tokenIndex57 := position, tokenIndex
									if buffer[position] != rune('\r') {
										goto l58
									}
									position++
									goto l57
								l58:
									position, tokenIndex = position57, tokenIndex57
									if buffer[position] != rune('\n') {
										goto l54
									}
									position++
								}
							l57:
								add(rulePegText, position56)
							}
							{
								add(ruleAction3, position)
							}
							break
						}
					}

					goto l53
				l54:
					position, tokenIndex = position54, tokenIndex54
				}
				add(rulewsnl, position52)
			}
			return true
		},
		/* 5 comment <- <('#' <('\t' / [ -\U0010ffff])*>)> */
		func() bool {
			position60, tokenIndex60 := position, tokenIndex
			{
				position61 := position
				if buffer[position] != rune('#') {
					goto l60
				}
				position++
				{
					position62 := position
				l63:
					{
						position64, tokenIndex64 := position, tokenIndex
						{
							position65, tokenIndex65 := position, tokenIndex
							if buffer[position] != rune('\t') {
								goto l66
							}
							position++
							goto l65
						l66:
							position, tokenIndex = position65, tokenIndex65
							if c := buffer[position]; c < rune(' ') || c > rune('\U0010ffff') {
								goto l64
							}
							position++
						}
					l65:
						goto l63
					l64:
						position, tokenIndex = position64, tokenIndex64
					}
					add(rulePegText, position62)
				}
				add(rulecomment, position61)
			}
			return true
		l60:
			position, tokenIndex = position60, tokenIndex60
			return false
		},
		/* 6 keyval <- <(key ws '=' ws val Action4)> */
		func() bool {
			position67, tokenIndex67 := position, tokenIndex
			{
				position68 := position
				if !_rules[rulekey]() {
					goto l67
				}
				if !_rules[rulews]() {
					goto l67
				}
				if buffer[position] != rune('=') {
					goto l67
				}
				position++
				if !_rules[rulews]() {
					goto l67
				}
				if !_rules[ruleval]() {
					goto l67
				}
				{
					add(ruleAction4, position)
				}
				add(rulekeyval, position68)
			}
			return true
		l67:
			position, tokenIndex = position67, tokenIndex67
			return false
		},
		/* 7 key <- <(bareKey / quotedKey)> */
		func() bool {
			position70, tokenIndex70 := position, tokenIndex
			{
				position71 := position
				{
					position72, tokenIndex72 := position, tokenIndex
					{
						position74 := position
						{
							position75 := position
							{
								switch buffer[position] {
								case '_':
									if buffer[position] != rune('_') {
										goto l73
									}
									position++
									break
								case '-':
									if buffer[position] != rune('-') {
										goto l73
									}
									position++
									break
								case 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z':
									if c := buffer[position]; c < rune('a') || c > rune('z') {
										goto l73
									}
									position++
									break
								case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
									if c := buffer[position]; c < rune('0') || c > rune('9') {
										goto l73
									}
									position++
									break
								default:
									if c := buffer[position]; c < rune('A') || c > rune('Z') {
										goto l73
									}
									position++
									break
								}
							}

						l76:
							{
								position77, tokenIndex77 := position, tokenIndex
								{
									switch buffer[position] {
									case '_':
										if buffer[position] != rune('_') {
											goto l77
										}
										position++
										break
									case '-':
										if buffer[position] != rune('-') {
											goto l77
										}
										position++
										break
									case 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z':
										if c := buffer[position]; c < rune('a') || c > rune('z') {
											goto l77
										}
										position++
										break
									case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
										if c := buffer[position]; c < rune('0') || c > rune('9') {
											goto l77
										}
										position++
										break
									default:
										if c := buffer[position]; c < rune('A') || c > rune('Z') {
											goto l77
										}
										position++
										break
									}
								}

								goto l76
							l77:
								position, tokenIndex = position77, tokenIndex77
							}
							add(rulePegText, position75)
						}
						{
							add(ruleAction5, position)
						}
						add(rulebareKey, position74)
					}
					goto l72
				l73:
					position, tokenIndex = position72, tokenIndex72
					{
						position81 := position
						{
							position82 := position
							if buffer[position] != rune('"') {
								goto l70
							}
							position++
						l83:
							{
								position84, tokenIndex84 := position, tokenIndex
								if !_rules[rulebasicChar]() {
									goto l84
								}
								goto l83
							l84:
								position, tokenIndex = position84, tokenIndex84
							}
							if buffer[position] != rune('"') {
								goto l70
							}
							position++
							add(rulePegText, position82)
						}
						{
							add(ruleAction6, position)
						}
						add(rulequotedKey, position81)
					}
				}
			l72:
				add(rulekey, position71)
			}
			return true
		l70:
			position, tokenIndex = position70, tokenIndex70
			return false
		},
		/* 8 bareKey <- <(<((&('_') '_') | (&('-') '-') | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]) | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]))+> Action5)> */
		nil,
		/* 9 quotedKey <- <(<('"' basicChar* '"')> Action6)> */
		nil,
		/* 10 val <- <((<datetime> Action7) / (<float> Action8) / ((&('{') inlineTable) | (&('[') (<array> Action12)) | (&('f' | 't') (<boolean> Action11)) | (&('"' | '\'') (<string> Action10)) | (&('+' | '-' | '0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') (<integer> Action9))))> */
		func() bool {
			position88, tokenIndex88 := position, tokenIndex
			{
				position89 := position
				{
					position90, tokenIndex90 := position, tokenIndex
					{
						position92 := position
						{
							position93 := position
							{
								position94, tokenIndex94 := position, tokenIndex
								{
									position96 := position
									{
										position97 := position
										{
											position98 := position
											if !_rules[ruledigitDual]() {
												goto l95
											}
											if !_rules[ruledigitDual]() {
												goto l95
											}
											add(ruledigitQuad, position98)
										}
										add(ruledateFullYear, position97)
									}
									if buffer[position] != rune('-') {
										goto l95
									}
									position++
									{
										position99 := position
										if !_rules[ruledigitDual]() {
											goto l95
										}
										add(ruledateMonth, position99)
									}
									if buffer[position] != rune('-') {
										goto l95
									}
									position++
									{
										position100 := position
										if !_rules[ruledigitDual]() {
											goto l95
										}
										add(ruledateMDay, position100)
									}
									add(rulefullDate, position96)
								}
								{
									position101, tokenIndex101 := position, tokenIndex
									if buffer[position] != rune('T') {
										goto l101
									}
									position++
									{
										position103 := position
										if !_rules[rulepartialTime]() {
											goto l101
										}
										{
											position104, tokenIndex104 := position, tokenIndex
											{
												position106 := position
												{
													position107, tokenIndex107 := position, tokenIndex
													if buffer[position] != rune('Z') {
														goto l108
													}
													position++
													goto l107
												l108:
													position, tokenIndex = position107, tokenIndex107
													{
														position109 := position
														{
															position110, tokenIndex110 := position, tokenIndex
															if buffer[position] != rune('-') {
																goto l111
															}
															position++
															goto l110
														l111:
															position, tokenIndex = position110, tokenIndex110
															if buffer[position] != rune('+') {
																goto l104
															}
															position++
														}
													l110:
														if !_rules[ruletimeHour]() {
															goto l104
														}
														if buffer[position] != rune(':') {
															goto l104
														}
														position++
														if !_rules[ruletimeMinute]() {
															goto l104
														}
														add(ruletimeNumoffset, position109)
													}
												}
											l107:
												add(ruletimeOffset, position106)
											}
											goto l105
										l104:
											position, tokenIndex = position104, tokenIndex104
										}
									l105:
										add(rulefullTime, position103)
									}
									goto l102
								l101:
									position, tokenIndex = position101, tokenIndex101
								}
							l102:
								goto l94
							l95:
								position, tokenIndex = position94, tokenIndex94
								if !_rules[rulepartialTime]() {
									goto l91
								}
							}
						l94:
							add(ruledatetime, position93)
						}
						add(rulePegText, position92)
					}
					{
						add(ruleAction7, position)
					}
					goto l90
				l91:
					position, tokenIndex = position90, tokenIndex90
					{
						position114 := position
						{
							position115 := position
							if !_rules[ruleinteger]() {
								goto l113
							}
							{
								position116, tokenIndex116 := position, tokenIndex
								if !_rules[rulefrac]() {
									goto l117
								}
								{
									position118, tokenIndex118 := position, tokenIndex
									if !_rules[ruleexp]() {
										goto l118
									}
									goto l119
								l118:
									position, tokenIndex = position118, tokenIndex118
								}
							l119:
								goto l116
							l117:
								position, tokenIndex = position116, tokenIndex116
								{
									position120, tokenIndex120 := position, tokenIndex
									if !_rules[rulefrac]() {
										goto l120
									}
									goto l121
								l120:
									position, tokenIndex = position120, tokenIndex120
								}
							l121:
								if !_rules[ruleexp]() {
									goto l113
								}
							}
						l116:
							add(rulefloat, position115)
						}
						add(rulePegText, position114)
					}
					{
						add(ruleAction8, position)
					}
					goto l90
				l113:
					position, tokenIndex = position90, tokenIndex90
					{
						switch buffer[position] {
						case '{':
							{
								position124 := position
								if buffer[position] != rune('{') {
									goto l88
								}
								position++
								{
									add(ruleAction15, position)
								}
								if !_rules[rulews]() {
									goto l88
								}
								{
									position126 := position
								l127:
									{
										position128, tokenIndex128 := position, tokenIndex
										if !_rules[rulekeyval]() {
											goto l128
										}
										{
											position129, tokenIndex129 := position, tokenIndex
											{
												position131 := position
												if !_rules[rulews]() {
													goto l129
												}
												if buffer[position] != rune(',') {
													goto l129
												}
												position++
												if !_rules[rulews]() {
													goto l129
												}
												add(ruleinlineTableValSep, position131)
											}
											goto l130
										l129:
											position, tokenIndex = position129, tokenIndex129
										}
									l130:
										goto l127
									l128:
										position, tokenIndex = position128, tokenIndex128
									}
									add(ruleinlineTableKeyValues, position126)
								}
								if !_rules[rulews]() {
									goto l88
								}
								if buffer[position] != rune('}') {
									goto l88
								}
								position++
								{
									add(ruleAction16, position)
								}
								add(ruleinlineTable, position124)
							}
							break
						case '[':
							{
								position133 := position
								{
									position134 := position
									if buffer[position] != rune('[') {
										goto l88
									}
									position++
									{
										add(ruleAction23, position)
									}
									if !_rules[rulewsnl]() {
										goto l88
									}
									{
										position136, tokenIndex136 := position, tokenIndex
										{
											position138 := position
											if !_rules[ruleval]() {
												goto l136
											}
											{
												add(ruleAction24, position)
											}
										l140:
											{
												position141, tokenIndex141 := position, tokenIndex
												if !_rules[rulewsnl]() {
													goto l141
												}
												{
													position142, tokenIndex142 := position, tokenIndex
													if !_rules[rulecomment]() {
														goto l142
													}
													goto l143
												l142:
													position, tokenIndex = position142, tokenIndex142
												}
											l143:
												if !_rules[rulewsnl]() {
													goto l141
												}
												if !_rules[rulearraySep]() {
													goto l141
												}
												if !_rules[rulewsnl]() {
													goto l141
												}
												{
													position144, tokenIndex144 := position, tokenIndex
													if !_rules[rulecomment]() {
														goto l144
													}
													goto l145
												l144:
													position, tokenIndex = position144, tokenIndex144
												}
											l145:
												if !_rules[rulewsnl]() {
													goto l141
												}
												if !_rules[ruleval]() {
													goto l141
												}
												{
													add(ruleAction25, position)
												}
												goto l140
											l141:
												position, tokenIndex = position141, tokenIndex141
											}
											if !_rules[rulewsnl]() {
												goto l136
											}
											{
												position147, tokenIndex147 := position, tokenIndex
												if !_rules[rulearraySep]() {
													goto l147
												}
												goto l148
											l147:
												position, tokenIndex = position147, tokenIndex147
											}
										l148:
											if !_rules[rulewsnl]() {
												goto l136
											}
											{
												position149, tokenIndex149 := position, tokenIndex
												if !_rules[rulecomment]() {
													goto l149
												}
												goto l150
											l149:
												position, tokenIndex = position149, tokenIndex149
											}
										l150:
											add(rulearrayValues, position138)
										}
										goto l137
									l136:
										position, tokenIndex = position136, tokenIndex136
									}
								l137:
									if !_rules[rulewsnl]() {
										goto l88
									}
									if buffer[position] != rune(']') {
										goto l88
									}
									position++
									add(rulearray, position134)
								}
								add(rulePegText, position133)
							}
							{
								add(ruleAction12, position)
							}
							break
						case 'f', 't':
							{
								position152 := position
								{
									position153 := position
									{
										position154, tokenIndex154 := position, tokenIndex
										if buffer[position] != rune('t') {
											goto l155
										}
										position++
										if buffer[position] != rune('r') {
											goto l155
										}
										position++
										if buffer[position] != rune('u') {
											goto l155
										}
										position++
										if buffer[position] != rune('e') {
											goto l155
										}
										position++
										goto l154
									l155:
										position, tokenIndex = position154, tokenIndex154
										if buffer[position] != rune('f') {
											goto l88
										}
										position++
										if buffer[position] != rune('a') {
											goto l88
										}
										position++
										if buffer[position] != rune('l') {
											goto l88
										}
										position++
										if buffer[position] != rune('s') {
											goto l88
										}
										position++
										if buffer[position] != rune('e') {
											goto l88
										}
										position++
									}
								l154:
									add(ruleboolean, position153)
								}
								add(rulePegText, position152)
							}
							{
								add(ruleAction11, position)
							}
							break
						case '"', '\'':
							{
								position157 := position
								{
									position158 := position
									{
										position159, tokenIndex159 := position, tokenIndex
										{
											position161 := position
											if buffer[position] != rune('\'') {
												goto l160
											}
											position++
											if buffer[position] != rune('\'') {
												goto l160
											}
											position++
											if buffer[position] != rune('\'') {
												goto l160
											}
											position++
											{
												position162 := position
												{
													position163 := position
												l164:
													{
														position165, tokenIndex165 := position, tokenIndex
														{
															position166, tokenIndex166 := position, tokenIndex
															if buffer[position] != rune('\'') {
																goto l166
															}
															position++
															if buffer[position] != rune('\'') {
																goto l166
															}
															position++
															if buffer[position] != rune('\'') {
																goto l166
															}
															position++
															goto l165
														l166:
															position, tokenIndex = position166, tokenIndex166
														}
														{
															position167, tokenIndex167 := position, tokenIndex
															{
																position169 := position
																{
																	position170, tokenIndex170 := position, tokenIndex
																	if buffer[position] != rune('\t') {
																		goto l171
																	}
																	position++
																	goto l170
																l171:
																	position, tokenIndex = position170, tokenIndex170
																	if c := buffer[position]; c < rune(' ') || c > rune('\U0010ffff') {
																		goto l168
																	}
																	position++
																}
															l170:
																add(rulemlLiteralChar, position169)
															}
															goto l167
														l168:
															position, tokenIndex = position167, tokenIndex167
															if !_rules[rulenewline]() {
																goto l165
															}
														}
													l167:
														goto l164
													l165:
														position, tokenIndex = position165, tokenIndex165
													}
													add(rulemlLiteralBody, position163)
												}
												add(rulePegText, position162)
											}
											if buffer[position] != rune('\'') {
												goto l160
											}
											position++
											if buffer[position] != rune('\'') {
												goto l160
											}
											position++
											if buffer[position] != rune('\'') {
												goto l160
											}
											position++
											{
												add(ruleAction22, position)
											}
											add(rulemlLiteralString, position161)
										}
										goto l159
									l160:
										position, tokenIndex = position159, tokenIndex159
										{
											position174 := position
											if buffer[position] != rune('\'') {
												goto l173
											}
											position++
											{
												position175 := position
											l176:
												{
													position177, tokenIndex177 := position, tokenIndex
													{
														position178 := position
														{
															switch buffer[position] {
															case '\t':
																if buffer[position] != rune('\t') {
																	goto l177
																}
																position++
																break
															case ' ', '!', '"', '#', '$', '%', '&':
																if c := buffer[position]; c < rune(' ') || c > rune('&') {
																	goto l177
																}
																position++
																break
															default:
																if c := buffer[position]; c < rune('(') || c > rune('\U0010ffff') {
																	goto l177
																}
																position++
																break
															}
														}

														add(ruleliteralChar, position178)
													}
													goto l176
												l177:
													position, tokenIndex = position177, tokenIndex177
												}
												add(rulePegText, position175)
											}
											if buffer[position] != rune('\'') {
												goto l173
											}
											position++
											{
												add(ruleAction21, position)
											}
											add(ruleliteralString, position174)
										}
										goto l159
									l173:
										position, tokenIndex = position159, tokenIndex159
										{
											position182 := position
											if buffer[position] != rune('"') {
												goto l181
											}
											position++
											if buffer[position] != rune('"') {
												goto l181
											}
											position++
											if buffer[position] != rune('"') {
												goto l181
											}
											position++
											{
												position183 := position
											l184:
												{
													position185, tokenIndex185 := position, tokenIndex
													{
														position186, tokenIndex186 := position, tokenIndex
														{
															position188 := position
															{
																position189, tokenIndex189 := position, tokenIndex
																if !_rules[rulebasicChar]() {
																	goto l190
																}
																goto l189
															l190:
																position, tokenIndex = position189, tokenIndex189
																if !_rules[rulenewline]() {
																	goto l187
																}
															}
														l189:
															add(rulePegText, position188)
														}
														{
															add(ruleAction20, position)
														}
														goto l186
													l187:
														position, tokenIndex = position186, tokenIndex186
														if !_rules[ruleescape]() {
															goto l185
														}
														if !_rules[rulenewline]() {
															goto l185
														}
														if !_rules[rulewsnl]() {
															goto l185
														}
													}
												l186:
													goto l184
												l185:
													position, tokenIndex = position185, tokenIndex185
												}
												add(rulemlBasicBody, position183)
											}
											if buffer[position] != rune('"') {
												goto l181
											}
											position++
											if buffer[position] != rune('"') {
												goto l181
											}
											position++
											if buffer[position] != rune('"') {
												goto l181
											}
											position++
											{
												add(ruleAction19, position)
											}
											add(rulemlBasicString, position182)
										}
										goto l159
									l181:
										position, tokenIndex = position159, tokenIndex159
										{
											position193 := position
											{
												position194 := position
												if buffer[position] != rune('"') {
													goto l88
												}
												position++
											l195:
												{
													position196, tokenIndex196 := position, tokenIndex
													if !_rules[rulebasicChar]() {
														goto l196
													}
													goto l195
												l196:
													position, tokenIndex = position196, tokenIndex196
												}
												if buffer[position] != rune('"') {
													goto l88
												}
												position++
												add(rulePegText, position194)
											}
											{
												add(ruleAction18, position)
											}
											add(rulebasicString, position193)
										}
									}
								l159:
									add(rulestring, position158)
								}
								add(rulePegText, position157)
							}
							{
								add(ruleAction10, position)
							}
							break
						default:
							{
								position199 := position
								if !_rules[ruleinteger]() {
									goto l88
								}
								add(rulePegText, position199)
							}
							{
								add(ruleAction9, position)
							}
							break
						}
					}

				}
			l90:
				add(ruleval, position89)
			}
			return true
		l88:
			position, tokenIndex = position88, tokenIndex88
			return false
		},
		/* 11 table <- <(stdTable / arrayTable)> */
		nil,
		/* 12 stdTable <- <('[' ws <tableKey> ws ']' Action13)> */
		nil,
		/* 13 arrayTable <- <('[' '[' ws <tableKey> ws (']' ']') Action14)> */
		nil,
		/* 14 inlineTable <- <('{' Action15 ws inlineTableKeyValues ws '}' Action16)> */
		nil,
		/* 15 inlineTableKeyValues <- <(keyval inlineTableValSep?)*> */
		nil,
		/* 16 tableKey <- <(tableKeyComp (tableKeySep tableKeyComp)*)> */
		func() bool {
			position206, tokenIndex206 := position, tokenIndex
			{
				position207 := position
				if !_rules[ruletableKeyComp]() {
					goto l206
				}
			l208:
				{
					position209, tokenIndex209 := position, tokenIndex
					{
						position210 := position
						if !_rules[rulews]() {
							goto l209
						}
						if buffer[position] != rune('.') {
							goto l209
						}
						position++
						if !_rules[rulews]() {
							goto l209
						}
						add(ruletableKeySep, position210)
					}
					if !_rules[ruletableKeyComp]() {
						goto l209
					}
					goto l208
				l209:
					position, tokenIndex = position209, tokenIndex209
				}
				add(ruletableKey, position207)
			}
			return true
		l206:
			position, tokenIndex = position206, tokenIndex206
			return false
		},
		/* 17 tableKeyComp <- <(key Action17)> */
		func() bool {
			position211, tokenIndex211 := position, tokenIndex
			{
				position212 := position
				if !_rules[rulekey]() {
					goto l211
				}
				{
					add(ruleAction17, position)
				}
				add(ruletableKeyComp, position212)
			}
			return true
		l211:
			position, tokenIndex = position211, tokenIndex211
			return false
		},
		/* 18 tableKeySep <- <(ws '.' ws)> */
		nil,
		/* 19 inlineTableValSep <- <(ws ',' ws)> */
		nil,
		/* 20 integer <- <(('-' / '+')? int)> */
		func() bool {
			position216, tokenIndex216 := position, tokenIndex
			{
				position217 := position
				{
					position218, tokenIndex218 := position, tokenIndex
					{
						position220, tokenIndex220 := position, tokenIndex
						if buffer[position] != rune('-') {
							goto l221
						}
						position++
						goto l220
					l221:
						position, tokenIndex = position220, tokenIndex220
						if buffer[position] != rune('+') {
							goto l218
						}
						position++
					}
				l220:
					goto l219
				l218:
					position, tokenIndex = position218, tokenIndex218
				}
			l219:
				{
					position222 := position
					{
						position223, tokenIndex223 := position, tokenIndex
						if c := buffer[position]; c < rune('1') || c > rune('9') {
							goto l224
						}
						position++
						{
							position227, tokenIndex227 := position, tokenIndex
							if !_rules[ruledigit]() {
								goto l228
							}
							goto l227
						l228:
							position, tokenIndex = position227, tokenIndex227
							if buffer[position] != rune('_') {
								goto l224
							}
							position++
							if !_rules[ruledigit]() {
								goto l224
							}
						}
					l227:
					l225:
						{
							position226, tokenIndex226 := position, tokenIndex
							{
								position229, tokenIndex229 := position, tokenIndex
								if !_rules[ruledigit]() {
									goto l230
								}
								goto l229
							l230:
								position, tokenIndex = position229, tokenIndex229
								if buffer[position] != rune('_') {
									goto l226
								}
								position++
								if !_rules[ruledigit]() {
									goto l226
								}
							}
						l229:
							goto l225
						l226:
							position, tokenIndex = position226, tokenIndex226
						}
						goto l223
					l224:
						position, tokenIndex = position223, tokenIndex223
						if !_rules[ruledigit]() {
							goto l216
						}
					}
				l223:
					add(ruleint, position222)
				}
				add(ruleinteger, position217)
			}
			return true
		l216:
			position, tokenIndex = position216, tokenIndex216
			return false
		},
		/* 21 int <- <(([1-9] (digit / ('_' digit))+) / digit)> */
		nil,
		/* 22 float <- <(integer ((frac exp?) / (frac? exp)))> */
		nil,
		/* 23 frac <- <('.' digit (digit / ('_' digit))*)> */
		func() bool {
			position233, tokenIndex233 := position, tokenIndex
			{
				position234 := position
				if buffer[position] != rune('.') {
					goto l233
				}
				position++
				if !_rules[ruledigit]() {
					goto l233
				}
			l235:
				{
					position236, tokenIndex236 := position, tokenIndex
					{
						position237, tokenIndex237 := position, tokenIndex
						if !_rules[ruledigit]() {
							goto l238
						}
						goto l237
					l238:
						position, tokenIndex = position237, tokenIndex237
						if buffer[position] != rune('_') {
							goto l236
						}
						position++
						if !_rules[ruledigit]() {
							goto l236
						}
					}
				l237:
					goto l235
				l236:
					position, tokenIndex = position236, tokenIndex236
				}
				add(rulefrac, position234)
			}
			return true
		l233:
			position, tokenIndex = position233, tokenIndex233
			return false
		},
		/* 24 exp <- <(('e' / 'E') ('-' / '+')? digit (digit / ('_' digit))*)> */
		func() bool {
			position239, tokenIndex239 := position, tokenIndex
			{
				position240 := position
				{
					position241, tokenIndex241 := position, tokenIndex
					if buffer[position] != rune('e') {
						goto l242
					}
					position++
					goto l241
				l242:
					position, tokenIndex = position241, tokenIndex241
					if buffer[position] != rune('E') {
						goto l239
					}
					position++
				}
			l241:
				{
					position243, tokenIndex243 := position, tokenIndex
					{
						position245, tokenIndex245 := position, tokenIndex
						if buffer[position] != rune('-') {
							goto l246
						}
						position++
						goto l245
					l246:
						position, tokenIndex = position245, tokenIndex245
						if buffer[position] != rune('+') {
							goto l243
						}
						position++
					}
				l245:
					goto l244
				l243:
					position, tokenIndex = position243, tokenIndex243
				}
			l244:
				if !_rules[ruledigit]() {
					goto l239
				}
			l247:
				{
					position248, tokenIndex248 := position, tokenIndex
					{
						position249, tokenIndex249 := position, tokenIndex
						if !_rules[ruledigit]() {
							goto l250
						}
						goto l249
					l250:
						position, tokenIndex = position249, tokenIndex249
						if buffer[position] != rune('_') {
							goto l248
						}
						position++
						if !_rules[ruledigit]() {
							goto l248
						}
					}
				l249:
					goto l247
				l248:
					position, tokenIndex = position248, tokenIndex248
				}
				add(ruleexp, position240)
			}
			return true
		l239:
			position, tokenIndex = position239, tokenIndex239
			return false
		},
		/* 25 string <- <(mlLiteralString / literalString / mlBasicString / basicString)> */
		nil,
		/* 26 basicString <- <(<('"' basicChar* '"')> Action18)> */
		nil,
		/* 27 basicChar <- <(basicUnescaped / escaped)> */
		func() bool {
			position253, tokenIndex253 := position, tokenIndex
			{
				position254 := position
				{
					position255, tokenIndex255 := position, tokenIndex
					{
						position257 := position
						{
							switch buffer[position] {
							case ' ', '!':
								if c := buffer[position]; c < rune(' ') || c > rune('!') {
									goto l256
								}
								position++
								break
							case '#', '$', '%', '&', '\'', '(', ')', '*', '+', ',', '-', '.', '/', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', ':', ';', '<', '=', '>', '?', '@', 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z', '[':
								if c := buffer[position]; c < rune('#') || c > rune('[') {
									goto l256
								}
								position++
								break
							default:
								if c := buffer[position]; c < rune(']') || c > rune('\U0010ffff') {
									goto l256
								}
								position++
								break
							}
						}

						add(rulebasicUnescaped, position257)
					}
					goto l255
				l256:
					position, tokenIndex = position255, tokenIndex255
					{
						position259 := position
						if !_rules[ruleescape]() {
							goto l253
						}
						{
							switch buffer[position] {
							case 'U':
								if buffer[position] != rune('U') {
									goto l253
								}
								position++
								if !_rules[rulehexQuad]() {
									goto l253
								}
								if !_rules[rulehexQuad]() {
									goto l253
								}
								break
							case 'u':
								if buffer[position] != rune('u') {
									goto l253
								}
								position++
								if !_rules[rulehexQuad]() {
									goto l253
								}
								break
							case '\\':
								if buffer[position] != rune('\\') {
									goto l253
								}
								position++
								break
							case '/':
								if buffer[position] != rune('/') {
									goto l253
								}
								position++
								break
							case '"':
								if buffer[position] != rune('"') {
									goto l253
								}
								position++
								break
							case 'r':
								if buffer[position] != rune('r') {
									goto l253
								}
								position++
								break
							case 'f':
								if buffer[position] != rune('f') {
									goto l253
								}
								position++
								break
							case 'n':
								if buffer[position] != rune('n') {
									goto l253
								}
								position++
								break
							case 't':
								if buffer[position] != rune('t') {
									goto l253
								}
								position++
								break
							default:
								if buffer[position] != rune('b') {
									goto l253
								}
								position++
								break
							}
						}

						add(ruleescaped, position259)
					}
				}
			l255:
				add(rulebasicChar, position254)
			}
			return true
		l253:
			position, tokenIndex = position253, tokenIndex253
			return false
		},
		/* 28 escaped <- <(escape ((&('U') ('U' hexQuad hexQuad)) | (&('u') ('u' hexQuad)) | (&('\\') '\\') | (&('/') '/') | (&('"') '"') | (&('r') 'r') | (&('f') 'f') | (&('n') 'n') | (&('t') 't') | (&('b') 'b')))> */
		nil,
		/* 29 basicUnescaped <- <((&(' ' | '!') [ -!]) | (&('#' | '$' | '%' | '&' | '\'' | '(' | ')' | '*' | '+' | ',' | '-' | '.' | '/' | '0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9' | ':' | ';' | '<' | '=' | '>' | '?' | '@' | 'A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z' | '[') [#-[]) | (&(']' | '^' | '_' | '`' | 'a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z' | '{' | '|' | '}' | '~' | '\u007f' | '\u0080' | '\u0081' | '\u0082' | '\u0083' | '\u0084' | '\u0085' | '\u0086' | '\u0087' | '\u0088' | '\u0089' | '\u008a' | '\u008b' | '\u008c' | '\u008d' | '\u008e' | '\u008f' | '\u0090' | '\u0091' | '\u0092' | '\u0093' | '\u0094' | '\u0095' | '\u0096' | '\u0097' | '\u0098' | '\u0099' | '\u009a' | '\u009b' | '\u009c' | '\u009d' | '\u009e' | '\u009f' | '\u00a0' | '¡' | '¢' | '£' | '¤' | '¥' | '¦' | '§' | '¨' | '©' | 'ª' | '«' | '¬' | '\u00ad' | '®' | '¯' | '°' | '±' | '²' | '³' | '´' | 'µ' | '¶' | '·' | '¸' | '¹' | 'º' | '»' | '¼' | '½' | '¾' | '¿' | 'À' | 'Á' | 'Â' | 'Ã' | 'Ä' | 'Å' | 'Æ' | 'Ç' | 'È' | 'É' | 'Ê' | 'Ë' | 'Ì' | 'Í' | 'Î' | 'Ï' | 'Ð' | 'Ñ' | 'Ò' | 'Ó' | 'Ô' | 'Õ' | 'Ö' | '×' | 'Ø' | 'Ù' | 'Ú' | 'Û' | 'Ü' | 'Ý' | 'Þ' | 'ß' | 'à' | 'á' | 'â' | 'ã' | 'ä' | 'å' | 'æ' | 'ç' | 'è' | 'é' | 'ê' | 'ë' | 'ì' | 'í' | 'î' | 'ï' | 'ð' | 'ñ' | 'ò' | 'ó' | 'ô' | 'õ' | 'ö' | '÷' | 'ø' | 'ù' | 'ú' | 'û' | 'ü' | 'ý' | 'þ' | 'ÿ') []-\U0010ffff]))> */
		nil,
		/* 30 escape <- <'\\'> */
		func() bool {
			position263, tokenIndex263 := position, tokenIndex
			{
				position264 := position
				if buffer[position] != rune('\\') {
					goto l263
				}
				position++
				add(ruleescape, position264)
			}
			return true
		l263:
			position, tokenIndex = position263, tokenIndex263
			return false
		},
		/* 31 mlBasicString <- <('"' '"' '"' mlBasicBody ('"' '"' '"') Action19)> */
		nil,
		/* 32 mlBasicBody <- <((<(basicChar / newline)> Action20) / (escape newline wsnl))*> */
		nil,
		/* 33 literalString <- <('\'' <literalChar*> '\'' Action21)> */
		nil,
		/* 34 literalChar <- <((&('\t') '\t') | (&(' ' | '!' | '"' | '#' | '$' | '%' | '&') [ -&]) | (&('(' | ')' | '*' | '+' | ',' | '-' | '.' | '/' | '0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9' | ':' | ';' | '<' | '=' | '>' | '?' | '@' | 'A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z' | '[' | '\\' | ']' | '^' | '_' | '`' | 'a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z' | '{' | '|' | '}' | '~' | '\u007f' | '\u0080' | '\u0081' | '\u0082' | '\u0083' | '\u0084' | '\u0085' | '\u0086' | '\u0087' | '\u0088' | '\u0089' | '\u008a' | '\u008b' | '\u008c' | '\u008d' | '\u008e' | '\u008f' | '\u0090' | '\u0091' | '\u0092' | '\u0093' | '\u0094' | '\u0095' | '\u0096' | '\u0097' | '\u0098' | '\u0099' | '\u009a' | '\u009b' | '\u009c' | '\u009d' | '\u009e' | '\u009f' | '\u00a0' | '¡' | '¢' | '£' | '¤' | '¥' | '¦' | '§' | '¨' | '©' | 'ª' | '«' | '¬' | '\u00ad' | '®' | '¯' | '°' | '±' | '²' | '³' | '´' | 'µ' | '¶' | '·' | '¸' | '¹' | 'º' | '»' | '¼' | '½' | '¾' | '¿' | 'À' | 'Á' | 'Â' | 'Ã' | 'Ä' | 'Å' | 'Æ' | 'Ç' | 'È' | 'É' | 'Ê' | 'Ë' | 'Ì' | 'Í' | 'Î' | 'Ï' | 'Ð' | 'Ñ' | 'Ò' | 'Ó' | 'Ô' | 'Õ' | 'Ö' | '×' | 'Ø' | 'Ù' | 'Ú' | 'Û' | 'Ü' | 'Ý' | 'Þ' | 'ß' | 'à' | 'á' | 'â' | 'ã' | 'ä' | 'å' | 'æ' | 'ç' | 'è' | 'é' | 'ê' | 'ë' | 'ì' | 'í' | 'î' | 'ï' | 'ð' | 'ñ' | 'ò' | 'ó' | 'ô' | 'õ' | 'ö' | '÷' | 'ø' | 'ù' | 'ú' | 'û' | 'ü' | 'ý' | 'þ' | 'ÿ') [(-\U0010ffff]))> */
		nil,
		/* 35 mlLiteralString <- <('\'' '\'' '\'' <mlLiteralBody> ('\'' '\'' '\'') Action22)> */
		nil,
		/* 36 mlLiteralBody <- <(!('\'' '\'' '\'') (mlLiteralChar / newline))*> */
		nil,
		/* 37 mlLiteralChar <- <('\t' / [ -\U0010ffff])> */
		nil,
		/* 38 hexdigit <- <((&('a' | 'b' | 'c' | 'd' | 'e' | 'f') [a-f]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F') [A-F]) | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]))> */
		func() bool {
			position272, tokenIndex272 := position, tokenIndex
			{
				position273 := position
				{
					switch buffer[position] {
					case 'a', 'b', 'c', 'd', 'e', 'f':
						if c := buffer[position]; c < rune('a') || c > rune('f') {
							goto l272
						}
						position++
						break
					case 'A', 'B', 'C', 'D', 'E', 'F':
						if c := buffer[position]; c < rune('A') || c > rune('F') {
							goto l272
						}
						position++
						break
					default:
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l272
						}
						position++
						break
					}
				}

				add(rulehexdigit, position273)
			}
			return true
		l272:
			position, tokenIndex = position272, tokenIndex272
			return false
		},
		/* 39 hexQuad <- <(hexdigit hexdigit hexdigit hexdigit)> */
		func() bool {
			position275, tokenIndex275 := position, tokenIndex
			{
				position276 := position
				if !_rules[rulehexdigit]() {
					goto l275
				}
				if !_rules[rulehexdigit]() {
					goto l275
				}
				if !_rules[rulehexdigit]() {
					goto l275
				}
				if !_rules[rulehexdigit]() {
					goto l275
				}
				add(rulehexQuad, position276)
			}
			return true
		l275:
			position, tokenIndex = position275, tokenIndex275
			return false
		},
		/* 40 boolean <- <(('t' 'r' 'u' 'e') / ('f' 'a' 'l' 's' 'e'))> */
		nil,
		/* 41 dateFullYear <- <digitQuad> */
		nil,
		/* 42 dateMonth <- <digitDual> */
		nil,
		/* 43 dateMDay <- <digitDual> */
		nil,
		/* 44 timeHour <- <digitDual> */
		func() bool {
			position281, tokenIndex281 := position, tokenIndex
			{
				position282 := position
				if !_rules[ruledigitDual]() {
					goto l281
				}
				add(ruletimeHour, position282)
			}
			return true
		l281:
			position, tokenIndex = position281, tokenIndex281
			return false
		},
		/* 45 timeMinute <- <digitDual> */
		func() bool {
			position283, tokenIndex283 := position, tokenIndex
			{
				position284 := position
				if !_rules[ruledigitDual]() {
					goto l283
				}
				add(ruletimeMinute, position284)
			}
			return true
		l283:
			position, tokenIndex = position283, tokenIndex283
			return false
		},
		/* 46 timeSecond <- <digitDual> */
		nil,
		/* 47 timeSecfrac <- <('.' digit+)> */
		nil,
		/* 48 timeNumoffset <- <(('-' / '+') timeHour ':' timeMinute)> */
		nil,
		/* 49 timeOffset <- <('Z' / timeNumoffset)> */
		nil,
		/* 50 partialTime <- <(timeHour ':' timeMinute ':' timeSecond timeSecfrac?)> */
		func() bool {
			position289, tokenIndex289 := position, tokenIndex
			{
				position290 := position
				if !_rules[ruletimeHour]() {
					goto l289
				}
				if buffer[position] != rune(':') {
					goto l289
				}
				position++
				if !_rules[ruletimeMinute]() {
					goto l289
				}
				if buffer[position] != rune(':') {
					goto l289
				}
				position++
				{
					position291 := position
					if !_rules[ruledigitDual]() {
						goto l289
					}
					add(ruletimeSecond, position291)
				}
				{
					position292, tokenIndex292 := position, tokenIndex
					{
						position294 := position
						if buffer[position] != rune('.') {
							goto l292
						}
						position++
						if !_rules[ruledigit]() {
							goto l292
						}
					l295:
						{
							position296, tokenIndex296 := position, tokenIndex
							if !_rules[ruledigit]() {
								goto l296
							}
							goto l295
						l296:
							position, tokenIndex = position296, tokenIndex296
						}
						add(ruletimeSecfrac, position294)
					}
					goto l293
				l292:
					position, tokenIndex = position292, tokenIndex292
				}
			l293:
				add(rulepartialTime, position290)
			}
			return true
		l289:
			position, tokenIndex = position289, tokenIndex289
			return false
		},
		/* 51 fullDate <- <(dateFullYear '-' dateMonth '-' dateMDay)> */
		nil,
		/* 52 fullTime <- <(partialTime timeOffset?)> */
		nil,
		/* 53 datetime <- <((fullDate ('T' fullTime)?) / partialTime)> */
		nil,
		/* 54 digit <- <[0-9]> */
		func() bool {
			position300, tokenIndex300 := position, tokenIndex
			{
				position301 := position
				if c := buffer[position]; c < rune('0') || c > rune('9') {
					goto l300
				}
				position++
				add(ruledigit, position301)
			}
			return true
		l300:
			position, tokenIndex = position300, tokenIndex300
			return false
		},
		/* 55 digitDual <- <(digit digit)> */
		func() bool {
			position302, tokenIndex302 := position, tokenIndex
			{
				position303 := position
				if !_rules[ruledigit]() {
					goto l302
				}
				if !_rules[ruledigit]() {
					goto l302
				}
				add(ruledigitDual, position303)
			}
			return true
		l302:
			position, tokenIndex = position302, tokenIndex302
			return false
		},
		/* 56 digitQuad <- <(digitDual digitDual)> */
		nil,
		/* 57 array <- <('[' Action23 wsnl arrayValues? wsnl ']')> */
		nil,
		/* 58 arrayValues <- <(val Action24 (wsnl comment? wsnl arraySep wsnl comment? wsnl val Action25)* wsnl arraySep? wsnl comment?)> */
		nil,
		/* 59 arraySep <- <','> */
		func() bool {
			position307, tokenIndex307 := position, tokenIndex
			{
				position308 := position
				if buffer[position] != rune(',') {
					goto l307
				}
				position++
				add(rulearraySep, position308)
			}
			return true
		l307:
			position, tokenIndex = position307, tokenIndex307
			return false
		},
		/* 61 Action0 <- <{ _ = buffer }> */
		nil,
		nil,
		/* 63 Action1 <- <{ p.SetTableString(begin, end) }> */
		nil,
		/* 64 Action2 <- <{ p.AddLineCount(end - begin) }> */
		nil,
		/* 65 Action3 <- <{ p.AddLineCount(end - begin) }> */
		nil,
		/* 66 Action4 <- <{ p.AddKeyValue() }> */
		nil,
		/* 67 Action5 <- <{ p.SetKey(p.buffer, begin, end) }> */
		nil,
		/* 68 Action6 <- <{ p.SetKey(p.buffer, begin, end) }> */
		nil,
		/* 69 Action7 <- <{ p.SetTime(begin, end) }> */
		nil,
		/* 70 Action8 <- <{ p.SetFloat64(begin, end) }> */
		nil,
		/* 71 Action9 <- <{ p.SetInt64(begin, end) }> */
		nil,
		/* 72 Action10 <- <{ p.SetString(begin, end) }> */
		nil,
		/* 73 Action11 <- <{ p.SetBool(begin, end) }> */
		nil,
		/* 74 Action12 <- <{ p.SetArray(begin, end) }> */
		nil,
		/* 75 Action13 <- <{ p.SetTable(p.buffer, begin, end) }> */
		nil,
		/* 76 Action14 <- <{ p.SetArrayTable(p.buffer, begin, end) }> */
		nil,
		/* 77 Action15 <- <{ p.StartInlineTable() }> */
		nil,
		/* 78 Action16 <- <{ p.EndInlineTable() }> */
		nil,
		/* 79 Action17 <- <{ p.AddTableKey() }> */
		nil,
		/* 80 Action18 <- <{ p.SetBasicString(p.buffer, begin, end) }> */
		nil,
		/* 81 Action19 <- <{ p.SetMultilineString() }> */
		nil,
		/* 82 Action20 <- <{ p.AddMultilineBasicBody(p.buffer, begin, end) }> */
		nil,
		/* 83 Action21 <- <{ p.SetLiteralString(p.buffer, begin, end) }> */
		nil,
		/* 84 Action22 <- <{ p.SetMultilineLiteralString(p.buffer, begin, end) }> */
		nil,
		/* 85 Action23 <- <{ p.StartArray() }> */
		nil,
		/* 86 Action24 <- <{ p.AddArrayVal() }> */
		nil,
		/* 87 Action25 <- <{ p.AddArrayVal() }> */
		nil,
	}
	p.rules = _rules
}
