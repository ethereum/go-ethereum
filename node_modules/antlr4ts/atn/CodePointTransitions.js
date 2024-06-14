"use strict";
/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
Object.defineProperty(exports, "__esModule", { value: true });
exports.createWithCodePointRange = exports.createWithCodePoint = void 0;
const Character = require("../misc/Character");
const AtomTransition_1 = require("./AtomTransition");
const IntervalSet_1 = require("../misc/IntervalSet");
const RangeTransition_1 = require("./RangeTransition");
const SetTransition_1 = require("./SetTransition");
/**
 * Utility functions to create {@link AtomTransition}, {@link RangeTransition},
 * and {@link SetTransition} appropriately based on the range of the input.
 *
 * To keep the serialized ATN size small, we only inline atom and
 * range transitions for Unicode code points <= U+FFFF.
 *
 * Whenever we encounter a Unicode code point > U+FFFF, we represent that
 * as a set transition (even if it is logically an atom or a range).
 */
/**
 * If {@code codePoint} is <= U+FFFF, returns a new {@link AtomTransition}.
 * Otherwise, returns a new {@link SetTransition}.
 */
function createWithCodePoint(target, codePoint) {
    if (Character.isSupplementaryCodePoint(codePoint)) {
        return new SetTransition_1.SetTransition(target, IntervalSet_1.IntervalSet.of(codePoint));
    }
    else {
        return new AtomTransition_1.AtomTransition(target, codePoint);
    }
}
exports.createWithCodePoint = createWithCodePoint;
/**
 * If {@code codePointFrom} and {@code codePointTo} are both
 * <= U+FFFF, returns a new {@link RangeTransition}.
 * Otherwise, returns a new {@link SetTransition}.
 */
function createWithCodePointRange(target, codePointFrom, codePointTo) {
    if (Character.isSupplementaryCodePoint(codePointFrom) || Character.isSupplementaryCodePoint(codePointTo)) {
        return new SetTransition_1.SetTransition(target, IntervalSet_1.IntervalSet.of(codePointFrom, codePointTo));
    }
    else {
        return new RangeTransition_1.RangeTransition(target, codePointFrom, codePointTo);
    }
}
exports.createWithCodePointRange = createWithCodePointRange;
//# sourceMappingURL=CodePointTransitions.js.map