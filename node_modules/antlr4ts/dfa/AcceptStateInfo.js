"use strict";
/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
Object.defineProperty(exports, "__esModule", { value: true });
exports.AcceptStateInfo = void 0;
/**
 * Stores information about a {@link DFAState} which is an accept state under
 * some condition. Certain settings, such as
 * {@link ParserATNSimulator#getPredictionMode()}, may be used in addition to
 * this information to determine whether or not a particular state is an accept
 * state.
 *
 * @author Sam Harwell
 */
class AcceptStateInfo {
    constructor(prediction, lexerActionExecutor) {
        this._prediction = prediction;
        this._lexerActionExecutor = lexerActionExecutor;
    }
    /**
     * Gets the prediction made by this accept state. Note that this value
     * assumes the predicates, if any, in the {@link DFAState} evaluate to
     * `true`. If predicate evaluation is enabled, the final prediction of
     * the accept state will be determined by the result of predicate
     * evaluation.
     */
    get prediction() {
        return this._prediction;
    }
    /**
     * Gets the {@link LexerActionExecutor} which can be used to execute actions
     * and/or commands after the lexer matches a token.
     */
    get lexerActionExecutor() {
        return this._lexerActionExecutor;
    }
}
exports.AcceptStateInfo = AcceptStateInfo;
//# sourceMappingURL=AcceptStateInfo.js.map