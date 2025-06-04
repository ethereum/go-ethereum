/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { Parser } from "../Parser";
import { ParseTreeVisitor } from "./ParseTreeVisitor";
import { RuleContext } from "../RuleContext";
import { SyntaxTree } from "./SyntaxTree";
/** An interface to access the tree of {@link RuleContext} objects created
 *  during a parse that makes the data structure look like a simple parse tree.
 *  This node represents both internal nodes, rule invocations,
 *  and leaf nodes, token matches.
 *
 *  The payload is either a {@link Token} or a {@link RuleContext} object.
 */
export interface ParseTree extends SyntaxTree {
    readonly parent: ParseTree | undefined;
    /**
     * Set the parent for this node.
     *
     * @since 4.7
     */
    setParent(parent: RuleContext): void;
    getChild(i: number): ParseTree;
    /** The {@link ParseTreeVisitor} needs a double dispatch method. */
    accept<T>(visitor: ParseTreeVisitor<T>): T;
    /** Return the combined text of all leaf nodes. Does not get any
     *  off-channel tokens (if any) so won't return whitespace and
     *  comments if they are sent to parser on hidden channel.
     */
    readonly text: string;
    /** Specialize toStringTree so that it can print out more information
     * 	based upon the parser.
     */
    toStringTree(parser?: Parser): string;
}
