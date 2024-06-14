/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { Tree } from "./Tree";
import { Interval } from "../misc/Interval";
/** A tree that knows about an interval in a token stream
 *  is some kind of syntax tree. Subinterfaces distinguish
 *  between parse trees and other kinds of syntax trees we might want to create.
 */
export interface SyntaxTree extends Tree {
    /**
     * Return an {@link Interval} indicating the index in the
     * {@link TokenStream} of the first and last token associated with this
     * subtree. If this node is a leaf, then the interval represents a single
     * token and has interval i..i for token index i.
     *
     * An interval of i..i-1 indicates an empty interval at position
     * i in the input stream, where 0 &lt;= i &lt;= the size of the input
     * token stream.  Currently, the code base can only have i=0..n-1 but
     * in concept one could have an empty interval after EOF.
     *
     * If source interval is unknown, this returns {@link Interval#INVALID}.
     *
     * As a weird special case, the source interval for rules matched after
     * EOF is unspecified.
     */
    readonly sourceInterval: Interval;
}
