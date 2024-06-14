/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ATNConfigSet } from "./atn/ATNConfigSet";
import { RecognitionException } from "./RecognitionException";
import { Lexer } from "./Lexer";
import { CharStream } from "./CharStream";
export declare class LexerNoViableAltException extends RecognitionException {
    /** Matching attempted at what input index? */
    private _startIndex;
    /** Which configurations did we try at input.index that couldn't match input.LA(1)? */
    private _deadEndConfigs?;
    constructor(lexer: Lexer | undefined, input: CharStream, startIndex: number, deadEndConfigs: ATNConfigSet | undefined);
    get startIndex(): number;
    get deadEndConfigs(): ATNConfigSet | undefined;
    get inputStream(): CharStream;
    toString(): string;
}
