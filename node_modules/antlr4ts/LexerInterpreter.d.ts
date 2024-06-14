/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ATN } from "./atn/ATN";
import { CharStream } from "./CharStream";
import { Lexer } from "./Lexer";
import { Vocabulary } from "./Vocabulary";
export declare class LexerInterpreter extends Lexer {
    protected _grammarFileName: string;
    protected _atn: ATN;
    protected _ruleNames: string[];
    protected _channelNames: string[];
    protected _modeNames: string[];
    private _vocabulary;
    constructor(grammarFileName: string, vocabulary: Vocabulary, ruleNames: string[], channelNames: string[], modeNames: string[], atn: ATN, input: CharStream);
    get atn(): ATN;
    get grammarFileName(): string;
    get ruleNames(): string[];
    get channelNames(): string[];
    get modeNames(): string[];
    get vocabulary(): Vocabulary;
}
