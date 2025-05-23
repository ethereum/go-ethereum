/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ATN } from "./atn/ATN";
import { ATNState } from "./atn/ATNState";
import { BitSet } from "./misc/BitSet";
import { DecisionState } from "./atn/DecisionState";
import { InterpreterRuleContext } from "./InterpreterRuleContext";
import { Parser } from "./Parser";
import { ParserRuleContext } from "./ParserRuleContext";
import { RecognitionException } from "./RecognitionException";
import { Token } from "./Token";
import { TokenStream } from "./TokenStream";
import { Vocabulary } from "./Vocabulary";
/** A parser simulator that mimics what ANTLR's generated
 *  parser code does. A ParserATNSimulator is used to make
 *  predictions via adaptivePredict but this class moves a pointer through the
 *  ATN to simulate parsing. ParserATNSimulator just
 *  makes us efficient rather than having to backtrack, for example.
 *
 *  This properly creates parse trees even for left recursive rules.
 *
 *  We rely on the left recursive rule invocation and special predicate
 *  transitions to make left recursive rules work.
 *
 *  See TestParserInterpreter for examples.
 */
export declare class ParserInterpreter extends Parser {
    protected _grammarFileName: string;
    protected _atn: ATN;
    /** This identifies StarLoopEntryState's that begin the (...)*
     *  precedence loops of left recursive rules.
     */
    protected pushRecursionContextStates: BitSet;
    protected _ruleNames: string[];
    private _vocabulary;
    /** This stack corresponds to the _parentctx, _parentState pair of locals
     *  that would exist on call stack frames with a recursive descent parser;
     *  in the generated function for a left-recursive rule you'd see:
     *
     *  private EContext e(int _p) {
     *      ParserRuleContext _parentctx = _ctx;    // Pair.a
     *      int _parentState = state;          // Pair.b
     *      ...
     *  }
     *
     *  Those values are used to create new recursive rule invocation contexts
     *  associated with left operand of an alt like "expr '*' expr".
     */
    protected readonly _parentContextStack: Array<[ParserRuleContext, number]>;
    /** We need a map from (decision,inputIndex)->forced alt for computing ambiguous
     *  parse trees. For now, we allow exactly one override.
     */
    protected overrideDecision: number;
    protected overrideDecisionInputIndex: number;
    protected overrideDecisionAlt: number;
    protected overrideDecisionReached: boolean;
    /** What is the current context when we override a decisions?  This tells
     *  us what the root of the parse tree is when using override
     *  for an ambiguity/lookahead check.
     */
    protected _overrideDecisionRoot?: InterpreterRuleContext;
    protected _rootContext: InterpreterRuleContext;
    /** A copy constructor that creates a new parser interpreter by reusing
     *  the fields of a previous interpreter.
     *
     *  @param old The interpreter to copy
     *
     *  @since 4.5
     */
    constructor(/*@NotNull*/ old: ParserInterpreter);
    constructor(grammarFileName: string, /*@NotNull*/ vocabulary: Vocabulary, ruleNames: string[], atn: ATN, input: TokenStream);
    reset(resetInput?: boolean): void;
    get atn(): ATN;
    get vocabulary(): Vocabulary;
    get ruleNames(): string[];
    get grammarFileName(): string;
    /** Begin parsing at startRuleIndex */
    parse(startRuleIndex: number): ParserRuleContext;
    enterRecursionRule(localctx: ParserRuleContext, state: number, ruleIndex: number, precedence: number): void;
    protected get atnState(): ATNState;
    protected visitState(p: ATNState): void;
    /** Method visitDecisionState() is called when the interpreter reaches
     *  a decision state (instance of DecisionState). It gives an opportunity
     *  for subclasses to track interesting things.
     */
    protected visitDecisionState(p: DecisionState): number;
    /** Provide simple "factory" for InterpreterRuleContext's.
     *  @since 4.5.1
     */
    protected createInterpreterRuleContext(parent: ParserRuleContext | undefined, invokingStateNumber: number, ruleIndex: number): InterpreterRuleContext;
    protected visitRuleStopState(p: ATNState): void;
    /** Override this parser interpreters normal decision-making process
     *  at a particular decision and input token index. Instead of
     *  allowing the adaptive prediction mechanism to choose the
     *  first alternative within a block that leads to a successful parse,
     *  force it to take the alternative, 1..n for n alternatives.
     *
     *  As an implementation limitation right now, you can only specify one
     *  override. This is sufficient to allow construction of different
     *  parse trees for ambiguous input. It means re-parsing the entire input
     *  in general because you're never sure where an ambiguous sequence would
     *  live in the various parse trees. For example, in one interpretation,
     *  an ambiguous input sequence would be matched completely in expression
     *  but in another it could match all the way back to the root.
     *
     *  s : e '!'? ;
     *  e : ID
     *    | ID '!'
     *    ;
     *
     *  Here, x! can be matched as (s (e ID) !) or (s (e ID !)). In the first
     *  case, the ambiguous sequence is fully contained only by the root.
     *  In the second case, the ambiguous sequences fully contained within just
     *  e, as in: (e ID !).
     *
     *  Rather than trying to optimize this and make
     *  some intelligent decisions for optimization purposes, I settled on
     *  just re-parsing the whole input and then using
     *  {link Trees#getRootOfSubtreeEnclosingRegion} to find the minimal
     *  subtree that contains the ambiguous sequence. I originally tried to
     *  record the call stack at the point the parser detected and ambiguity but
     *  left recursive rules create a parse tree stack that does not reflect
     *  the actual call stack. That impedance mismatch was enough to make
     *  it it challenging to restart the parser at a deeply nested rule
     *  invocation.
     *
     *  Only parser interpreters can override decisions so as to avoid inserting
     *  override checking code in the critical ALL(*) prediction execution path.
     *
     *  @since 4.5
     */
    addDecisionOverride(decision: number, tokenIndex: number, forcedAlt: number): void;
    get overrideDecisionRoot(): InterpreterRuleContext | undefined;
    /** Rely on the error handler for this parser but, if no tokens are consumed
     *  to recover, add an error node. Otherwise, nothing is seen in the parse
     *  tree.
     */
    protected recover(e: RecognitionException): void;
    protected recoverInline(): Token;
    /** Return the root of the parse, which can be useful if the parser
     *  bails out. You still can access the top node. Note that,
     *  because of the way left recursive rules add children, it's possible
     *  that the root will not have any children if the start rule immediately
     *  called and left recursive rule that fails.
     *
     * @since 4.5.1
     */
    get rootContext(): InterpreterRuleContext;
}
