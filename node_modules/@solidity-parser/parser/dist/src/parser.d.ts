import { ASTNode, ASTVisitor, SourceUnit } from './ast-types';
import { ParseOptions, Token, TokenizeOptions } from './types';
interface ParserErrorItem {
    message: string;
    line: number;
    column: number;
}
declare type ParseResult = SourceUnit & {
    errors?: any[];
    tokens?: Token[];
};
export declare class ParserError extends Error {
    errors: ParserErrorItem[];
    constructor(args: {
        errors: ParserErrorItem[];
    });
}
export declare function tokenize(input: string, options?: TokenizeOptions): any;
export declare function parse(input: string, options?: ParseOptions): ParseResult;
export declare function visit(node: unknown, visitor: ASTVisitor, nodeParent?: ASTNode): void;
export {};
