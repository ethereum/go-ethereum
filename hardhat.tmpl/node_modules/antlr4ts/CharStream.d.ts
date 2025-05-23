/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { Interval } from "./misc/Interval";
import { IntStream } from "./IntStream";
/** A source of characters for an ANTLR lexer. */
export interface CharStream extends IntStream {
    /**
     * This method returns the text for a range of characters within this input
     * stream. This method is guaranteed to not throw an exception if the
     * specified `interval` lies entirely within a marked range. For more
     * information about marked ranges, see {@link IntStream#mark}.
     *
     * @param interval an interval within the stream
     * @returns the text of the specified interval
     *
     * @throws NullPointerException if `interval` is `undefined`
     * @throws IllegalArgumentException if `interval.a < 0`, or if
     * `interval.b < interval.a - 1`, or if `interval.b` lies at or
     * past the end of the stream
     * @throws UnsupportedOperationException if the stream does not support
     * getting the text of the specified interval
     */
    getText(/*@NotNull*/ interval: Interval): string;
}
