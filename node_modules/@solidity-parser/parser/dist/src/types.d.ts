import { Token as Antlr4TsToken } from "antlr4ts";
export interface Node {
    type: string;
}
export declare type AntlrToken = Antlr4TsToken;
export interface TokenizeOptions {
    range?: boolean;
    loc?: boolean;
}
export interface ParseOptions extends TokenizeOptions {
    tokens?: boolean;
    tolerant?: boolean;
}
export interface Token {
    type: string;
    value: string | undefined;
    range?: [number, number];
    loc?: {
        start: {
            line: number;
            column: number;
        };
        end: {
            line: number;
            column: number;
        };
    };
}
