import { Token, AntlrToken, TokenizeOptions } from './types'
import untypedTokens from './tokens-string'

const tokens = untypedTokens as string

const TYPE_TOKENS = [
  'var',
  'bool',
  'address',
  'string',
  'Int',
  'Uint',
  'Byte',
  'Fixed',
  'UFixed',
]

function rsplit(str: string, value: string) {
  const index = str.lastIndexOf(value)
  return [str.substring(0, index), str.substring(index + 1, str.length)]
}

function normalizeTokenType(value: string) {
  if (value.endsWith("'")) {
    value = value.substring(0, value.length - 1)
  }
  if (value.startsWith("'")) {
    value = value.substring(1, value.length)
  }
  return value
}

function getTokenType(value: string) {
  if (value === 'Identifier' || value === 'from') {
    return 'Identifier'
  } else if (value === 'TrueLiteral' || value === 'FalseLiteral') {
    return 'Boolean'
  } else if (value === 'VersionLiteral') {
    return 'Version'
  } else if (value === 'StringLiteral') {
    return 'String'
  } else if (TYPE_TOKENS.includes(value)) {
    return 'Type'
  } else if (value === 'NumberUnit') {
    return 'Subdenomination'
  } else if (value === 'DecimalNumber') {
    return 'Numeric'
  } else if (value === 'HexLiteral') {
    return 'Hex'
  } else if (value === 'ReservedKeyword') {
    return 'Reserved'
  } else if (/^\W+$/.test(value)) {
    return 'Punctuator'
  } else {
    return 'Keyword'
  }
}

function getTokenTypeMap() {
  return tokens
    .split('\n')
    .map((line) => rsplit(line, '='))
    .reduce((acum: any, [value, key]) => {
      acum[parseInt(key, 10)] = normalizeTokenType(value)
      return acum
    }, {})
}

export function buildTokenList(
  tokensArg: AntlrToken[],
  options: TokenizeOptions
): Token[] {
  const tokenTypes = getTokenTypeMap()

  const result = tokensArg.map((token) => {
    const type = getTokenType(tokenTypes[token.type])
    const node: Token = { type, value: token.text }
    if (options.range === true) {
      node.range = [token.startIndex, token.stopIndex + 1]
    }
    if (options.loc === true) {
      node.loc = {
        start: { line: token.line, column: token.charPositionInLine },
        end: { line: token.line, column: token.charPositionInLine + (token.text?.length ?? 0) },
      }
    }
    return node
  })

  return result
}
