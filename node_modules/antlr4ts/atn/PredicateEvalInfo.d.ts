/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { DecisionEventInfo } from "./DecisionEventInfo";
import { SemanticContext } from "./SemanticContext";
import { SimulatorState } from "./SimulatorState";
import { TokenStream } from "../TokenStream";
/**
 * This class represents profiling event information for semantic predicate
 * evaluations which occur during prediction.
 *
 * @see ParserATNSimulator#evalSemanticContext
 *
 * @since 4.3
 */
export declare class PredicateEvalInfo extends DecisionEventInfo {
    /**
     * The semantic context which was evaluated.
     */
    semctx: SemanticContext;
    /**
     * The alternative number for the decision which is guarded by the semantic
     * context {@link #semctx}. Note that other ATN
     * configurations may predict the same alternative which are guarded by
     * other semantic contexts and/or {@link SemanticContext#NONE}.
     */
    predictedAlt: number;
    /**
     * The result of evaluating the semantic context {@link #semctx}.
     */
    evalResult: boolean;
    /**
     * Constructs a new instance of the {@link PredicateEvalInfo} class with the
     * specified detailed predicate evaluation information.
     *
     * @param state The simulator state
     * @param decision The decision number
     * @param input The input token stream
     * @param startIndex The start index for the current prediction
     * @param stopIndex The index at which the predicate evaluation was
     * triggered. Note that the input stream may be reset to other positions for
     * the actual evaluation of individual predicates.
     * @param semctx The semantic context which was evaluated
     * @param evalResult The results of evaluating the semantic context
     * @param predictedAlt The alternative number for the decision which is
     * guarded by the semantic context `semctx`. See {@link #predictedAlt}
     * for more information.
     *
     * @see ParserATNSimulator#evalSemanticContext(SemanticContext, ParserRuleContext, int)
     * @see SemanticContext#eval(Recognizer, RuleContext)
     */
    constructor(state: SimulatorState, decision: number, input: TokenStream, startIndex: number, stopIndex: number, semctx: SemanticContext, evalResult: boolean, predictedAlt: number);
}
