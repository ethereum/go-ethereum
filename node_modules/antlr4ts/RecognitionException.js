"use strict";
/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
Object.defineProperty(exports, "__esModule", { value: true });
exports.RecognitionException = void 0;
/** The root of the ANTLR exception hierarchy. In general, ANTLR tracks just
 *  3 kinds of errors: prediction errors, failed predicate errors, and
 *  mismatched input errors. In each case, the parser knows where it is
 *  in the input, where it is in the ATN, the rule invocation stack,
 *  and what kind of problem occurred.
 */
class RecognitionException extends Error {
    constructor(recognizer, input, ctx, message) {
        super(message);
        this._offendingState = -1;
        this._recognizer = recognizer;
        this.input = input;
        this.ctx = ctx;
        if (recognizer) {
            this._offendingState = recognizer.state;
        }
    }
    /**
     * Get the ATN state number the parser was in at the time the error
     * occurred. For {@link NoViableAltException} and
     * {@link LexerNoViableAltException} exceptions, this is the
     * {@link DecisionState} number. For others, it is the state whose outgoing
     * edge we couldn't match.
     *
     * If the state number is not known, this method returns -1.
     */
    get offendingState() {
        return this._offendingState;
    }
    setOffendingState(offendingState) {
        this._offendingState = offendingState;
    }
    /**
     * Gets the set of input symbols which could potentially follow the
     * previously matched symbol at the time this exception was thrown.
     *
     * If the set of expected tokens is not known and could not be computed,
     * this method returns `undefined`.
     *
     * @returns The set of token types that could potentially follow the current
     * state in the ATN, or `undefined` if the information is not available.
     */
    get expectedTokens() {
        if (this._recognizer) {
            return this._recognizer.atn.getExpectedTokens(this._offendingState, this.ctx);
        }
        return undefined;
    }
    /**
     * Gets the {@link RuleContext} at the time this exception was thrown.
     *
     * If the context is not available, this method returns `undefined`.
     *
     * @returns The {@link RuleContext} at the time this exception was thrown.
     * If the context is not available, this method returns `undefined`.
     */
    get context() {
        return this.ctx;
    }
    /**
     * Gets the input stream which is the symbol source for the recognizer where
     * this exception was thrown.
     *
     * If the input stream is not available, this method returns `undefined`.
     *
     * @returns The input stream which is the symbol source for the recognizer
     * where this exception was thrown, or `undefined` if the stream is not
     * available.
     */
    get inputStream() {
        return this.input;
    }
    getOffendingToken(recognizer) {
        if (recognizer && recognizer !== this._recognizer) {
            return undefined;
        }
        return this.offendingToken;
    }
    setOffendingToken(recognizer, offendingToken) {
        if (recognizer === this._recognizer) {
            this.offendingToken = offendingToken;
        }
    }
    /**
     * Gets the {@link Recognizer} where this exception occurred.
     *
     * If the recognizer is not available, this method returns `undefined`.
     *
     * @returns The recognizer where this exception occurred, or `undefined` if
     * the recognizer is not available.
     */
    get recognizer() {
        return this._recognizer;
    }
}
exports.RecognitionException = RecognitionException;
//# sourceMappingURL=RecognitionException.js.map