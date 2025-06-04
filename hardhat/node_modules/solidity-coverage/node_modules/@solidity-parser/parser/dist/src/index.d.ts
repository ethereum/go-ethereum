export * from './parser';
import { ParserError, parse, tokenize, visit } from './parser';
export type { ParseOptions } from './types';
declare const _default: {
    ParserError: typeof ParserError;
    parse: typeof parse;
    tokenize: typeof tokenize;
    visit: typeof visit;
};
export default _default;
