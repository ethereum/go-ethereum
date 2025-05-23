import { ANTLRInputStream, CommonTokenStream } from 'antlr4ts'

import { SolidityLexer } from './antlr/SolidityLexer'
import { SolidityParser } from './antlr/SolidityParser'
import { ASTNode, astNodeTypes, ASTNodeTypeString, ASTVisitor, SourceUnit } from './ast-types'
import { ASTBuilder } from './ASTBuilder'
import ErrorListener from './ErrorListener'
import { buildTokenList } from './tokens'
import { ParseOptions, Token, TokenizeOptions } from './types'

interface ParserErrorItem {
  message: string
  line: number
  column: number
}

type ParseResult = SourceUnit & {
  errors?: any[]
  tokens?: Token[]
}

export class ParserError extends Error {
  public errors: ParserErrorItem[]

  constructor(args: { errors: ParserErrorItem[] }) {
    super()
    const { message, line, column } = args.errors[0]
    this.message = `${message} (${line}:${column})`
    this.errors = args.errors

    if (Error.captureStackTrace !== undefined) {
      Error.captureStackTrace(this, this.constructor)
    } else {
      this.stack = new Error().stack
    }
  }
}

export function tokenize(input: string, options: TokenizeOptions = {}): any {
  const inputStream = new ANTLRInputStream(input)
  const lexer = new SolidityLexer(inputStream)

  return buildTokenList(lexer.getAllTokens(), options)
}

export function parse(
  input: string,
  options: ParseOptions = {}
): ParseResult {
  const inputStream = new ANTLRInputStream(input)
  const lexer = new SolidityLexer(inputStream)
  const tokenStream = new CommonTokenStream(lexer)
  const parser = new SolidityParser(tokenStream)

  const listener = new ErrorListener()
  lexer.removeErrorListeners()
  lexer.addErrorListener(listener)

  parser.removeErrorListeners()
  parser.addErrorListener(listener)
  parser.buildParseTree = true

  const sourceUnit = parser.sourceUnit()

  const astBuilder = new ASTBuilder(options)

  astBuilder.visit(sourceUnit)

  const ast: ParseResult | null = astBuilder.result as any

  if (ast === null) {
    throw new Error('ast should never be null')
  }

  let tokenList: Token[] = []
  if (options.tokens === true) {
    tokenList = buildTokenList(tokenStream.getTokens(), options)
  }

  if (options.tolerant !== true && listener.hasErrors()) {
    throw new ParserError({ errors: listener.getErrors() })
  }
  if (options.tolerant === true && listener.hasErrors()) {
    ast.errors = listener.getErrors()
  }
  if (options.tokens === true) {
    ast.tokens = tokenList
  }

  return ast
}

function _isASTNode(node: unknown): node is ASTNode {
  if (typeof node !== 'object' || node === null) {
    return false
  }

  const nodeAsAny: any = node

  if (Object.prototype.hasOwnProperty.call(nodeAsAny, 'type') && typeof nodeAsAny.type === "string") {
    return astNodeTypes.includes(nodeAsAny.type)
  }

  return false;
}

export function visit(node: unknown, visitor: ASTVisitor, nodeParent?: ASTNode): void {
  if (Array.isArray(node)) {
    node.forEach((child) => visit(child, visitor, nodeParent))
  }

  if (!_isASTNode(node)) return

  let cont = true

  if (visitor[node.type] !== undefined) {
    // TODO can we avoid this `as any`
    cont = visitor[node.type]!(node as any, nodeParent)
  }

  if (cont === false) return

  for (const prop in node) {
    if (Object.prototype.hasOwnProperty.call(node, prop)) {
      // TODO can we avoid this `as any`
      visit((node as any)[prop], visitor, node)
    }
  }

  const selector = (node.type + ':exit') as `${ASTNodeTypeString}:exit`
  if (visitor[selector] !== undefined) {
      // TODO can we avoid this `as any`
    visitor[selector]!(node as any, nodeParent)
  }
}
