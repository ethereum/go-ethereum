/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ParseTree } from "./ParseTree";
/**
 * Associate a property with a parse tree node. Useful with parse tree listeners
 * that need to associate values with particular tree nodes, kind of like
 * specifying a return value for the listener event method that visited a
 * particular node. Example:
 *
 * ```
 * ParseTreeProperty<Integer> values = new ParseTreeProperty<Integer>();
 * values.put(tree, 36);
 * int x = values.get(tree);
 * values.removeFrom(tree);
 * ```
 *
 * You would make one decl (values here) in the listener and use lots of times
 * in your event methods.
 */
export declare class ParseTreeProperty<V> {
    private _symbol;
    constructor(name?: string);
    get(node: ParseTree): V;
    set(node: ParseTree, value: V): void;
    removeFrom(node: ParseTree): V;
}
