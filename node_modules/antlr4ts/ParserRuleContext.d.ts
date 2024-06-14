/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ErrorNode } from "./tree/ErrorNode";
import { Interval } from "./misc/Interval";
import { Parser } from "./Parser";
import { ParseTree } from "./tree/ParseTree";
import { ParseTreeListener } from "./tree/ParseTreeListener";
import { RecognitionException } from "./RecognitionException";
import { RuleContext } from "./RuleContext";
import { TerminalNode } from "./tree/TerminalNode";
import { Token } from "./Token";
/** A rule invocation record for parsing.
 *
 *  Contains all of the information about the current rule not stored in the
 *  RuleContext. It handles parse tree children list, Any ATN state
 *  tracing, and the default values available for rule invocations:
 *  start, stop, rule index, current alt number.
 *
 *  Subclasses made for each rule and grammar track the parameters,
 *  return values, locals, and labels specific to that rule. These
 *  are the objects that are returned from rules.
 *
 *  Note text is not an actual field of a rule return value; it is computed
 *  from start and stop using the input stream's toString() method.  I
 *  could add a ctor to this so that we can pass in and store the input
 *  stream, but I'm not sure we want to do that.  It would seem to be undefined
 *  to get the .text property anyway if the rule matches tokens from multiple
 *  input streams.
 *
 *  I do not use getters for fields of objects that are used simply to
 *  group values such as this aggregate.  The getters/setters are there to
 *  satisfy the superclass interface.
 */
export declare class ParserRuleContext extends RuleContext {
    private static readonly EMPTY;
    /** If we are debugging or building a parse tree for a visitor,
     *  we need to track all of the tokens and rule invocations associated
     *  with this rule's context. This is empty for parsing w/o tree constr.
     *  operation because we don't the need to track the details about
     *  how we parse this rule.
     */
    children?: ParseTree[];
    /** For debugging/tracing purposes, we want to track all of the nodes in
     *  the ATN traversed by the parser for a particular rule.
     *  This list indicates the sequence of ATN nodes used to match
     *  the elements of the children list. This list does not include
     *  ATN nodes and other rules used to match rule invocations. It
     *  traces the rule invocation node itself but nothing inside that
     *  other rule's ATN submachine.
     *
     *  There is NOT a one-to-one correspondence between the children and
     *  states list. There are typically many nodes in the ATN traversed
     *  for each element in the children list. For example, for a rule
     *  invocation there is the invoking state and the following state.
     *
     *  The parser state property updates field s and adds it to this list
     *  if we are debugging/tracing.
     *
     *  This does not trace states visited during prediction.
     */
    _start: Token;
    _stop: Token | undefined;
    /**
     * The exception that forced this rule to return. If the rule successfully
     * completed, this is `undefined`.
     */
    exception?: RecognitionException;
    constructor();
    constructor(parent: ParserRuleContext | undefined, invokingStateNumber: number);
    static emptyContext(): ParserRuleContext;
    /**
     * COPY a ctx (I'm deliberately not using copy constructor) to avoid
     * confusion with creating node with parent. Does not copy children
     * (except error leaves).
     *
     * This is used in the generated parser code to flip a generic XContext
     * node for rule X to a YContext for alt label Y. In that sense, it is not
     * really a generic copy function.
     *
     * If we do an error sync() at start of a rule, we might add error nodes
     * to the generic XContext so this function must copy those nodes to the
     * YContext as well else they are lost!
     */
    copyFrom(ctx: ParserRuleContext): void;
    enterRule(listener: ParseTreeListener): void;
    exitRule(listener: ParseTreeListener): void;
    /** Add a parse tree node to this as a child.  Works for
     *  internal and leaf nodes. Does not set parent link;
     *  other add methods must do that. Other addChild methods
     *  call this.
     *
     *  We cannot set the parent pointer of the incoming node
     *  because the existing interfaces do not have a setParent()
     *  method and I don't want to break backward compatibility for this.
     *
     *  @since 4.7
     */
    addAnyChild<T extends ParseTree>(t: T): T;
    /** Add a token leaf node child and force its parent to be this node. */
    addChild(t: TerminalNode): void;
    addChild(ruleInvocation: RuleContext): void;
    /**
     * Add a child to this node based upon matchedToken. It
     * creates a TerminalNodeImpl rather than using
     * {@link Parser#createTerminalNode(ParserRuleContext, Token)}. I'm leaving this
     * in for compatibility but the parser doesn't use this anymore.
     *
     * @deprecated Use another overload instead.
     */
    addChild(matchedToken: Token): TerminalNode;
    /** Add an error node child and force its parent to be this node.
     *
     * @since 4.7
     */
    addErrorNode(errorNode: ErrorNode): ErrorNode;
    /**
     * Add a child to this node based upon badToken. It
     * creates a ErrorNode rather than using
     * {@link Parser#createErrorNode(ParserRuleContext, Token)}. I'm leaving this
     * in for compatibility but the parser doesn't use this anymore.
     *
     * @deprecated Use another overload instead.
     */
    addErrorNode(badToken: Token): ErrorNode;
    /** Used by enterOuterAlt to toss out a RuleContext previously added as
     *  we entered a rule. If we have # label, we will need to remove
     *  generic ruleContext object.
     */
    removeLastChild(): void;
    get parent(): ParserRuleContext | undefined;
    getChild(i: number): ParseTree;
    getChild<T extends ParseTree>(i: number, ctxType: {
        new (...args: any[]): T;
    }): T;
    tryGetChild<T extends ParseTree>(i: number, ctxType: {
        new (...args: any[]): T;
    }): T | undefined;
    getToken(ttype: number, i: number): TerminalNode;
    tryGetToken(ttype: number, i: number): TerminalNode | undefined;
    getTokens(ttype: number): TerminalNode[];
    get ruleContext(): this;
    getRuleContext<T extends ParserRuleContext>(i: number, ctxType: {
        new (...args: any[]): T;
    }): T;
    tryGetRuleContext<T extends ParserRuleContext>(i: number, ctxType: {
        new (...args: any[]): T;
    }): T | undefined;
    getRuleContexts<T extends ParserRuleContext>(ctxType: {
        new (...args: any[]): T;
    }): T[];
    get childCount(): number;
    get sourceInterval(): Interval;
    /**
     * Get the initial token in this context.
     * Note that the range from start to stop is inclusive, so for rules that do not consume anything
     * (for example, zero length or error productions) this token may exceed stop.
     */
    get start(): Token;
    /**
     * Get the final token in this context.
     * Note that the range from start to stop is inclusive, so for rules that do not consume anything
     * (for example, zero length or error productions) this token may precede start.
     */
    get stop(): Token | undefined;
    /** Used for rule context info debugging during parse-time, not so much for ATN debugging */
    toInfoString(recognizer: Parser): string;
}
