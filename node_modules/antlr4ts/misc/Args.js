"use strict";
/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
Object.defineProperty(exports, "__esModule", { value: true });
exports.notNull = void 0;
// ConvertTo-TS run at 2016-10-04T11:26:39.6568608-07:00
/**
 * Validates that an argument is not `null` or `undefined`.
 *
 * @param parameterName The name of the parameter
 * @param value The argument value
 *
 * @throws `TypeError` if `value` is `null` or `undefined`.
 */
function notNull(parameterName, value) {
    if (value == null) {
        throw new TypeError(parameterName + " cannot be null or undefined.");
    }
}
exports.notNull = notNull;
//# sourceMappingURL=Args.js.map