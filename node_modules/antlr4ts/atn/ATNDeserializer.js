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
exports.ATNDeserializer = void 0;
// ConvertTo-TS run at 2016-10-04T11:26:25.9683447-07:00
const ActionTransition_1 = require("./ActionTransition");
const Array2DHashSet_1 = require("../misc/Array2DHashSet");
const ATN_1 = require("./ATN");
const ATNDeserializationOptions_1 = require("./ATNDeserializationOptions");
const ATNStateType_1 = require("./ATNStateType");
const AtomTransition_1 = require("./AtomTransition");
const BasicBlockStartState_1 = require("./BasicBlockStartState");
const BasicState_1 = require("./BasicState");
const BitSet_1 = require("../misc/BitSet");
const BlockEndState_1 = require("./BlockEndState");
const BlockStartState_1 = require("./BlockStartState");
const DecisionState_1 = require("./DecisionState");
const DFA_1 = require("../dfa/DFA");
const EpsilonTransition_1 = require("./EpsilonTransition");
const IntervalSet_1 = require("../misc/IntervalSet");
const InvalidState_1 = require("./InvalidState");
const LexerChannelAction_1 = require("./LexerChannelAction");
const LexerCustomAction_1 = require("./LexerCustomAction");
const LexerModeAction_1 = require("./LexerModeAction");
const LexerMoreAction_1 = require("./LexerMoreAction");
const LexerPopModeAction_1 = require("./LexerPopModeAction");
const LexerPushModeAction_1 = require("./LexerPushModeAction");
const LexerSkipAction_1 = require("./LexerSkipAction");
const LexerTypeAction_1 = require("./LexerTypeAction");
const LoopEndState_1 = require("./LoopEndState");
const Decorators_1 = require("../Decorators");
const NotSetTransition_1 = require("./NotSetTransition");
const ParserATNSimulator_1 = require("./ParserATNSimulator");
const PlusBlockStartState_1 = require("./PlusBlockStartState");
const PlusLoopbackState_1 = require("./PlusLoopbackState");
const PrecedencePredicateTransition_1 = require("./PrecedencePredicateTransition");
const PredicateTransition_1 = require("./PredicateTransition");
const RangeTransition_1 = require("./RangeTransition");
const RuleStartState_1 = require("./RuleStartState");
const RuleStopState_1 = require("./RuleStopState");
const RuleTransition_1 = require("./RuleTransition");
const SetTransition_1 = require("./SetTransition");
const StarBlockStartState_1 = require("./StarBlockStartState");
const StarLoopbackState_1 = require("./StarLoopbackState");
const StarLoopEntryState_1 = require("./StarLoopEntryState");
const Token_1 = require("../Token");
const TokensStartState_1 = require("./TokensStartState");
const UUID_1 = require("../misc/UUID");
const WildcardTransition_1 = require("./WildcardTransition");
var UnicodeDeserializingMode;
(function (UnicodeDeserializingMode) {
    UnicodeDeserializingMode[UnicodeDeserializingMode["UNICODE_BMP"] = 0] = "UNICODE_BMP";
    UnicodeDeserializingMode[UnicodeDeserializingMode["UNICODE_SMP"] = 1] = "UNICODE_SMP";
})(UnicodeDeserializingMode || (UnicodeDeserializingMode = {}));
/**
 *
 * @author Sam Harwell
 */
class ATNDeserializer {
    constructor(deserializationOptions) {
        if (deserializationOptions === undefined) {
            deserializationOptions = ATNDeserializationOptions_1.ATNDeserializationOptions.defaultOptions;
        }
        this.deserializationOptions = deserializationOptions;
    }
    static get SERIALIZED_VERSION() {
        /* This value should never change. Updates following this version are
         * reflected as change in the unique ID SERIALIZED_UUID.
         */
        return 3;
    }
    /**
     * Determines if a particular serialized representation of an ATN supports
     * a particular feature, identified by the {@link UUID} used for serializing
     * the ATN at the time the feature was first introduced.
     *
     * @param feature The {@link UUID} marking the first time the feature was
     * supported in the serialized ATN.
     * @param actualUuid The {@link UUID} of the actual serialized ATN which is
     * currently being deserialized.
     * @returns `true` if the `actualUuid` value represents a
     * serialized ATN at or after the feature identified by `feature` was
     * introduced; otherwise, `false`.
     */
    static isFeatureSupported(feature, actualUuid) {
        let featureIndex = ATNDeserializer.SUPPORTED_UUIDS.findIndex((e) => e.equals(feature));
        if (featureIndex < 0) {
            return false;
        }
        return ATNDeserializer.SUPPORTED_UUIDS.findIndex((e) => e.equals(actualUuid)) >= featureIndex;
    }
    static getUnicodeDeserializer(mode) {
        if (mode === 0 /* UNICODE_BMP */) {
            return {
                readUnicode: (data, p) => {
                    return ATNDeserializer.toInt(data[p]);
                },
                size: 1,
            };
        }
        else {
            return {
                readUnicode: (data, p) => {
                    return ATNDeserializer.toInt32(data, p);
                },
                size: 2,
            };
        }
    }
    deserialize(data) {
        data = data.slice(0);
        // Each Uint16 value in data is shifted by +2 at the entry to this method. This is an encoding optimization
        // targeting the serialized values 0 and -1 (serialized to 0xFFFF), each of which are very common in the
        // serialized form of the ATN. In the modified UTF-8 that Java uses for compiled string literals, these two
        // character values have multi-byte forms. By shifting each value by +2, they become characters 2 and 1 prior to
        // writing the string, each of which have single-byte representations. Since the shift occurs in the tool during
        // ATN serialization, each target is responsible for adjusting the values during deserialization.
        //
        // As a special case, note that the first element of data is not adjusted because it contains the major version
        // number of the serialized ATN, which was fixed at 3 at the time the value shifting was implemented.
        for (let i = 1; i < data.length; i++) {
            data[i] = (data[i] - 2) & 0xFFFF;
        }
        let p = 0;
        let version = ATNDeserializer.toInt(data[p++]);
        if (version !== ATNDeserializer.SERIALIZED_VERSION) {
            let reason = `Could not deserialize ATN with version ${version} (expected ${ATNDeserializer.SERIALIZED_VERSION}).`;
            throw new Error(reason);
        }
        let uuid = ATNDeserializer.toUUID(data, p);
        p += 8;
        if (ATNDeserializer.SUPPORTED_UUIDS.findIndex((e) => e.equals(uuid)) < 0) {
            let reason = `Could not deserialize ATN with UUID ${uuid} (expected ${ATNDeserializer.SERIALIZED_UUID} or a legacy UUID).`;
            throw new Error(reason);
        }
        let supportsLexerActions = ATNDeserializer.isFeatureSupported(ATNDeserializer.ADDED_LEXER_ACTIONS, uuid);
        let grammarType = ATNDeserializer.toInt(data[p++]);
        let maxTokenType = ATNDeserializer.toInt(data[p++]);
        let atn = new ATN_1.ATN(grammarType, maxTokenType);
        //
        // STATES
        //
        let loopBackStateNumbers = [];
        let endStateNumbers = [];
        let nstates = ATNDeserializer.toInt(data[p++]);
        for (let i = 0; i < nstates; i++) {
            let stype = ATNDeserializer.toInt(data[p++]);
            // ignore bad type of states
            if (stype === ATNStateType_1.ATNStateType.INVALID_TYPE) {
                atn.addState(new InvalidState_1.InvalidState());
                continue;
            }
            let ruleIndex = ATNDeserializer.toInt(data[p++]);
            if (ruleIndex === 0xFFFF) {
                ruleIndex = -1;
            }
            let s = this.stateFactory(stype, ruleIndex);
            if (stype === ATNStateType_1.ATNStateType.LOOP_END) { // special case
                let loopBackStateNumber = ATNDeserializer.toInt(data[p++]);
                loopBackStateNumbers.push([s, loopBackStateNumber]);
            }
            else if (s instanceof BlockStartState_1.BlockStartState) {
                let endStateNumber = ATNDeserializer.toInt(data[p++]);
                endStateNumbers.push([s, endStateNumber]);
            }
            atn.addState(s);
        }
        // delay the assignment of loop back and end states until we know all the state instances have been initialized
        for (let pair of loopBackStateNumbers) {
            pair[0].loopBackState = atn.states[pair[1]];
        }
        for (let pair of endStateNumbers) {
            pair[0].endState = atn.states[pair[1]];
        }
        let numNonGreedyStates = ATNDeserializer.toInt(data[p++]);
        for (let i = 0; i < numNonGreedyStates; i++) {
            let stateNumber = ATNDeserializer.toInt(data[p++]);
            atn.states[stateNumber].nonGreedy = true;
        }
        let numSllDecisions = ATNDeserializer.toInt(data[p++]);
        for (let i = 0; i < numSllDecisions; i++) {
            let stateNumber = ATNDeserializer.toInt(data[p++]);
            atn.states[stateNumber].sll = true;
        }
        let numPrecedenceStates = ATNDeserializer.toInt(data[p++]);
        for (let i = 0; i < numPrecedenceStates; i++) {
            let stateNumber = ATNDeserializer.toInt(data[p++]);
            atn.states[stateNumber].isPrecedenceRule = true;
        }
        //
        // RULES
        //
        let nrules = ATNDeserializer.toInt(data[p++]);
        if (atn.grammarType === 0 /* LEXER */) {
            atn.ruleToTokenType = new Int32Array(nrules);
        }
        atn.ruleToStartState = new Array(nrules);
        for (let i = 0; i < nrules; i++) {
            let s = ATNDeserializer.toInt(data[p++]);
            let startState = atn.states[s];
            startState.leftFactored = ATNDeserializer.toInt(data[p++]) !== 0;
            atn.ruleToStartState[i] = startState;
            if (atn.grammarType === 0 /* LEXER */) {
                let tokenType = ATNDeserializer.toInt(data[p++]);
                if (tokenType === 0xFFFF) {
                    tokenType = Token_1.Token.EOF;
                }
                atn.ruleToTokenType[i] = tokenType;
                if (!ATNDeserializer.isFeatureSupported(ATNDeserializer.ADDED_LEXER_ACTIONS, uuid)) {
                    // this piece of unused metadata was serialized prior to the
                    // addition of LexerAction
                    let actionIndexIgnored = ATNDeserializer.toInt(data[p++]);
                    if (actionIndexIgnored === 0xFFFF) {
                        actionIndexIgnored = -1;
                    }
                }
            }
        }
        atn.ruleToStopState = new Array(nrules);
        for (let state of atn.states) {
            if (!(state instanceof RuleStopState_1.RuleStopState)) {
                continue;
            }
            atn.ruleToStopState[state.ruleIndex] = state;
            atn.ruleToStartState[state.ruleIndex].stopState = state;
        }
        //
        // MODES
        //
        let nmodes = ATNDeserializer.toInt(data[p++]);
        for (let i = 0; i < nmodes; i++) {
            let s = ATNDeserializer.toInt(data[p++]);
            atn.modeToStartState.push(atn.states[s]);
        }
        atn.modeToDFA = new Array(nmodes);
        for (let i = 0; i < nmodes; i++) {
            atn.modeToDFA[i] = new DFA_1.DFA(atn.modeToStartState[i]);
        }
        //
        // SETS
        //
        let sets = [];
        // First, read all sets with 16-bit Unicode code points <= U+FFFF.
        p = this.deserializeSets(data, p, sets, ATNDeserializer.getUnicodeDeserializer(0 /* UNICODE_BMP */));
        // Next, if the ATN was serialized with the Unicode SMP feature,
        // deserialize sets with 32-bit arguments <= U+10FFFF.
        if (ATNDeserializer.isFeatureSupported(ATNDeserializer.ADDED_UNICODE_SMP, uuid)) {
            p = this.deserializeSets(data, p, sets, ATNDeserializer.getUnicodeDeserializer(1 /* UNICODE_SMP */));
        }
        //
        // EDGES
        //
        let nedges = ATNDeserializer.toInt(data[p++]);
        for (let i = 0; i < nedges; i++) {
            let src = ATNDeserializer.toInt(data[p]);
            let trg = ATNDeserializer.toInt(data[p + 1]);
            let ttype = ATNDeserializer.toInt(data[p + 2]);
            let arg1 = ATNDeserializer.toInt(data[p + 3]);
            let arg2 = ATNDeserializer.toInt(data[p + 4]);
            let arg3 = ATNDeserializer.toInt(data[p + 5]);
            let trans = this.edgeFactory(atn, ttype, src, trg, arg1, arg2, arg3, sets);
            // console.log(`EDGE ${trans.constructor.name} ${src}->${trg} ${Transition.serializationNames[ttype]} ${arg1},${arg2},${arg3}`);
            let srcState = atn.states[src];
            srcState.addTransition(trans);
            p += 6;
        }
        let returnTransitionsSet = new Array2DHashSet_1.Array2DHashSet({
            hashCode: (o) => o.stopState ^ o.returnState ^ o.outermostPrecedenceReturn,
            equals: (a, b) => {
                return a.stopState === b.stopState
                    && a.returnState === b.returnState
                    && a.outermostPrecedenceReturn === b.outermostPrecedenceReturn;
            },
        });
        let returnTransitions = [];
        for (let state of atn.states) {
            let returningToLeftFactored = state.ruleIndex >= 0 && atn.ruleToStartState[state.ruleIndex].leftFactored;
            for (let i = 0; i < state.numberOfTransitions; i++) {
                let t = state.transition(i);
                if (!(t instanceof RuleTransition_1.RuleTransition)) {
                    continue;
                }
                let ruleTransition = t;
                let returningFromLeftFactored = atn.ruleToStartState[ruleTransition.target.ruleIndex].leftFactored;
                if (!returningFromLeftFactored && returningToLeftFactored) {
                    continue;
                }
                let outermostPrecedenceReturn = -1;
                if (atn.ruleToStartState[ruleTransition.target.ruleIndex].isPrecedenceRule) {
                    if (ruleTransition.precedence === 0) {
                        outermostPrecedenceReturn = ruleTransition.target.ruleIndex;
                    }
                }
                let current = { stopState: ruleTransition.target.ruleIndex, returnState: ruleTransition.followState.stateNumber, outermostPrecedenceReturn };
                if (returnTransitionsSet.add(current)) {
                    returnTransitions.push(current);
                }
            }
        }
        // Add all elements from returnTransitions to the ATN
        for (let returnTransition of returnTransitions) {
            let transition = new EpsilonTransition_1.EpsilonTransition(atn.states[returnTransition.returnState], returnTransition.outermostPrecedenceReturn);
            atn.ruleToStopState[returnTransition.stopState].addTransition(transition);
        }
        for (let state of atn.states) {
            if (state instanceof BlockStartState_1.BlockStartState) {
                // we need to know the end state to set its start state
                if (state.endState === undefined) {
                    throw new Error("IllegalStateException");
                }
                // block end states can only be associated to a single block start state
                if (state.endState.startState !== undefined) {
                    throw new Error("IllegalStateException");
                }
                state.endState.startState = state;
            }
            if (state instanceof PlusLoopbackState_1.PlusLoopbackState) {
                let loopbackState = state;
                for (let i = 0; i < loopbackState.numberOfTransitions; i++) {
                    let target = loopbackState.transition(i).target;
                    if (target instanceof PlusBlockStartState_1.PlusBlockStartState) {
                        target.loopBackState = loopbackState;
                    }
                }
            }
            else if (state instanceof StarLoopbackState_1.StarLoopbackState) {
                let loopbackState = state;
                for (let i = 0; i < loopbackState.numberOfTransitions; i++) {
                    let target = loopbackState.transition(i).target;
                    if (target instanceof StarLoopEntryState_1.StarLoopEntryState) {
                        target.loopBackState = loopbackState;
                    }
                }
            }
        }
        //
        // DECISIONS
        //
        let ndecisions = ATNDeserializer.toInt(data[p++]);
        for (let i = 1; i <= ndecisions; i++) {
            let s = ATNDeserializer.toInt(data[p++]);
            let decState = atn.states[s];
            atn.decisionToState.push(decState);
            decState.decision = i - 1;
        }
        //
        // LEXER ACTIONS
        //
        if (atn.grammarType === 0 /* LEXER */) {
            if (supportsLexerActions) {
                atn.lexerActions = new Array(ATNDeserializer.toInt(data[p++]));
                for (let i = 0; i < atn.lexerActions.length; i++) {
                    let actionType = ATNDeserializer.toInt(data[p++]);
                    let data1 = ATNDeserializer.toInt(data[p++]);
                    if (data1 === 0xFFFF) {
                        data1 = -1;
                    }
                    let data2 = ATNDeserializer.toInt(data[p++]);
                    if (data2 === 0xFFFF) {
                        data2 = -1;
                    }
                    let lexerAction = this.lexerActionFactory(actionType, data1, data2);
                    atn.lexerActions[i] = lexerAction;
                }
            }
            else {
                // for compatibility with older serialized ATNs, convert the old
                // serialized action index for action transitions to the new
                // form, which is the index of a LexerCustomAction
                let legacyLexerActions = [];
                for (let state of atn.states) {
                    for (let i = 0; i < state.numberOfTransitions; i++) {
                        let transition = state.transition(i);
                        if (!(transition instanceof ActionTransition_1.ActionTransition)) {
                            continue;
                        }
                        let ruleIndex = transition.ruleIndex;
                        let actionIndex = transition.actionIndex;
                        let lexerAction = new LexerCustomAction_1.LexerCustomAction(ruleIndex, actionIndex);
                        state.setTransition(i, new ActionTransition_1.ActionTransition(transition.target, ruleIndex, legacyLexerActions.length, false));
                        legacyLexerActions.push(lexerAction);
                    }
                }
                atn.lexerActions = legacyLexerActions;
            }
        }
        this.markPrecedenceDecisions(atn);
        atn.decisionToDFA = new Array(ndecisions);
        for (let i = 0; i < ndecisions; i++) {
            atn.decisionToDFA[i] = new DFA_1.DFA(atn.decisionToState[i], i);
        }
        if (this.deserializationOptions.isVerifyATN) {
            this.verifyATN(atn);
        }
        if (this.deserializationOptions.isGenerateRuleBypassTransitions && atn.grammarType === 1 /* PARSER */) {
            atn.ruleToTokenType = new Int32Array(atn.ruleToStartState.length);
            for (let i = 0; i < atn.ruleToStartState.length; i++) {
                atn.ruleToTokenType[i] = atn.maxTokenType + i + 1;
            }
            for (let i = 0; i < atn.ruleToStartState.length; i++) {
                let bypassStart = new BasicBlockStartState_1.BasicBlockStartState();
                bypassStart.ruleIndex = i;
                atn.addState(bypassStart);
                let bypassStop = new BlockEndState_1.BlockEndState();
                bypassStop.ruleIndex = i;
                atn.addState(bypassStop);
                bypassStart.endState = bypassStop;
                atn.defineDecisionState(bypassStart);
                bypassStop.startState = bypassStart;
                let endState;
                let excludeTransition;
                if (atn.ruleToStartState[i].isPrecedenceRule) {
                    // wrap from the beginning of the rule to the StarLoopEntryState
                    endState = undefined;
                    for (let state of atn.states) {
                        if (state.ruleIndex !== i) {
                            continue;
                        }
                        if (!(state instanceof StarLoopEntryState_1.StarLoopEntryState)) {
                            continue;
                        }
                        let maybeLoopEndState = state.transition(state.numberOfTransitions - 1).target;
                        if (!(maybeLoopEndState instanceof LoopEndState_1.LoopEndState)) {
                            continue;
                        }
                        if (maybeLoopEndState.epsilonOnlyTransitions && maybeLoopEndState.transition(0).target instanceof RuleStopState_1.RuleStopState) {
                            endState = state;
                            break;
                        }
                    }
                    if (!endState) {
                        throw new Error("Couldn't identify final state of the precedence rule prefix section.");
                    }
                    excludeTransition = endState.loopBackState.transition(0);
                }
                else {
                    endState = atn.ruleToStopState[i];
                }
                // all non-excluded transitions that currently target end state need to target blockEnd instead
                for (let state of atn.states) {
                    for (let i = 0; i < state.numberOfTransitions; i++) {
                        let transition = state.transition(i);
                        if (transition === excludeTransition) {
                            continue;
                        }
                        if (transition.target === endState) {
                            transition.target = bypassStop;
                        }
                    }
                }
                // all transitions leaving the rule start state need to leave blockStart instead
                while (atn.ruleToStartState[i].numberOfTransitions > 0) {
                    let transition = atn.ruleToStartState[i].removeTransition(atn.ruleToStartState[i].numberOfTransitions - 1);
                    bypassStart.addTransition(transition);
                }
                // link the new states
                atn.ruleToStartState[i].addTransition(new EpsilonTransition_1.EpsilonTransition(bypassStart));
                bypassStop.addTransition(new EpsilonTransition_1.EpsilonTransition(endState));
                let matchState = new BasicState_1.BasicState();
                atn.addState(matchState);
                matchState.addTransition(new AtomTransition_1.AtomTransition(bypassStop, atn.ruleToTokenType[i]));
                bypassStart.addTransition(new EpsilonTransition_1.EpsilonTransition(matchState));
            }
            if (this.deserializationOptions.isVerifyATN) {
                // reverify after modification
                this.verifyATN(atn);
            }
        }
        if (this.deserializationOptions.isOptimize) {
            while (true) {
                let optimizationCount = 0;
                optimizationCount += ATNDeserializer.inlineSetRules(atn);
                optimizationCount += ATNDeserializer.combineChainedEpsilons(atn);
                let preserveOrder = atn.grammarType === 0 /* LEXER */;
                optimizationCount += ATNDeserializer.optimizeSets(atn, preserveOrder);
                if (optimizationCount === 0) {
                    break;
                }
            }
            if (this.deserializationOptions.isVerifyATN) {
                // reverify after modification
                this.verifyATN(atn);
            }
        }
        ATNDeserializer.identifyTailCalls(atn);
        return atn;
    }
    deserializeSets(data, p, sets, unicodeDeserializer) {
        let nsets = ATNDeserializer.toInt(data[p++]);
        for (let i = 0; i < nsets; i++) {
            let nintervals = ATNDeserializer.toInt(data[p]);
            p++;
            let set = new IntervalSet_1.IntervalSet();
            sets.push(set);
            let containsEof = ATNDeserializer.toInt(data[p++]) !== 0;
            if (containsEof) {
                set.add(-1);
            }
            for (let j = 0; j < nintervals; j++) {
                let a = unicodeDeserializer.readUnicode(data, p);
                p += unicodeDeserializer.size;
                let b = unicodeDeserializer.readUnicode(data, p);
                p += unicodeDeserializer.size;
                set.add(a, b);
            }
        }
        return p;
    }
    /**
     * Analyze the {@link StarLoopEntryState} states in the specified ATN to set
     * the {@link StarLoopEntryState#precedenceRuleDecision} field to the
     * correct value.
     *
     * @param atn The ATN.
     */
    markPrecedenceDecisions(atn) {
        // Map rule index -> precedence decision for that rule
        let rulePrecedenceDecisions = new Map();
        for (let state of atn.states) {
            if (!(state instanceof StarLoopEntryState_1.StarLoopEntryState)) {
                continue;
            }
            /* We analyze the ATN to determine if this ATN decision state is the
             * decision for the closure block that determines whether a
             * precedence rule should continue or complete.
             */
            if (atn.ruleToStartState[state.ruleIndex].isPrecedenceRule) {
                let maybeLoopEndState = state.transition(state.numberOfTransitions - 1).target;
                if (maybeLoopEndState instanceof LoopEndState_1.LoopEndState) {
                    if (maybeLoopEndState.epsilonOnlyTransitions && maybeLoopEndState.transition(0).target instanceof RuleStopState_1.RuleStopState) {
                        rulePrecedenceDecisions.set(state.ruleIndex, state);
                        state.precedenceRuleDecision = true;
                        state.precedenceLoopbackStates = new BitSet_1.BitSet(atn.states.length);
                    }
                }
            }
        }
        // After marking precedence decisions, we go back through and fill in
        // StarLoopEntryState.precedenceLoopbackStates.
        for (let precedenceDecision of rulePrecedenceDecisions) {
            for (let transition of atn.ruleToStopState[precedenceDecision[0]].getTransitions()) {
                if (transition.serializationType !== 1 /* EPSILON */) {
                    continue;
                }
                let epsilonTransition = transition;
                if (epsilonTransition.outermostPrecedenceReturn !== -1) {
                    continue;
                }
                precedenceDecision[1].precedenceLoopbackStates.set(transition.target.stateNumber);
            }
        }
    }
    verifyATN(atn) {
        // verify assumptions
        for (let state of atn.states) {
            this.checkCondition(state !== undefined, "ATN states should not be undefined.");
            if (state.stateType === ATNStateType_1.ATNStateType.INVALID_TYPE) {
                continue;
            }
            this.checkCondition(state.onlyHasEpsilonTransitions || state.numberOfTransitions <= 1);
            if (state instanceof PlusBlockStartState_1.PlusBlockStartState) {
                this.checkCondition(state.loopBackState !== undefined);
            }
            if (state instanceof StarLoopEntryState_1.StarLoopEntryState) {
                let starLoopEntryState = state;
                this.checkCondition(starLoopEntryState.loopBackState !== undefined);
                this.checkCondition(starLoopEntryState.numberOfTransitions === 2);
                if (starLoopEntryState.transition(0).target instanceof StarBlockStartState_1.StarBlockStartState) {
                    this.checkCondition(starLoopEntryState.transition(1).target instanceof LoopEndState_1.LoopEndState);
                    this.checkCondition(!starLoopEntryState.nonGreedy);
                }
                else if (starLoopEntryState.transition(0).target instanceof LoopEndState_1.LoopEndState) {
                    this.checkCondition(starLoopEntryState.transition(1).target instanceof StarBlockStartState_1.StarBlockStartState);
                    this.checkCondition(starLoopEntryState.nonGreedy);
                }
                else {
                    throw new Error("IllegalStateException");
                }
            }
            if (state instanceof StarLoopbackState_1.StarLoopbackState) {
                this.checkCondition(state.numberOfTransitions === 1);
                this.checkCondition(state.transition(0).target instanceof StarLoopEntryState_1.StarLoopEntryState);
            }
            if (state instanceof LoopEndState_1.LoopEndState) {
                this.checkCondition(state.loopBackState !== undefined);
            }
            if (state instanceof RuleStartState_1.RuleStartState) {
                this.checkCondition(state.stopState !== undefined);
            }
            if (state instanceof BlockStartState_1.BlockStartState) {
                this.checkCondition(state.endState !== undefined);
            }
            if (state instanceof BlockEndState_1.BlockEndState) {
                this.checkCondition(state.startState !== undefined);
            }
            if (state instanceof DecisionState_1.DecisionState) {
                let decisionState = state;
                this.checkCondition(decisionState.numberOfTransitions <= 1 || decisionState.decision >= 0);
            }
            else {
                this.checkCondition(state.numberOfTransitions <= 1 || state instanceof RuleStopState_1.RuleStopState);
            }
        }
    }
    checkCondition(condition, message) {
        if (!condition) {
            throw new Error("IllegalStateException: " + message);
        }
    }
    static inlineSetRules(atn) {
        let inlinedCalls = 0;
        let ruleToInlineTransition = new Array(atn.ruleToStartState.length);
        for (let i = 0; i < atn.ruleToStartState.length; i++) {
            let startState = atn.ruleToStartState[i];
            let middleState = startState;
            while (middleState.onlyHasEpsilonTransitions
                && middleState.numberOfOptimizedTransitions === 1
                && middleState.getOptimizedTransition(0).serializationType === 1 /* EPSILON */) {
                middleState = middleState.getOptimizedTransition(0).target;
            }
            if (middleState.numberOfOptimizedTransitions !== 1) {
                continue;
            }
            let matchTransition = middleState.getOptimizedTransition(0);
            let matchTarget = matchTransition.target;
            if (matchTransition.isEpsilon
                || !matchTarget.onlyHasEpsilonTransitions
                || matchTarget.numberOfOptimizedTransitions !== 1
                || !(matchTarget.getOptimizedTransition(0).target instanceof RuleStopState_1.RuleStopState)) {
                continue;
            }
            switch (matchTransition.serializationType) {
                case 5 /* ATOM */:
                case 2 /* RANGE */:
                case 7 /* SET */:
                    ruleToInlineTransition[i] = matchTransition;
                    break;
                case 8 /* NOT_SET */:
                case 9 /* WILDCARD */:
                    // not implemented yet
                    continue;
                default:
                    continue;
            }
        }
        for (let state of atn.states) {
            if (state.ruleIndex < 0) {
                continue;
            }
            let optimizedTransitions;
            for (let i = 0; i < state.numberOfOptimizedTransitions; i++) {
                let transition = state.getOptimizedTransition(i);
                if (!(transition instanceof RuleTransition_1.RuleTransition)) {
                    if (optimizedTransitions !== undefined) {
                        optimizedTransitions.push(transition);
                    }
                    continue;
                }
                let ruleTransition = transition;
                let effective = ruleToInlineTransition[ruleTransition.target.ruleIndex];
                if (effective === undefined) {
                    if (optimizedTransitions !== undefined) {
                        optimizedTransitions.push(transition);
                    }
                    continue;
                }
                if (optimizedTransitions === undefined) {
                    optimizedTransitions = [];
                    for (let j = 0; j < i; j++) {
                        optimizedTransitions.push(state.getOptimizedTransition(i));
                    }
                }
                inlinedCalls++;
                let target = ruleTransition.followState;
                let intermediateState = new BasicState_1.BasicState();
                intermediateState.setRuleIndex(target.ruleIndex);
                atn.addState(intermediateState);
                optimizedTransitions.push(new EpsilonTransition_1.EpsilonTransition(intermediateState));
                switch (effective.serializationType) {
                    case 5 /* ATOM */:
                        intermediateState.addTransition(new AtomTransition_1.AtomTransition(target, effective._label));
                        break;
                    case 2 /* RANGE */:
                        intermediateState.addTransition(new RangeTransition_1.RangeTransition(target, effective.from, effective.to));
                        break;
                    case 7 /* SET */:
                        intermediateState.addTransition(new SetTransition_1.SetTransition(target, effective.label));
                        break;
                    default:
                        throw new Error("UnsupportedOperationException");
                }
            }
            if (optimizedTransitions !== undefined) {
                if (state.isOptimized) {
                    while (state.numberOfOptimizedTransitions > 0) {
                        state.removeOptimizedTransition(state.numberOfOptimizedTransitions - 1);
                    }
                }
                for (let transition of optimizedTransitions) {
                    state.addOptimizedTransition(transition);
                }
            }
        }
        if (ParserATNSimulator_1.ParserATNSimulator.debug) {
            console.log("ATN runtime optimizer removed " + inlinedCalls + " rule invocations by inlining sets.");
        }
        return inlinedCalls;
    }
    static combineChainedEpsilons(atn) {
        let removedEdges = 0;
        for (let state of atn.states) {
            if (!state.onlyHasEpsilonTransitions || state instanceof RuleStopState_1.RuleStopState) {
                continue;
            }
            let optimizedTransitions;
            nextTransition: for (let i = 0; i < state.numberOfOptimizedTransitions; i++) {
                let transition = state.getOptimizedTransition(i);
                let intermediate = transition.target;
                if (transition.serializationType !== 1 /* EPSILON */
                    || transition.outermostPrecedenceReturn !== -1
                    || intermediate.stateType !== ATNStateType_1.ATNStateType.BASIC
                    || !intermediate.onlyHasEpsilonTransitions) {
                    if (optimizedTransitions !== undefined) {
                        optimizedTransitions.push(transition);
                    }
                    continue nextTransition;
                }
                for (let j = 0; j < intermediate.numberOfOptimizedTransitions; j++) {
                    if (intermediate.getOptimizedTransition(j).serializationType !== 1 /* EPSILON */
                        || intermediate.getOptimizedTransition(j).outermostPrecedenceReturn !== -1) {
                        if (optimizedTransitions !== undefined) {
                            optimizedTransitions.push(transition);
                        }
                        continue nextTransition;
                    }
                }
                removedEdges++;
                if (optimizedTransitions === undefined) {
                    optimizedTransitions = [];
                    for (let j = 0; j < i; j++) {
                        optimizedTransitions.push(state.getOptimizedTransition(j));
                    }
                }
                for (let j = 0; j < intermediate.numberOfOptimizedTransitions; j++) {
                    let target = intermediate.getOptimizedTransition(j).target;
                    optimizedTransitions.push(new EpsilonTransition_1.EpsilonTransition(target));
                }
            }
            if (optimizedTransitions !== undefined) {
                if (state.isOptimized) {
                    while (state.numberOfOptimizedTransitions > 0) {
                        state.removeOptimizedTransition(state.numberOfOptimizedTransitions - 1);
                    }
                }
                for (let transition of optimizedTransitions) {
                    state.addOptimizedTransition(transition);
                }
            }
        }
        if (ParserATNSimulator_1.ParserATNSimulator.debug) {
            console.log("ATN runtime optimizer removed " + removedEdges + " transitions by combining chained epsilon transitions.");
        }
        return removedEdges;
    }
    static optimizeSets(atn, preserveOrder) {
        if (preserveOrder) {
            // this optimization currently doesn't preserve edge order.
            return 0;
        }
        let removedPaths = 0;
        let decisions = atn.decisionToState;
        for (let decision of decisions) {
            let setTransitions = new IntervalSet_1.IntervalSet();
            for (let i = 0; i < decision.numberOfOptimizedTransitions; i++) {
                let epsTransition = decision.getOptimizedTransition(i);
                if (!(epsTransition instanceof EpsilonTransition_1.EpsilonTransition)) {
                    continue;
                }
                if (epsTransition.target.numberOfOptimizedTransitions !== 1) {
                    continue;
                }
                let transition = epsTransition.target.getOptimizedTransition(0);
                if (!(transition.target instanceof BlockEndState_1.BlockEndState)) {
                    continue;
                }
                if (transition instanceof NotSetTransition_1.NotSetTransition) {
                    // TODO: not yet implemented
                    continue;
                }
                if (transition instanceof AtomTransition_1.AtomTransition
                    || transition instanceof RangeTransition_1.RangeTransition
                    || transition instanceof SetTransition_1.SetTransition) {
                    setTransitions.add(i);
                }
            }
            if (setTransitions.size <= 1) {
                continue;
            }
            let optimizedTransitions = [];
            for (let i = 0; i < decision.numberOfOptimizedTransitions; i++) {
                if (!setTransitions.contains(i)) {
                    optimizedTransitions.push(decision.getOptimizedTransition(i));
                }
            }
            let blockEndState = decision.getOptimizedTransition(setTransitions.minElement).target.getOptimizedTransition(0).target;
            let matchSet = new IntervalSet_1.IntervalSet();
            for (let interval of setTransitions.intervals) {
                for (let j = interval.a; j <= interval.b; j++) {
                    let matchTransition = decision.getOptimizedTransition(j).target.getOptimizedTransition(0);
                    if (matchTransition instanceof NotSetTransition_1.NotSetTransition) {
                        throw new Error("Not yet implemented.");
                    }
                    else {
                        matchSet.addAll(matchTransition.label);
                    }
                }
            }
            let newTransition;
            if (matchSet.intervals.length === 1) {
                if (matchSet.size === 1) {
                    newTransition = new AtomTransition_1.AtomTransition(blockEndState, matchSet.minElement);
                }
                else {
                    let matchInterval = matchSet.intervals[0];
                    newTransition = new RangeTransition_1.RangeTransition(blockEndState, matchInterval.a, matchInterval.b);
                }
            }
            else {
                newTransition = new SetTransition_1.SetTransition(blockEndState, matchSet);
            }
            let setOptimizedState = new BasicState_1.BasicState();
            setOptimizedState.setRuleIndex(decision.ruleIndex);
            atn.addState(setOptimizedState);
            setOptimizedState.addTransition(newTransition);
            optimizedTransitions.push(new EpsilonTransition_1.EpsilonTransition(setOptimizedState));
            removedPaths += decision.numberOfOptimizedTransitions - optimizedTransitions.length;
            if (decision.isOptimized) {
                while (decision.numberOfOptimizedTransitions > 0) {
                    decision.removeOptimizedTransition(decision.numberOfOptimizedTransitions - 1);
                }
            }
            for (let transition of optimizedTransitions) {
                decision.addOptimizedTransition(transition);
            }
        }
        if (ParserATNSimulator_1.ParserATNSimulator.debug) {
            console.log("ATN runtime optimizer removed " + removedPaths + " paths by collapsing sets.");
        }
        return removedPaths;
    }
    static identifyTailCalls(atn) {
        for (let state of atn.states) {
            for (let i = 0; i < state.numberOfTransitions; i++) {
                let transition = state.transition(i);
                if (!(transition instanceof RuleTransition_1.RuleTransition)) {
                    continue;
                }
                transition.tailCall = this.testTailCall(atn, transition, false);
                transition.optimizedTailCall = this.testTailCall(atn, transition, true);
            }
            if (!state.isOptimized) {
                continue;
            }
            for (let i = 0; i < state.numberOfOptimizedTransitions; i++) {
                let transition = state.getOptimizedTransition(i);
                if (!(transition instanceof RuleTransition_1.RuleTransition)) {
                    continue;
                }
                transition.tailCall = this.testTailCall(atn, transition, false);
                transition.optimizedTailCall = this.testTailCall(atn, transition, true);
            }
        }
    }
    static testTailCall(atn, transition, optimizedPath) {
        if (!optimizedPath && transition.tailCall) {
            return true;
        }
        if (optimizedPath && transition.optimizedTailCall) {
            return true;
        }
        let reachable = new BitSet_1.BitSet(atn.states.length);
        let worklist = [];
        worklist.push(transition.followState);
        while (true) {
            let state = worklist.pop();
            if (!state) {
                break;
            }
            if (reachable.get(state.stateNumber)) {
                continue;
            }
            if (state instanceof RuleStopState_1.RuleStopState) {
                continue;
            }
            if (!state.onlyHasEpsilonTransitions) {
                return false;
            }
            let transitionCount = optimizedPath ? state.numberOfOptimizedTransitions : state.numberOfTransitions;
            for (let i = 0; i < transitionCount; i++) {
                let t = optimizedPath ? state.getOptimizedTransition(i) : state.transition(i);
                if (t.serializationType !== 1 /* EPSILON */) {
                    return false;
                }
                worklist.push(t.target);
            }
        }
        return true;
    }
    static toInt(c) {
        return c;
    }
    static toInt32(data, offset) {
        return (data[offset] | (data[offset + 1] << 16)) >>> 0;
    }
    static toUUID(data, offset) {
        let leastSigBits = ATNDeserializer.toInt32(data, offset);
        let lessSigBits = ATNDeserializer.toInt32(data, offset + 2);
        let moreSigBits = ATNDeserializer.toInt32(data, offset + 4);
        let mostSigBits = ATNDeserializer.toInt32(data, offset + 6);
        return new UUID_1.UUID(mostSigBits, moreSigBits, lessSigBits, leastSigBits);
    }
    edgeFactory(atn, type, src, trg, arg1, arg2, arg3, sets) {
        let target = atn.states[trg];
        switch (type) {
            case 1 /* EPSILON */: return new EpsilonTransition_1.EpsilonTransition(target);
            case 2 /* RANGE */:
                if (arg3 !== 0) {
                    return new RangeTransition_1.RangeTransition(target, Token_1.Token.EOF, arg2);
                }
                else {
                    return new RangeTransition_1.RangeTransition(target, arg1, arg2);
                }
            case 3 /* RULE */:
                let rt = new RuleTransition_1.RuleTransition(atn.states[arg1], arg2, arg3, target);
                return rt;
            case 4 /* PREDICATE */:
                let pt = new PredicateTransition_1.PredicateTransition(target, arg1, arg2, arg3 !== 0);
                return pt;
            case 10 /* PRECEDENCE */:
                return new PrecedencePredicateTransition_1.PrecedencePredicateTransition(target, arg1);
            case 5 /* ATOM */:
                if (arg3 !== 0) {
                    return new AtomTransition_1.AtomTransition(target, Token_1.Token.EOF);
                }
                else {
                    return new AtomTransition_1.AtomTransition(target, arg1);
                }
            case 6 /* ACTION */:
                let a = new ActionTransition_1.ActionTransition(target, arg1, arg2, arg3 !== 0);
                return a;
            case 7 /* SET */: return new SetTransition_1.SetTransition(target, sets[arg1]);
            case 8 /* NOT_SET */: return new NotSetTransition_1.NotSetTransition(target, sets[arg1]);
            case 9 /* WILDCARD */: return new WildcardTransition_1.WildcardTransition(target);
        }
        throw new Error("The specified transition type is not valid.");
    }
    stateFactory(type, ruleIndex) {
        let s;
        switch (type) {
            case ATNStateType_1.ATNStateType.INVALID_TYPE: return new InvalidState_1.InvalidState();
            case ATNStateType_1.ATNStateType.BASIC:
                s = new BasicState_1.BasicState();
                break;
            case ATNStateType_1.ATNStateType.RULE_START:
                s = new RuleStartState_1.RuleStartState();
                break;
            case ATNStateType_1.ATNStateType.BLOCK_START:
                s = new BasicBlockStartState_1.BasicBlockStartState();
                break;
            case ATNStateType_1.ATNStateType.PLUS_BLOCK_START:
                s = new PlusBlockStartState_1.PlusBlockStartState();
                break;
            case ATNStateType_1.ATNStateType.STAR_BLOCK_START:
                s = new StarBlockStartState_1.StarBlockStartState();
                break;
            case ATNStateType_1.ATNStateType.TOKEN_START:
                s = new TokensStartState_1.TokensStartState();
                break;
            case ATNStateType_1.ATNStateType.RULE_STOP:
                s = new RuleStopState_1.RuleStopState();
                break;
            case ATNStateType_1.ATNStateType.BLOCK_END:
                s = new BlockEndState_1.BlockEndState();
                break;
            case ATNStateType_1.ATNStateType.STAR_LOOP_BACK:
                s = new StarLoopbackState_1.StarLoopbackState();
                break;
            case ATNStateType_1.ATNStateType.STAR_LOOP_ENTRY:
                s = new StarLoopEntryState_1.StarLoopEntryState();
                break;
            case ATNStateType_1.ATNStateType.PLUS_LOOP_BACK:
                s = new PlusLoopbackState_1.PlusLoopbackState();
                break;
            case ATNStateType_1.ATNStateType.LOOP_END:
                s = new LoopEndState_1.LoopEndState();
                break;
            default:
                let message = `The specified state type ${type} is not valid.`;
                throw new Error(message);
        }
        s.ruleIndex = ruleIndex;
        return s;
    }
    lexerActionFactory(type, data1, data2) {
        switch (type) {
            case 0 /* CHANNEL */:
                return new LexerChannelAction_1.LexerChannelAction(data1);
            case 1 /* CUSTOM */:
                return new LexerCustomAction_1.LexerCustomAction(data1, data2);
            case 2 /* MODE */:
                return new LexerModeAction_1.LexerModeAction(data1);
            case 3 /* MORE */:
                return LexerMoreAction_1.LexerMoreAction.INSTANCE;
            case 4 /* POP_MODE */:
                return LexerPopModeAction_1.LexerPopModeAction.INSTANCE;
            case 5 /* PUSH_MODE */:
                return new LexerPushModeAction_1.LexerPushModeAction(data1);
            case 6 /* SKIP */:
                return LexerSkipAction_1.LexerSkipAction.INSTANCE;
            case 7 /* TYPE */:
                return new LexerTypeAction_1.LexerTypeAction(data1);
            default:
                let message = `The specified lexer action type ${type} is not valid.`;
                throw new Error(message);
        }
    }
}
/* WARNING: DO NOT MERGE THESE LINES. If UUIDs differ during a merge,
 * resolve the conflict by generating a new ID!
 */
/**
 * This is the earliest supported serialized UUID.
 */
ATNDeserializer.BASE_SERIALIZED_UUID = UUID_1.UUID.fromString("E4178468-DF95-44D0-AD87-F22A5D5FB6D3");
/**
 * This UUID indicates an extension of {@link #ADDED_PRECEDENCE_TRANSITIONS}
 * for the addition of lexer actions encoded as a sequence of
 * {@link LexerAction} instances.
 */
ATNDeserializer.ADDED_LEXER_ACTIONS = UUID_1.UUID.fromString("AB35191A-1603-487E-B75A-479B831EAF6D");
/**
 * This UUID indicates the serialized ATN contains two sets of
 * IntervalSets, where the second set's values are encoded as
 * 32-bit integers to support the full Unicode SMP range up to U+10FFFF.
 */
ATNDeserializer.ADDED_UNICODE_SMP = UUID_1.UUID.fromString("C23FEA89-0605-4f51-AFB8-058BCAB8C91B");
/**
 * This list contains all of the currently supported UUIDs, ordered by when
 * the feature first appeared in this branch.
 */
ATNDeserializer.SUPPORTED_UUIDS = [
    ATNDeserializer.BASE_SERIALIZED_UUID,
    ATNDeserializer.ADDED_LEXER_ACTIONS,
    ATNDeserializer.ADDED_UNICODE_SMP,
];
/**
 * This is the current serialized UUID.
 */
ATNDeserializer.SERIALIZED_UUID = ATNDeserializer.ADDED_UNICODE_SMP;
__decorate([
    Decorators_1.NotNull
], ATNDeserializer.prototype, "deserializationOptions", void 0);
__decorate([
    __param(0, Decorators_1.NotNull)
], ATNDeserializer.prototype, "deserialize", null);
__decorate([
    __param(0, Decorators_1.NotNull)
], ATNDeserializer.prototype, "markPrecedenceDecisions", null);
__decorate([
    Decorators_1.NotNull,
    __param(0, Decorators_1.NotNull)
], ATNDeserializer.prototype, "edgeFactory", null);
exports.ATNDeserializer = ATNDeserializer;
//# sourceMappingURL=ATNDeserializer.js.map