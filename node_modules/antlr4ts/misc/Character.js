"use strict";
/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
Object.defineProperty(exports, "__esModule", { value: true });
exports.isSupplementaryCodePoint = exports.isLowSurrogate = exports.isHighSurrogate = void 0;
function isHighSurrogate(ch) {
    return ch >= 0xD800 && ch <= 0xDBFF;
}
exports.isHighSurrogate = isHighSurrogate;
function isLowSurrogate(ch) {
    return ch >= 0xDC00 && ch <= 0xDFFF;
}
exports.isLowSurrogate = isLowSurrogate;
function isSupplementaryCodePoint(ch) {
    return ch >= 0x10000;
}
exports.isSupplementaryCodePoint = isSupplementaryCodePoint;
//# sourceMappingURL=Character.js.map