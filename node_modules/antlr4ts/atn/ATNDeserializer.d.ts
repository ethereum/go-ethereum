/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ATN } from "./ATN";
import { ATNDeserializationOptions } from "./ATNDeserializationOptions";
import { ATNState } from "./ATNState";
import { ATNStateType } from "./ATNStateType";
import { IntervalSet } from "../misc/IntervalSet";
import { LexerAction } from "./LexerAction";
import { LexerActionType } from "./LexerActionType";
import { Transition } from "./Transition";
import { TransitionType } from "./TransitionType";
import { UUID } from "../misc/UUID";
/**
 *
 * @author Sam Harwell
 */
export declare class ATNDeserializer {
    static get SERIALIZED_VERSION(): number;
    /**
     * This is the earliest supported serialized UUID.
     */
    private static readonly BASE_SERIALIZED_UUID;
    /**
     * This UUID indicates an extension of {@link #ADDED_PRECEDENCE_TRANSITIONS}
     * for the addition of lexer actions encoded as a sequence of
     * {@link LexerAction} instances.
     */
    private static readonly ADDED_LEXER_ACTIONS;
    /**
     * This UUID indicates the serialized ATN contains two sets of
     * IntervalSets, where the second set's values are encoded as
     * 32-bit integers to support the full Unicode SMP range up to U+10FFFF.
     */
    private static readonly ADDED_UNICODE_SMP;
    /**
     * This list contains all of the currently supported UUIDs, ordered by when
     * the feature first appeared in this branch.
     */
    private static readonly SUPPORTED_UUIDS;
    /**
     * This is the current serialized UUID.
     */
    private static readonly SERIALIZED_UUID;
    private readonly deserializationOptions;
    constructor(deserializationOptions?: ATNDeserializationOptions);
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
    protected static isFeatureSupported(feature: UUID, actualUuid: UUID): boolean;
    private static getUnicodeDeserializer;
    deserialize(data: Uint16Array): ATN;
    private deserializeSets;
    /**
     * Analyze the {@link StarLoopEntryState} states in the specified ATN to set
     * the {@link StarLoopEntryState#precedenceRuleDecision} field to the
     * correct value.
     *
     * @param atn The ATN.
     */
    protected markPrecedenceDecisions(atn: ATN): void;
    protected verifyATN(atn: ATN): void;
    protected checkCondition(condition: boolean, message?: string): void;
    private static inlineSetRules;
    private static combineChainedEpsilons;
    private static optimizeSets;
    private static identifyTailCalls;
    private static testTailCall;
    protected static toInt(c: number): number;
    protected static toInt32(data: Uint16Array, offset: number): number;
    protected static toUUID(data: Uint16Array, offset: number): UUID;
    protected edgeFactory(atn: ATN, type: TransitionType, src: number, trg: number, arg1: number, arg2: number, arg3: number, sets: IntervalSet[]): Transition;
    protected stateFactory(type: ATNStateType, ruleIndex: number): ATNState;
    protected lexerActionFactory(type: LexerActionType, data1: number, data2: number): LexerAction;
}
