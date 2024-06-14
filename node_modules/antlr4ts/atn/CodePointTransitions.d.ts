/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ATNState } from "./ATNState";
import { Transition } from "./Transition";
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
export declare function createWithCodePoint(target: ATNState, codePoint: number): Transition;
/**
 * If {@code codePointFrom} and {@code codePointTo} are both
 * <= U+FFFF, returns a new {@link RangeTransition}.
 * Otherwise, returns a new {@link SetTransition}.
 */
export declare function createWithCodePointRange(target: ATNState, codePointFrom: number, codePointTo: number): Transition;
