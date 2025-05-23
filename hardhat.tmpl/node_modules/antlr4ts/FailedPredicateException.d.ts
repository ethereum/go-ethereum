/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { Parser } from "./Parser";
import { RecognitionException } from "./RecognitionException";
/** A semantic predicate failed during validation.  Validation of predicates
 *  occurs when normally parsing the alternative just like matching a token.
 *  Disambiguating predicate evaluation occurs when we test a predicate during
 *  prediction.
 */
export declare class FailedPredicateException extends RecognitionException {
    private _ruleIndex;
    private _predicateIndex;
    private _predicate?;
    constructor(recognizer: Parser, predicate?: string, message?: string);
    get ruleIndex(): number;
    get predicateIndex(): number;
    get predicate(): string | undefined;
    private static formatMessage;
}
