"use strict";
/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
Object.defineProperty(exports, "__esModule", { value: true });
exports.ParseCancellationException = void 0;
// ConvertTo-TS run at 2016-10-04T11:26:42.5447085-07:00
/**
 * This exception is thrown to cancel a parsing operation. This exception does
 * not extend {@link RecognitionException}, allowing it to bypass the standard
 * error recovery mechanisms. {@link BailErrorStrategy} throws this exception in
 * response to a parse error.
 *
 * @author Sam Harwell
 */
class ParseCancellationException extends Error {
    constructor(cause) {
        super(cause.message);
        this.cause = cause;
        this.stack = cause.stack;
    }
    getCause() {
        return this.cause;
    }
}
exports.ParseCancellationException = ParseCancellationException;
//# sourceMappingURL=ParseCancellationException.js.map