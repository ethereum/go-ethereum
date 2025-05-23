/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ATNConfigSet } from "./atn/ATNConfigSet";
import { Parser } from "./Parser";
import { ParserRuleContext } from "./ParserRuleContext";
import { RecognitionException } from "./RecognitionException";
import { Recognizer } from "./Recognizer";
import { Token } from "./Token";
import { TokenStream } from "./TokenStream";
/** Indicates that the parser could not decide which of two or more paths
 *  to take based upon the remaining input. It tracks the starting token
 *  of the offending input and also knows where the parser was
 *  in the various paths when the error. Reported by reportNoViableAlternative()
 */
export declare class NoViableAltException extends RecognitionException {
    /** Which configurations did we try at input.index that couldn't match input.LT(1)? */
    private _deadEndConfigs?;
    /** The token object at the start index; the input stream might
     * 	not be buffering tokens so get a reference to it. (At the
     *  time the error occurred, of course the stream needs to keep a
     *  buffer all of the tokens but later we might not have access to those.)
     */
    private _startToken;
    constructor(/*@NotNull*/ recognizer: Parser);
    constructor(recognizer: Recognizer<Token, any>, input: TokenStream, startToken: Token, offendingToken: Token, deadEndConfigs: ATNConfigSet | undefined, ctx: ParserRuleContext);
    get startToken(): Token;
    get deadEndConfigs(): ATNConfigSet | undefined;
}
