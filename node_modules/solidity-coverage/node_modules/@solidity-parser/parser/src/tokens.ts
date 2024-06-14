import { Token as AntlrToken } from 'antlr4'
import { Token, TokenizeOptions } from './types'
import { tokens } from './antlr/solidity-tokens'
import type { Comment, Location } from './ast-types'

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

function range(token: AntlrToken): [number, number] {
  return [token.start, token.stop + 1]
}

function loc(token: AntlrToken): Location {
  const tokenText = token.text ?? ''
  const textInLines = tokenText.split(/\r?\n/)
  const numberOfNewLines = textInLines.length - 1
  return {
    start: { line: token.line, column: token.column },
    end: {
      line: token.line + numberOfNewLines,
      column:
        textInLines[numberOfNewLines].length +
        (numberOfNewLines === 0 ? token.column : 0),
    },
  }
}

export function buildTokenList(
  tokensArg: AntlrToken[],
  options: TokenizeOptions
): Token[] {
  return tokensArg.map((token) => {
    const type = getTokenType(tokens[token.type.toString()])
    const node: Token = { type, value: token.text }
    if (options.range === true) {
      node.range = range(token)
    }
    if (options.loc === true) {
      node.loc = loc(token)
    }
    return node
  })
}

export function buildCommentList(
  tokensArg: AntlrToken[],
  commentsChannelId: number,
  options: TokenizeOptions
): Comment[] {
  return tokensArg
    .filter((token) => token.channel === commentsChannelId)
    .map((token) => {
      const comment: Comment = token.text.startsWith('//')
        ? { type: 'LineComment', value: token.text.slice(2) }
        : { type: 'BlockComment', value: token.text.slice(2, -2) }
      if (options.range === true) {
        comment.range = range(token)
      }
      if (options.loc === true) {
        comment.loc = loc(token)
      }
      return comment
    })
}
