"use strict";
/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
var __decorate = (this && this.__decorate) || function (decorators, target, key, desc) {
    var c = arguments.length, r = c < 3 ? target : desc === null ? desc = Object.getOwnPropertyDescriptor(target, key) : desc, d;
    if (typeof Reflect === "object" && typeof Reflect.decorate === "function") r = Reflect.decorate(decorators, target, key, desc);
    else for (var i = decorators.length - 1; i >= 0; i--) if (d = decorators[i]) r = (c < 3 ? d(r) : c > 3 ? d(target, key, r) : d(target, key)) || r;
    return c > 3 && r && Object.defineProperty(target, key, r), r;
};
var __param = (this && this.__param) || function (paramIndex, decorator) {
    return function (target, key) { decorator(target, key, paramIndex); }
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.ParserInterpreter = void 0;
const ATNState_1 = require("./atn/ATNState");
const ATNStateType_1 = require("./atn/ATNStateType");
const BitSet_1 = require("./misc/BitSet");
const FailedPredicateException_1 = require("./FailedPredicateException");
const InputMismatchException_1 = require("./InputMismatchException");
const InterpreterRuleContext_1 = require("./InterpreterRuleContext");
const LoopEndState_1 = require("./atn/LoopEndState");
const Decorators_1 = require("./Decorators");
const Decorators_2 = require("./Decorators");
const Parser_1 = require("./Parser");
const ParserATNSimulator_1 = require("./atn/ParserATNSimulator");
const RecognitionException_1 = require("./RecognitionException");
const StarLoopEntryState_1 = require("./atn/StarLoopEntryState");
const Token_1 = require("./Token");
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
let ParserInterpreter = class ParserInterpreter extends Parser_1.Parser {
    constructor(grammarFileName, vocabulary, ruleNames, atn, input) {
        super(grammarFileName instanceof ParserInterpreter ? grammarFileName.inputStream : input);
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
        this._parentContextStack = [];
        /** We need a map from (decision,inputIndex)->forced alt for computing ambiguous
         *  parse trees. For now, we allow exactly one override.
         */
        this.overrideDecision = -1;
        this.overrideDecisionInputIndex = -1;
        this.overrideDecisionAlt = -1;
        this.overrideDecisionReached = false; // latch and only override once; error might trigger infinite loop
        /** What is the current context when we override a decisions?  This tells
         *  us what the root of the parse tree is when using override
         *  for an ambiguity/lookahead check.
         */
        this._overrideDecisionRoot = undefined;
        if (grammarFileName instanceof ParserInterpreter) {
            let old = grammarFileName;
            this._grammarFileName = old._grammarFileName;
            this._atn = old._atn;
            this.pushRecursionContextStates = old.pushRecursionContextStates;
            this._ruleNames = old._ruleNames;
            this._vocabulary = old._vocabulary;
            this.interpreter = new ParserATNSimulator_1.ParserATNSimulator(this._atn, this);
        }
        else {
            // The second constructor requires non-null arguments
            vocabulary = vocabulary;
            ruleNames = ruleNames;
            atn = atn;
            this._grammarFileName = grammarFileName;
            this._atn = atn;
            this._ruleNames = ruleNames.slice(0);
            this._vocabulary = vocabulary;
            // identify the ATN states where pushNewRecursionContext() must be called
            this.pushRecursionContextStates = new BitSet_1.BitSet(atn.states.length);
            for (let state of atn.states) {
                if (!(state instanceof StarLoopEntryState_1.StarLoopEntryState)) {
                    continue;
                }
                if (state.precedenceRuleDecision) {
                    this.pushRecursionContextStates.set(state.stateNumber);
                }
            }
            // get atn simulator that knows how to do predictions
            this.interpreter = new ParserATNSimulator_1.ParserATNSimulator(atn, this);
        }
    }
    reset(resetInput) {
        if (resetInput === undefined) {
            super.reset();
        }
        else {
            super.reset(resetInput);
        }
        this.overrideDecisionReached = false;
        this._overrideDecisionRoot = undefined;
    }
    get atn() {
        return this._atn;
    }
    get vocabulary() {
        return this._vocabulary;
    }
    get ruleNames() {
        return this._ruleNames;
    }
    get grammarFileName() {
        return this._grammarFileName;
    }
    /** Begin parsing at startRuleIndex */
    parse(startRuleIndex) {
        let startRuleStartState = this._atn.ruleToStartState[startRuleIndex];
        this._rootContext = this.createInterpreterRuleContext(undefined, ATNState_1.ATNState.INVALID_STATE_NUMBER, startRuleIndex);
        if (startRuleStartState.isPrecedenceRule) {
            this.enterRecursionRule(this._rootContext, startRuleStartState.stateNumber, startRuleIndex, 0);
        }
        else {
            this.enterRule(this._rootContext, startRuleStartState.stateNumber, startRuleIndex);
        }
        while (true) {
            let p = this.atnState;
            switch (p.stateType) {
                case ATNStateType_1.ATNStateType.RULE_STOP:
                    // pop; return from rule
                    if (this._ctx.isEmpty) {
                        if (startRuleStartState.isPrecedenceRule) {
                            let result = this._ctx;
                            let parentContext = this._parentContextStack.pop();
                            this.unrollRecursionContexts(parentContext[0]);
                            return result;
                        }
                        else {
                            this.exitRule();
                            return this._rootContext;
                        }
                    }
                    this.visitRuleStopState(p);
                    break;
                default:
                    try {
                        this.visitState(p);
                    }
                    catch (e) {
                        if (e instanceof RecognitionException_1.RecognitionException) {
                            this.state = this._atn.ruleToStopState[p.ruleIndex].stateNumber;
                            this.context.exception = e;
                            this.errorHandler.reportError(this, e);
                            this.recover(e);
                        }
                        else {
                            throw e;
                        }
                    }
                    break;
            }
        }
    }
    enterRecursionRule(localctx, state, ruleIndex, precedence) {
        this._parentContextStack.push([this._ctx, localctx.invokingState]);
        super.enterRecursionRule(localctx, state, ruleIndex, precedence);
    }
    get atnState() {
        return this._atn.states[this.state];
    }
    visitState(p) {
        let predictedAlt = 1;
        if (p.numberOfTransitions > 1) {
            predictedAlt = this.visitDecisionState(p);
        }
        let transition = p.transition(predictedAlt - 1);
        switch (transition.serializationType) {
            case 1 /* EPSILON */:
                if (this.pushRecursionContextStates.get(p.stateNumber) &&
                    !(transition.target instanceof LoopEndState_1.LoopEndState)) {
                    // We are at the start of a left recursive rule's (...)* loop
                    // and we're not taking the exit branch of loop.
                    let parentContext = this._parentContextStack[this._parentContextStack.length - 1];
                    let localctx = this.createInterpreterRuleContext(parentContext[0], parentContext[1], this._ctx.ruleIndex);
                    this.pushNewRecursionContext(localctx, this._atn.ruleToStartState[p.ruleIndex].stateNumber, this._ctx.ruleIndex);
                }
                break;
            case 5 /* ATOM */:
                this.match(transition._label);
                break;
            case 2 /* RANGE */:
            case 7 /* SET */:
            case 8 /* NOT_SET */:
                if (!transition.matches(this._input.LA(1), Token_1.Token.MIN_USER_TOKEN_TYPE, 65535)) {
                    this.recoverInline();
                }
                this.matchWildcard();
                break;
            case 9 /* WILDCARD */:
                this.matchWildcard();
                break;
            case 3 /* RULE */:
                let ruleStartState = transition.target;
                let ruleIndex = ruleStartState.ruleIndex;
                let newctx = this.createInterpreterRuleContext(this._ctx, p.stateNumber, ruleIndex);
                if (ruleStartState.isPrecedenceRule) {
                    this.enterRecursionRule(newctx, ruleStartState.stateNumber, ruleIndex, transition.precedence);
                }
                else {
                    this.enterRule(newctx, transition.target.stateNumber, ruleIndex);
                }
                break;
            case 4 /* PREDICATE */:
                let predicateTransition = transition;
                if (!this.sempred(this._ctx, predicateTransition.ruleIndex, predicateTransition.predIndex)) {
                    throw new FailedPredicateException_1.FailedPredicateException(this);
                }
                break;
            case 6 /* ACTION */:
                let actionTransition = transition;
                this.action(this._ctx, actionTransition.ruleIndex, actionTransition.actionIndex);
                break;
            case 10 /* PRECEDENCE */:
                if (!this.precpred(this._ctx, transition.precedence)) {
                    let precedence = transition.precedence;
                    throw new FailedPredicateException_1.FailedPredicateException(this, `precpred(_ctx, ${precedence})`);
                }
                break;
            default:
                throw new Error("UnsupportedOperationException: Unrecognized ATN transition type.");
        }
        this.state = transition.target.stateNumber;
    }
    /** Method visitDecisionState() is called when the interpreter reaches
     *  a decision state (instance of DecisionState). It gives an opportunity
     *  for subclasses to track interesting things.
     */
    visitDecisionState(p) {
        let predictedAlt;
        this.errorHandler.sync(this);
        let decision = p.decision;
        if (decision === this.overrideDecision && this._input.index === this.overrideDecisionInputIndex && !this.overrideDecisionReached) {
            predictedAlt = this.overrideDecisionAlt;
            this.overrideDecisionReached = true;
        }
        else {
            predictedAlt = this.interpreter.adaptivePredict(this._input, decision, this._ctx);
        }
        return predictedAlt;
    }
    /** Provide simple "factory" for InterpreterRuleContext's.
     *  @since 4.5.1
     */
    createInterpreterRuleContext(parent, invokingStateNumber, ruleIndex) {
        return new InterpreterRuleContext_1.InterpreterRuleContext(ruleIndex, parent, invokingStateNumber);
    }
    visitRuleStopState(p) {
        let ruleStartState = this._atn.ruleToStartState[p.ruleIndex];
        if (ruleStartState.isPrecedenceRule) {
            let parentContext = this._parentContextStack.pop();
            this.unrollRecursionContexts(parentContext[0]);
            this.state = parentContext[1];
        }
        else {
            this.exitRule();
        }
        let ruleTransition = this._atn.states[this.state].transition(0);
        this.state = ruleTransition.followState.stateNumber;
    }
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
    addDecisionOverride(decision, tokenIndex, forcedAlt) {
        this.overrideDecision = decision;
        this.overrideDecisionInputIndex = tokenIndex;
        this.overrideDecisionAlt = forcedAlt;
    }
    get overrideDecisionRoot() {
        return this._overrideDecisionRoot;
    }
    /** Rely on the error handler for this parser but, if no tokens are consumed
     *  to recover, add an error node. Otherwise, nothing is seen in the parse
     *  tree.
     */
    recover(e) {
        let i = this._input.index;
        this.errorHandler.recover(this, e);
        if (this._input.index === i) {
            // no input consumed, better add an error node
            let tok = e.getOffendingToken();
            if (!tok) {
                throw new Error("Expected exception to have an offending token");
            }
            let source = tok.tokenSource;
            let stream = source !== undefined ? source.inputStream : undefined;
            let sourcePair = { source, stream };
            if (e instanceof InputMismatchException_1.InputMismatchException) {
                let expectedTokens = e.expectedTokens;
                if (expectedTokens === undefined) {
                    throw new Error("Expected the exception to provide expected tokens");
                }
                let expectedTokenType = Token_1.Token.INVALID_TYPE;
                if (!expectedTokens.isNil) {
                    // get any element
                    expectedTokenType = expectedTokens.minElement;
                }
                let errToken = this.tokenFactory.create(sourcePair, expectedTokenType, tok.text, Token_1.Token.DEFAULT_CHANNEL, -1, -1, // invalid start/stop
                tok.line, tok.charPositionInLine);
                this._ctx.addErrorNode(this.createErrorNode(this._ctx, errToken));
            }
            else { // NoViableAlt
                let source = tok.tokenSource;
                let errToken = this.tokenFactory.create(sourcePair, Token_1.Token.INVALID_TYPE, tok.text, Token_1.Token.DEFAULT_CHANNEL, -1, -1, // invalid start/stop
                tok.line, tok.charPositionInLine);
                this._ctx.addErrorNode(this.createErrorNode(this._ctx, errToken));
            }
        }
    }
    recoverInline() {
        return this._errHandler.recoverInline(this);
    }
    /** Return the root of the parse, which can be useful if the parser
     *  bails out. You still can access the top node. Note that,
     *  because of the way left recursive rules add children, it's possible
     *  that the root will not have any children if the start rule immediately
     *  called and left recursive rule that fails.
     *
     * @since 4.5.1
     */
    get rootContext() {
        return this._rootContext;
    }
};
__decorate([
    Decorators_1.NotNull
], ParserInterpreter.prototype, "_vocabulary", void 0);
__decorate([
    Decorators_2.Override
], ParserInterpreter.prototype, "reset", null);
__decorate([
    Decorators_2.Override
], ParserInterpreter.prototype, "atn", null);
__decorate([
    Decorators_2.Override
], ParserInterpreter.prototype, "vocabulary", null);
__decorate([
    Decorators_2.Override
], ParserInterpreter.prototype, "ruleNames", null);
__decorate([
    Decorators_2.Override
], ParserInterpreter.prototype, "grammarFileName", null);
__decorate([
    Decorators_2.Override
], ParserInterpreter.prototype, "enterRecursionRule", null);
ParserInterpreter = __decorate([
    __param(1, Decorators_1.NotNull)
], ParserInterpreter);
exports.ParserInterpreter = ParserInterpreter;
//# sourceMappingURL=ParserInterpreter.js.map