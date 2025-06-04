/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { Parser } from "./Parser";
import { Recognizer } from "./Recognizer";
import { RuleNode } from "./tree/RuleNode";
import { ParseTree } from "./tree/ParseTree";
import { Interval } from "./misc/Interval";
import { ParseTreeVisitor } from "./tree/ParseTreeVisitor";
/** A rule context is a record of a single rule invocation.
 *
 *  We form a stack of these context objects using the parent
 *  pointer. A parent pointer of `undefined` indicates that the current
 *  context is the bottom of the stack. The ParserRuleContext subclass
 *  as a children list so that we can turn this data structure into a
 *  tree.
 *
 *  The root node always has a `undefined` pointer and invokingState of -1.
 *
 *  Upon entry to parsing, the first invoked rule function creates a
 *  context object (a subclass specialized for that rule such as
 *  SContext) and makes it the root of a parse tree, recorded by field
 *  Parser._ctx.
 *
 *  public final SContext s() throws RecognitionException {
 *      SContext _localctx = new SContext(_ctx, state); <-- create new node
 *      enterRule(_localctx, 0, RULE_s);                     <-- push it
 *      ...
 *      exitRule();                                          <-- pop back to _localctx
 *      return _localctx;
 *  }
 *
 *  A subsequent rule invocation of r from the start rule s pushes a
 *  new context object for r whose parent points at s and use invoking
 *  state is the state with r emanating as edge label.
 *
 *  The invokingState fields from a context object to the root
 *  together form a stack of rule indication states where the root
 *  (bottom of the stack) has a -1 sentinel value. If we invoke start
 *  symbol s then call r1, which calls r2, the  would look like
 *  this:
 *
 *     SContext[-1]   <- root node (bottom of the stack)
 *     R1Context[p]   <- p in rule s called r1
 *     R2Context[q]   <- q in rule r1 called r2
 *
 *  So the top of the stack, _ctx, represents a call to the current
 *  rule and it holds the return address from another rule that invoke
 *  to this rule. To invoke a rule, we must always have a current context.
 *
 *  The parent contexts are useful for computing lookahead sets and
 *  getting error information.
 *
 *  These objects are used during parsing and prediction.
 *  For the special case of parsers, we use the subclass
 *  ParserRuleContext.
 *
 *  @see ParserRuleContext
 */
export declare class RuleContext extends RuleNode {
    _parent: RuleContext | undefined;
    invokingState: number;
    constructor();
    constructor(parent: RuleContext | undefined, invokingState: number);
    static getChildContext(parent: RuleContext, invokingState: number): RuleContext;
    depth(): number;
    /** A context is empty if there is no invoking state; meaning nobody called
     *  current context.
     */
    get isEmpty(): boolean;
    get sourceInterval(): Interval;
    get ruleContext(): RuleContext;
    get parent(): RuleContext | undefined;
    /** @since 4.7. {@see ParseTree#setParent} comment */
    setParent(parent: RuleContext): void;
    get payload(): RuleContext;
    /** Return the combined text of all child nodes. This method only considers
     *  tokens which have been added to the parse tree.
     *
     *  Since tokens on hidden channels (e.g. whitespace or comments) are not
     *  added to the parse trees, they will not appear in the output of this
     *  method.
     */
    get text(): string;
    get ruleIndex(): number;
    /** For rule associated with this parse tree internal node, return
     *  the outer alternative number used to match the input. Default
     *  implementation does not compute nor store this alt num. Create
     *  a subclass of ParserRuleContext with backing field and set
     *  option contextSuperClass.
     *  to set it.
     *
     *  @since 4.5.3
     */
    get altNumber(): number;
    /** Set the outer alternative number for this context node. Default
     *  implementation does nothing to avoid backing field overhead for
     *  trees that don't need it.  Create
     *  a subclass of ParserRuleContext with backing field and set
     *  option contextSuperClass.
     *
     *  @since 4.5.3
     */
    set altNumber(altNumber: number);
    getChild(i: number): ParseTree;
    get childCount(): number;
    accept<T>(visitor: ParseTreeVisitor<T>): T;
    /** Print out a whole tree, not just a node, in LISP format
     *  (root child1 .. childN). Print just a node if this is a leaf.
     *  We have to know the recognizer so we can get rule names.
     */
    toStringTree(recog: Parser): string;
    /** Print out a whole tree, not just a node, in LISP format
     *  (root child1 .. childN). Print just a node if this is a leaf.
     */
    toStringTree(ruleNames: string[] | undefined): string;
    toStringTree(): string;
    toString(): string;
    toString(recog: Recognizer<any, any> | undefined): string;
    toString(ruleNames: string[] | undefined): string;
    toString(recog: Recognizer<any, any> | undefined, stop: RuleContext | undefined): string;
    toString(ruleNames: string[] | undefined, stop: RuleContext | undefined): string;
}
