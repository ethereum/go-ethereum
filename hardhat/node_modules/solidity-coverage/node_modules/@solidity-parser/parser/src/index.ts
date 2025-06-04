export * from './parser'
import { ParserError, parse, tokenize, visit } from './parser'

export type { ParseOptions } from './types'

export default { ParserError, parse, tokenize, visit }
