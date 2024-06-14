/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ATNState } from "./ATNState";
import { Equatable } from "../misc/Stubs";
import { LexerActionExecutor } from "./LexerActionExecutor";
import { PredictionContext } from "./PredictionContext";
import { PredictionContextCache } from "./PredictionContextCache";
import { Recognizer } from "../Recognizer";
import { SemanticContext } from "./SemanticContext";
/**
 * Represents a location with context in an ATN. The location is identified by the following values:
 *
 * * The current ATN state
 * * The predicted alternative
 * * The semantic context which must be true for this configuration to be enabled
 * * The syntactic context, which is represented as a graph-structured stack whose path(s) lead to the root of the rule
 *   invocations leading to this state
 *
 * In addition to these values, `ATNConfig` stores several properties about paths taken to get to the location which
 * were added over time to help with performance, correctness, and/or debugging.
 *
 * * `reachesIntoOuterContext`:: Used to ensure semantic predicates are not evaluated in the wrong context.
 * * `hasPassedThroughNonGreedyDecision`: Used for enabling first-match-wins instead of longest-match-wins after
 *   crossing a non-greedy decision.
 * * `lexerActionExecutor`: Used for tracking the lexer action(s) to execute should this instance be selected during
 *   lexing.
 * * `isPrecedenceFilterSuppressed`: A state variable for one of the dynamic disambiguation strategies employed by
 *   `ParserATNSimulator.applyPrecedenceFilter`.
 *
 * Due to the use of a graph-structured stack, a single `ATNConfig` is capable of representing many individual ATN
 * configurations which reached the same location in an ATN by following different paths.
 *
 * PERF: To conserve memory, `ATNConfig` is split into several different concrete types. `ATNConfig` itself stores the
 * minimum amount of information typically used to define an `ATNConfig` instance. Various derived types provide
 * additional storage space for cases where a non-default value is used for some of the object properties. The
 * `ATNConfig.create` and `ATNConfig.transform` methods automatically select the smallest concrete type capable of
 * representing the unique information for any given `ATNConfig`.
 */
export declare class ATNConfig implements Equatable {
    /** The ATN state associated with this configuration */
    private _state;
    /**
     * This is a bit-field currently containing the following values.
     *
     * * 0x00FFFFFF: Alternative
     * * 0x7F000000: Outer context depth
     * * 0x80000000: Suppress precedence filter
     */
    private altAndOuterContextDepth;
    /** The stack of invoking states leading to the rule/states associated
     *  with this config.  We track only those contexts pushed during
     *  execution of the ATN simulator.
     */
    private _context;
    constructor(/*@NotNull*/ state: ATNState, alt: number, /*@NotNull*/ context: PredictionContext);
    constructor(/*@NotNull*/ state: ATNState, /*@NotNull*/ c: ATNConfig, /*@NotNull*/ context: PredictionContext);
    static create(/*@NotNull*/ state: ATNState, alt: number, context: PredictionContext): ATNConfig;
    static create(/*@NotNull*/ state: ATNState, alt: number, context: PredictionContext, /*@NotNull*/ semanticContext: SemanticContext): ATNConfig;
    static create(/*@NotNull*/ state: ATNState, alt: number, context: PredictionContext, /*@*/ semanticContext: SemanticContext, lexerActionExecutor: LexerActionExecutor | undefined): ATNConfig;
    /** Gets the ATN state associated with this configuration */
    get state(): ATNState;
    /** What alt (or lexer rule) is predicted by this configuration */
    get alt(): number;
    get context(): PredictionContext;
    set context(context: PredictionContext);
    get reachesIntoOuterContext(): boolean;
    /**
     * We cannot execute predicates dependent upon local context unless
     * we know for sure we are in the correct context. Because there is
     * no way to do this efficiently, we simply cannot evaluate
     * dependent predicates unless we are in the rule that initially
     * invokes the ATN simulator.
     *
     * closure() tracks the depth of how far we dip into the outer context:
     * depth &gt; 0.  Note that it may not be totally accurate depth since I
     * don't ever decrement. TODO: make it a boolean then
     */
    get outerContextDepth(): number;
    set outerContextDepth(outerContextDepth: number);
    get lexerActionExecutor(): LexerActionExecutor | undefined;
    get semanticContext(): SemanticContext;
    get hasPassedThroughNonGreedyDecision(): boolean;
    clone(): ATNConfig;
    transform(/*@NotNull*/ state: ATNState, checkNonGreedy: boolean): ATNConfig;
    transform(/*@NotNull*/ state: ATNState, checkNonGreedy: boolean, /*@NotNull*/ semanticContext: SemanticContext): ATNConfig;
    transform(/*@NotNull*/ state: ATNState, checkNonGreedy: boolean, context: PredictionContext): ATNConfig;
    transform(/*@NotNull*/ state: ATNState, checkNonGreedy: boolean, lexerActionExecutor: LexerActionExecutor): ATNConfig;
    private transformImpl;
    private static checkNonGreedyDecision;
    appendContext(context: number, contextCache: PredictionContextCache): ATNConfig;
    appendContext(context: PredictionContext, contextCache: PredictionContextCache): ATNConfig;
    /**
     * Determines if this `ATNConfig` fully contains another `ATNConfig`.
     *
     * An ATN configuration represents a position (including context) in an ATN during parsing. Since `ATNConfig` stores
     * the context as a graph, a single `ATNConfig` instance is capable of representing many ATN configurations which
     * are all in the same "location" but have different contexts. These `ATNConfig` instances are again merged when
     * they are added to an `ATNConfigSet`. This method supports `ATNConfigSet.contains` by evaluating whether a
     * particular `ATNConfig` contains all of the ATN configurations represented by another `ATNConfig`.
     *
     * An `ATNConfig` _a_ contains another `ATNConfig` _b_ if all of the following conditions are met:
     *
     * * The configurations are in the same state (`state`)
     * * The configurations predict the same alternative (`alt`)
     * * The semantic context of _a_ implies the semantic context of _b_ (this method performs a weaker equality check)
     * * Joining the prediction contexts of _a_ and _b_ results in the prediction context of _a_
     *
     * This method implements a conservative approximation of containment. As a result, when this method returns `true`
     * it is known that parsing from `subconfig` can only recognize a subset of the inputs which can be recognized
     * starting at the current `ATNConfig`. However, due to the imprecise evaluation of implication for the semantic
     * contexts, no assumptions can be made about the relationship between the configurations when this method returns
     * `false`.
     *
     * @param subconfig The sub configuration.
     * @returns `true` if this configuration contains `subconfig`; otherwise, `false`.
     */
    contains(subconfig: ATNConfig): boolean;
    get isPrecedenceFilterSuppressed(): boolean;
    set isPrecedenceFilterSuppressed(value: boolean);
    /** An ATN configuration is equal to another if both have
     *  the same state, they predict the same alternative, and
     *  syntactic/semantic contexts are the same.
     */
    equals(o: any): boolean;
    hashCode(): number;
    /**
     * Returns a graphical representation of the current `ATNConfig` in Graphviz format. The graph can be stored to a
     * **.dot** file and then rendered to an image using Graphviz.
     *
     * @returns A Graphviz graph representing the current `ATNConfig`.
     *
     * @see http://www.graphviz.org/
     */
    toDotString(): string;
    toString(): string;
    toString(recog: Recognizer<any, any> | undefined, showAlt: boolean): string;
    toString(recog: Recognizer<any, any> | undefined, showAlt: boolean, showContext: boolean): string;
}
