/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ErrorNode } from "./ErrorNode";
import { ParseTree } from "./ParseTree";
import { RuleNode } from "./RuleNode";
import { TerminalNode } from "./TerminalNode";
/**
 * This interface defines the basic notion of a parse tree visitor. Generated
 * visitors implement this interface and the `XVisitor` interface for
 * grammar `X`.
 *
 * @author Sam Harwell
 * @param <Result> The return type of the visit operation. Use {@link Void} for
 * operations with no return type.
 */
export interface ParseTreeVisitor<Result> {
    /**
     * Visit a parse tree, and return a user-defined result of the operation.
     *
     * @param tree The {@link ParseTree} to visit.
     * @returns The result of visiting the parse tree.
     */
    visit(/*@NotNull*/ tree: ParseTree): Result;
    /**
     * Visit the children of a node, and return a user-defined result
     * of the operation.
     *
     * @param node The {@link RuleNode} whose children should be visited.
     * @returns The result of visiting the children of the node.
     */
    visitChildren(/*@NotNull*/ node: RuleNode): Result;
    /**
     * Visit a terminal node, and return a user-defined result of the operation.
     *
     * @param node The {@link TerminalNode} to visit.
     * @returns The result of visiting the node.
     */
    visitTerminal(/*@NotNull*/ node: TerminalNode): Result;
    /**
     * Visit an error node, and return a user-defined result of the operation.
     *
     * @param node The {@link ErrorNode} to visit.
     * @returns The result of visiting the node.
     */
    visitErrorNode(/*@NotNull*/ node: ErrorNode): Result;
}
