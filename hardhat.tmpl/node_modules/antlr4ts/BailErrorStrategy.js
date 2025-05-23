"use strict";
/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
var __decorate = (this && this.__decorate) || function (decorators, target, key, desc) {
    var c = arguments.length, r = c < 3 ? target : desc === null ? desc = Object.getOwnPropertyDescriptor(target, key) : desc, d;
    if (typeof Reflect === "object" && typeof Reflect.decorate === "function") r = Reflect.decorate(decorators, target, key, desc);
    else for (var i = decorators.length - 1; i >= 0; i--) if (d = decorators[i]) r = (c < 3 ? d(r) : c > 3 ? d(target, key, r) : d(target, key)) || r;
    return c > 3 && r && Object.defineProperty(target, key, r), r;
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.BailErrorStrategy = void 0;
// ConvertTo-TS run at 2016-10-04T11:26:49.2855056-07:00
const DefaultErrorStrategy_1 = require("./DefaultErrorStrategy");
const InputMismatchException_1 = require("./InputMismatchException");
const Decorators_1 = require("./Decorators");
const ParseCancellationException_1 = require("./misc/ParseCancellationException");
/**
 * This implementation of {@link ANTLRErrorStrategy} responds to syntax errors
 * by immediately canceling the parse operation with a
 * {@link ParseCancellationException}. The implementation ensures that the
 * {@link ParserRuleContext#exception} field is set for all parse tree nodes
 * that were not completed prior to encountering the error.
 *
 * This error strategy is useful in the following scenarios.
 *
 * * **Two-stage parsing:** This error strategy allows the first
 *   stage of two-stage parsing to immediately terminate if an error is
 *   encountered, and immediately fall back to the second stage. In addition to
 *   avoiding wasted work by attempting to recover from errors here, the empty
 *   implementation of {@link BailErrorStrategy#sync} improves the performance of
 *   the first stage.
 * * **Silent validation:** When syntax errors are not being
 *   reported or logged, and the parse result is simply ignored if errors occur,
 *   the {@link BailErrorStrategy} avoids wasting work on recovering from errors
 *   when the result will be ignored either way.
 *
 * ```
 * myparser.errorHandler = new BailErrorStrategy();
 * ```
 *
 * @see Parser.errorHandler
 */
class BailErrorStrategy extends DefaultErrorStrategy_1.DefaultErrorStrategy {
    /** Instead of recovering from exception `e`, re-throw it wrapped
     *  in a {@link ParseCancellationException} so it is not caught by the
     *  rule function catches.  Use {@link Exception#getCause()} to get the
     *  original {@link RecognitionException}.
     */
    recover(recognizer, e) {
        for (let context = recognizer.context; context; context = context.parent) {
            context.exception = e;
        }
        throw new ParseCancellationException_1.ParseCancellationException(e);
    }
    /** Make sure we don't attempt to recover inline; if the parser
     *  successfully recovers, it won't throw an exception.
     */
    recoverInline(recognizer) {
        let e = new InputMismatchException_1.InputMismatchException(recognizer);
        for (let context = recognizer.context; context; context = context.parent) {
            context.exception = e;
        }
        throw new ParseCancellationException_1.ParseCancellationException(e);
    }
    /** Make sure we don't attempt to recover from problems in subrules. */
    sync(recognizer) {
        // intentionally empty
    }
}
__decorate([
    Decorators_1.Override
], BailErrorStrategy.prototype, "recover", null);
__decorate([
    Decorators_1.Override
], BailErrorStrategy.prototype, "recoverInline", null);
__decorate([
    Decorators_1.Override
], BailErrorStrategy.prototype, "sync", null);
exports.BailErrorStrategy = BailErrorStrategy;
//# sourceMappingURL=BailErrorStrategy.js.map