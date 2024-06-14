/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ANTLRErrorListener } from "../../ANTLRErrorListener";
import { Recognizer } from "../../Recognizer";
import { RecognitionException } from "../../RecognitionException";
export declare class XPathLexerErrorListener implements ANTLRErrorListener<number> {
    syntaxError<T extends number>(recognizer: Recognizer<T, any>, offendingSymbol: T | undefined, line: number, charPositionInLine: number, msg: string, e: RecognitionException | undefined): void;
}
