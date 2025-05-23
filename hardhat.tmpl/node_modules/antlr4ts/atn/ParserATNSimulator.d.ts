/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ActionTransition } from "./ActionTransition";
import { Array2DHashSet } from "../misc/Array2DHashSet";
import { ATN } from "./ATN";
import { ATNConfig } from "./ATNConfig";
import { ATNConfigSet } from "./ATNConfigSet";
import { ATNSimulator } from "./ATNSimulator";
import { ATNState } from "./ATNState";
import { BitSet } from "../misc/BitSet";
import { DFA } from "../dfa/DFA";
import { DFAState } from "../dfa/DFAState";
import { IntegerList } from "../misc/IntegerList";
import { NoViableAltException } from "../NoViableAltException";
import { Parser } from "../Parser";
import { ParserRuleContext } from "../ParserRuleContext";
import { PrecedencePredicateTransition } from "./PrecedencePredicateTransition";
import { PredicateTransition } from "./PredicateTransition";
import { PredictionContextCache } from "./PredictionContextCache";
import { PredictionMode } from "./PredictionMode";
import { RuleContext } from "../RuleContext";
import { RuleTransition } from "./RuleTransition";
import { SemanticContext } from "./SemanticContext";
import { SimulatorState } from "./SimulatorState";
import { TokenStream } from "../TokenStream";
import { Transition } from "./Transition";
/**
 * The embodiment of the adaptive LL(*), ALL(*), parsing strategy.
 *
 * The basic complexity of the adaptive strategy makes it harder to understand.
 * We begin with ATN simulation to build paths in a DFA. Subsequent prediction
 * requests go through the DFA first. If they reach a state without an edge for
 * the current symbol, the algorithm fails over to the ATN simulation to
 * complete the DFA path for the current input (until it finds a conflict state
 * or uniquely predicting state).
 *
 * All of that is done without using the outer context because we want to create
 * a DFA that is not dependent upon the rule invocation stack when we do a
 * prediction. One DFA works in all contexts. We avoid using context not
 * necessarily because it's slower, although it can be, but because of the DFA
 * caching problem. The closure routine only considers the rule invocation stack
 * created during prediction beginning in the decision rule. For example, if
 * prediction occurs without invoking another rule's ATN, there are no context
 * stacks in the configurations. When lack of context leads to a conflict, we
 * don't know if it's an ambiguity or a weakness in the strong LL(*) parsing
 * strategy (versus full LL(*)).
 *
 * When SLL yields a configuration set with conflict, we rewind the input and
 * retry the ATN simulation, this time using full outer context without adding
 * to the DFA. Configuration context stacks will be the full invocation stacks
 * from the start rule. If we get a conflict using full context, then we can
 * definitively say we have a true ambiguity for that input sequence. If we
 * don't get a conflict, it implies that the decision is sensitive to the outer
 * context. (It is not context-sensitive in the sense of context-sensitive
 * grammars.)
 *
 * The next time we reach this DFA state with an SLL conflict, through DFA
 * simulation, we will again retry the ATN simulation using full context mode.
 * This is slow because we can't save the results and have to "interpret" the
 * ATN each time we get that input.
 *
 * **CACHING FULL CONTEXT PREDICTIONS**
 *
 * We could cache results from full context to predicted alternative easily and
 * that saves a lot of time but doesn't work in presence of predicates. The set
 * of visible predicates from the ATN start state changes depending on the
 * context, because closure can fall off the end of a rule. I tried to cache
 * tuples (stack context, semantic context, predicted alt) but it was slower
 * than interpreting and much more complicated. Also required a huge amount of
 * memory. The goal is not to create the world's fastest parser anyway. I'd like
 * to keep this algorithm simple. By launching multiple threads, we can improve
 * the speed of parsing across a large number of files.
 *
 * There is no strict ordering between the amount of input used by SLL vs LL,
 * which makes it really hard to build a cache for full context. Let's say that
 * we have input A B C that leads to an SLL conflict with full context X. That
 * implies that using X we might only use A B but we could also use A B C D to
 * resolve conflict. Input A B C D could predict alternative 1 in one position
 * in the input and A B C E could predict alternative 2 in another position in
 * input. The conflicting SLL configurations could still be non-unique in the
 * full context prediction, which would lead us to requiring more input than the
 * original A B C.	To make a	prediction cache work, we have to track	the exact
 * input	used during the previous prediction. That amounts to a cache that maps
 * X to a specific DFA for that context.
 *
 * Something should be done for left-recursive expression predictions. They are
 * likely LL(1) + pred eval. Easier to do the whole SLL unless error and retry
 * with full LL thing Sam does.
 *
 * **AVOIDING FULL CONTEXT PREDICTION**
 *
 * We avoid doing full context retry when the outer context is empty, we did not
 * dip into the outer context by falling off the end of the decision state rule,
 * or when we force SLL mode.
 *
 * As an example of the not dip into outer context case, consider as super
 * constructor calls versus function calls. One grammar might look like
 * this:
 *
 * ```antlr
 * ctorBody
 *   : '{' superCall? stat* '}'
 *   ;
 * ```
 *
 * Or, you might see something like
 *
 * ```antlr
 * stat
 *   : superCall ';'
 *   | expression ';'
 *   | ...
 *   ;
 * ```
 *
 * In both cases I believe that no closure operations will dip into the outer
 * context. In the first case ctorBody in the worst case will stop at the '}'.
 * In the 2nd case it should stop at the ';'. Both cases should stay within the
 * entry rule and not dip into the outer context.
 *
 * **PREDICATES**
 *
 * Predicates are always evaluated if present in either SLL or LL both. SLL and
 * LL simulation deals with predicates differently. SLL collects predicates as
 * it performs closure operations like ANTLR v3 did. It delays predicate
 * evaluation until it reaches and accept state. This allows us to cache the SLL
 * ATN simulation whereas, if we had evaluated predicates on-the-fly during
 * closure, the DFA state configuration sets would be different and we couldn't
 * build up a suitable DFA.
 *
 * When building a DFA accept state during ATN simulation, we evaluate any
 * predicates and return the sole semantically valid alternative. If there is
 * more than 1 alternative, we report an ambiguity. If there are 0 alternatives,
 * we throw an exception. Alternatives without predicates act like they have
 * true predicates. The simple way to think about it is to strip away all
 * alternatives with false predicates and choose the minimum alternative that
 * remains.
 *
 * When we start in the DFA and reach an accept state that's predicated, we test
 * those and return the minimum semantically viable alternative. If no
 * alternatives are viable, we throw an exception.
 *
 * During full LL ATN simulation, closure always evaluates predicates and
 * on-the-fly. This is crucial to reducing the configuration set size during
 * closure. It hits a landmine when parsing with the Java grammar, for example,
 * without this on-the-fly evaluation.
 *
 * **SHARING DFA**
 *
 * All instances of the same parser share the same decision DFAs through a
 * static field. Each instance gets its own ATN simulator but they share the
 * same {@link ATN#decisionToDFA} field. They also share a
 * {@link PredictionContextCache} object that makes sure that all
 * {@link PredictionContext} objects are shared among the DFA states. This makes
 * a big size difference.
 *
 * **THREAD SAFETY**
 *
 * The {@link ParserATNSimulator} locks on the {@link ATN#decisionToDFA} field when
 * it adds a new DFA object to that array. {@link #addDFAEdge}
 * locks on the DFA for the current decision when setting the
 * {@link DFAState#edges} field. {@link #addDFAState} locks on
 * the DFA for the current decision when looking up a DFA state to see if it
 * already exists. We must make sure that all requests to add DFA states that
 * are equivalent result in the same shared DFA object. This is because lots of
 * threads will be trying to update the DFA at once. The
 * {@link #addDFAState} method also locks inside the DFA lock
 * but this time on the shared context cache when it rebuilds the
 * configurations' {@link PredictionContext} objects using cached
 * subgraphs/nodes. No other locking occurs, even during DFA simulation. This is
 * safe as long as we can guarantee that all threads referencing
 * `s.edge[t]` get the same physical target {@link DFAState}, or
 * `undefined`. Once into the DFA, the DFA simulation does not reference the
 * {@link DFA#states} map. It follows the {@link DFAState#edges} field to new
 * targets. The DFA simulator will either find {@link DFAState#edges} to be
 * `undefined`, to be non-`undefined` and `dfa.edges[t]` undefined, or
 * `dfa.edges[t]` to be non-undefined. The
 * {@link #addDFAEdge} method could be racing to set the field
 * but in either case the DFA simulator works; if `undefined`, and requests ATN
 * simulation. It could also race trying to get `dfa.edges[t]`, but either
 * way it will work because it's not doing a test and set operation.
 *
 * **Starting with SLL then failing to combined SLL/LL (Two-Stage
 * Parsing)**
 *
 * Sam pointed out that if SLL does not give a syntax error, then there is no
 * point in doing full LL, which is slower. We only have to try LL if we get a
 * syntax error. For maximum speed, Sam starts the parser set to pure SLL
 * mode with the {@link BailErrorStrategy}:
 *
 * ```
 * parser.interpreter.{@link #setPredictionMode setPredictionMode}`(`{@link PredictionMode#SLL}`)`;
 * parser.{@link Parser#setErrorHandler setErrorHandler}(new {@link BailErrorStrategy}());
 * ```
 *
 * If it does not get a syntax error, then we're done. If it does get a syntax
 * error, we need to retry with the combined SLL/LL strategy.
 *
 * The reason this works is as follows. If there are no SLL conflicts, then the
 * grammar is SLL (at least for that input set). If there is an SLL conflict,
 * the full LL analysis must yield a set of viable alternatives which is a
 * subset of the alternatives reported by SLL. If the LL set is a singleton,
 * then the grammar is LL but not SLL. If the LL set is the same size as the SLL
 * set, the decision is SLL. If the LL set has size &gt; 1, then that decision
 * is truly ambiguous on the current input. If the LL set is smaller, then the
 * SLL conflict resolution might choose an alternative that the full LL would
 * rule out as a possibility based upon better context information. If that's
 * the case, then the SLL parse will definitely get an error because the full LL
 * analysis says it's not viable. If SLL conflict resolution chooses an
 * alternative within the LL set, them both SLL and LL would choose the same
 * alternative because they both choose the minimum of multiple conflicting
 * alternatives.
 *
 * Let's say we have a set of SLL conflicting alternatives `{1, 2, 3}` and
 * a smaller LL set called *s*. If *s* is `{2, 3}`, then SLL
 * parsing will get an error because SLL will pursue alternative 1. If
 * *s* is `{1, 2}` or `{1, 3}` then both SLL and LL will
 * choose the same alternative because alternative one is the minimum of either
 * set. If *s* is `{2}` or `{3}` then SLL will get a syntax
 * error. If *s* is `{1}` then SLL will succeed.
 *
 * Of course, if the input is invalid, then we will get an error for sure in
 * both SLL and LL parsing. Erroneous input will therefore require 2 passes over
 * the input.
 */
export declare class ParserATNSimulator extends ATNSimulator {
    static debug: boolean;
    static dfa_debug: boolean;
    static retry_debug: boolean;
    private predictionMode;
    force_global_context: boolean;
    always_try_local_context: boolean;
    /**
     * Determines whether the DFA is used for full-context predictions. When
     * `true`, the DFA stores transition information for both full-context
     * and SLL parsing; otherwise, the DFA only stores SLL transition
     * information.
     *
     * For some grammars, enabling the full-context DFA can result in a
     * substantial performance improvement. However, this improvement typically
     * comes at the expense of memory used for storing the cached DFA states,
     * configuration sets, and prediction contexts.
     *
     * The default value is `false`.
     */
    enable_global_context_dfa: boolean;
    optimize_unique_closure: boolean;
    optimize_ll1: boolean;
    optimize_tail_calls: boolean;
    tail_call_preserves_sll: boolean;
    treat_sllk1_conflict_as_ambiguity: boolean;
    protected _parser: Parser;
    /**
     * When `true`, ambiguous alternatives are reported when they are
     * encountered within {@link #execATN}. When `false`, these messages
     * are suppressed. The default is `false`.
     *
     * When messages about ambiguous alternatives are not required, setting this
     * to `false` enables additional internal optimizations which may lose
     * this information.
     */
    reportAmbiguities: boolean;
    /** By default we do full context-sensitive LL(*) parsing not
     *  Strong LL(*) parsing. If we fail with Strong LL(*) we
     *  try full LL(*). That means we rewind and use context information
     *  when closure operations fall off the end of the rule that
     *  holds the decision were evaluating.
     */
    protected userWantsCtxSensitive: boolean;
    private dfa?;
    constructor(atn: ATN, parser: Parser);
    getPredictionMode(): PredictionMode;
    setPredictionMode(predictionMode: PredictionMode): void;
    reset(): void;
    adaptivePredict(/*@NotNull*/ input: TokenStream, decision: number, outerContext: ParserRuleContext | undefined): number;
    adaptivePredict(/*@NotNull*/ input: TokenStream, decision: number, outerContext: ParserRuleContext | undefined, useContext: boolean): number;
    protected getStartState(dfa: DFA, input: TokenStream, outerContext: ParserRuleContext, useContext: boolean): SimulatorState | undefined;
    protected execDFA(dfa: DFA, input: TokenStream, startIndex: number, state: SimulatorState): number;
    /**
     * Determines if a particular DFA state should be treated as an accept state
     * for the current prediction mode. In addition to the `useContext`
     * parameter, the {@link #getPredictionMode()} method provides the
     * prediction mode controlling the prediction algorithm as a whole.
     *
     * The default implementation simply returns the value of
     * `DFAState.isAcceptState` except for conflict states when
     * `useContext` is `true` and {@link #getPredictionMode()} is
     * {@link PredictionMode#LL_EXACT_AMBIG_DETECTION}. In that case, only
     * conflict states where {@link ATNConfigSet#isExactConflict} is
     * `true` are considered accept states.
     *
     * @param state The DFA state to check.
     * @param useContext `true` if the prediction algorithm is currently
     * considering the full parser context; otherwise, `false` if the
     * algorithm is currently performing a local context prediction.
     *
     * @returns `true` if the specified `state` is an accept state;
     * otherwise, `false`.
     */
    protected isAcceptState(state: DFAState, useContext: boolean): boolean;
    /** Performs ATN simulation to compute a predicted alternative based
     *  upon the remaining input, but also updates the DFA cache to avoid
     *  having to traverse the ATN again for the same input sequence.
     *
     * There are some key conditions we're looking for after computing a new
     * set of ATN configs (proposed DFA state):
     *
     * * if the set is empty, there is no viable alternative for current symbol
     * * does the state uniquely predict an alternative?
     * * does the state have a conflict that would prevent us from
     *   putting it on the work list?
     * * if in non-greedy decision is there a config at a rule stop state?
     *
     * We also have some key operations to do:
     *
     * * add an edge from previous DFA state to potentially new DFA state, D,
     *   upon current symbol but only if adding to work list, which means in all
     *   cases except no viable alternative (and possibly non-greedy decisions?)
     * * collecting predicates and adding semantic context to DFA accept states
     * * adding rule context to context-sensitive DFA accept states
     * * consuming an input symbol
     * * reporting a conflict
     * * reporting an ambiguity
     * * reporting a context sensitivity
     * * reporting insufficient predicates
     *
     * We should isolate those operations, which are side-effecting, to the
     * main work loop. We can isolate lots of code into other functions, but
     * they should be side effect free. They can return package that
     * indicates whether we should report something, whether we need to add a
     * DFA edge, whether we need to augment accept state with semantic
     * context or rule invocation context. Actually, it seems like we always
     * add predicates if they exist, so that can simply be done in the main
     * loop for any accept state creation or modification request.
     *
     * cover these cases:
     *   dead end
     *   single alt
     *   single alt + preds
     *   conflict
     *   conflict + preds
     *
     * TODO: greedy + those
     */
    protected execATN(dfa: DFA, input: TokenStream, startIndex: number, initialState: SimulatorState): number;
    /**
     * This method is used to improve the localization of error messages by
     * choosing an alternative rather than throwing a
     * {@link NoViableAltException} in particular prediction scenarios where the
     * {@link #ERROR} state was reached during ATN simulation.
     *
     * The default implementation of this method uses the following
     * algorithm to identify an ATN configuration which successfully parsed the
     * decision entry rule. Choosing such an alternative ensures that the
     * {@link ParserRuleContext} returned by the calling rule will be complete
     * and valid, and the syntax error will be reported later at a more
     * localized location.
     *
     * * If no configuration in `configs` reached the end of the
     *   decision rule, return {@link ATN#INVALID_ALT_NUMBER}.
     * * If all configurations in `configs` which reached the end of the
     *   decision rule predict the same alternative, return that alternative.
     * * If the configurations in `configs` which reached the end of the
     *   decision rule predict multiple alternatives (call this *S*),
     *   choose an alternative in the following order.
     *
     *     1. Filter the configurations in `configs` to only those
     *        configurations which remain viable after evaluating semantic predicates.
     *        If the set of these filtered configurations which also reached the end of
     *        the decision rule is not empty, return the minimum alternative
     *        represented in this set.
     *     1. Otherwise, choose the minimum alternative in *S*.
     *
     * In some scenarios, the algorithm described above could predict an
     * alternative which will result in a {@link FailedPredicateException} in
     * parser. Specifically, this could occur if the *only* configuration
     * capable of successfully parsing to the end of the decision rule is
     * blocked by a semantic predicate. By choosing this alternative within
     * {@link #adaptivePredict} instead of throwing a
     * {@link NoViableAltException}, the resulting
     * {@link FailedPredicateException} in the parser will identify the specific
     * predicate which is preventing the parser from successfully parsing the
     * decision rule, which helps developers identify and correct logic errors
     * in semantic predicates.
     *
     * @param input The input {@link TokenStream}
     * @param startIndex The start index for the current prediction, which is
     * the input index where any semantic context in `configs` should be
     * evaluated
     * @param previous The ATN simulation state immediately before the
     * {@link #ERROR} state was reached
     *
     * @returns The value to return from {@link #adaptivePredict}, or
     * {@link ATN#INVALID_ALT_NUMBER} if a suitable alternative was not
     * identified and {@link #adaptivePredict} should report an error instead.
     */
    protected handleNoViableAlt(input: TokenStream, startIndex: number, previous: SimulatorState): number;
    protected computeReachSet(dfa: DFA, previous: SimulatorState, t: number, contextCache: PredictionContextCache): SimulatorState | undefined;
    /**
     * Get an existing target state for an edge in the DFA. If the target state
     * for the edge has not yet been computed or is otherwise not available,
     * this method returns `undefined`.
     *
     * @param s The current DFA state
     * @param t The next input symbol
     * @returns The existing target DFA state for the given input symbol
     * `t`, or `undefined` if the target state for this edge is not
     * already cached
     */
    protected getExistingTargetState(s: DFAState, t: number): DFAState | undefined;
    /**
     * Compute a target state for an edge in the DFA, and attempt to add the
     * computed state and corresponding edge to the DFA.
     *
     * @param dfa
     * @param s The current DFA state
     * @param remainingGlobalContext
     * @param t The next input symbol
     * @param useContext
     * @param contextCache
     *
     * @returns The computed target DFA state for the given input symbol
     * `t`. If `t` does not lead to a valid DFA state, this method
     * returns {@link #ERROR}.
     */
    protected computeTargetState(dfa: DFA, s: DFAState, remainingGlobalContext: ParserRuleContext | undefined, t: number, useContext: boolean, contextCache: PredictionContextCache): [DFAState, ParserRuleContext | undefined];
    /**
     * Return a configuration set containing only the configurations from
     * `configs` which are in a {@link RuleStopState}. If all
     * configurations in `configs` are already in a rule stop state, this
     * method simply returns `configs`.
     *
     * @param configs the configuration set to update
     * @param contextCache the {@link PredictionContext} cache
     *
     * @returns `configs` if all configurations in `configs` are in a
     * rule stop state, otherwise return a new configuration set containing only
     * the configurations from `configs` which are in a rule stop state
     */
    protected removeAllConfigsNotInRuleStopState(configs: ATNConfigSet, contextCache: PredictionContextCache): ATNConfigSet;
    protected computeStartState(dfa: DFA, globalContext: ParserRuleContext, useContext: boolean): SimulatorState;
    /**
     * This method transforms the start state computed by
     * {@link #computeStartState} to the special start state used by a
     * precedence DFA for a particular precedence value. The transformation
     * process applies the following changes to the start state's configuration
     * set.
     *
     * 1. Evaluate the precedence predicates for each configuration using
     *    {@link SemanticContext#evalPrecedence}.
     * 1. When {@link ATNConfig#isPrecedenceFilterSuppressed} is `false`,
     *    remove all configurations which predict an alternative greater than 1,
     *    for which another configuration that predicts alternative 1 is in the
     *    same ATN state with the same prediction context. This transformation is
     *    valid for the following reasons:
     *
     *     * The closure block cannot contain any epsilon transitions which bypass
     *       the body of the closure, so all states reachable via alternative 1 are
     *       part of the precedence alternatives of the transformed left-recursive
     *       rule.
     *     * The "primary" portion of a left recursive rule cannot contain an
     *       epsilon transition, so the only way an alternative other than 1 can exist
     *       in a state that is also reachable via alternative 1 is by nesting calls
     *       to the left-recursive rule, with the outer calls not being at the
     *       preferred precedence level. The
     *       {@link ATNConfig#isPrecedenceFilterSuppressed} property marks ATN
     *       configurations which do not meet this condition, and therefore are not
     *       eligible for elimination during the filtering process.
     *
     * The prediction context must be considered by this filter to address
     * situations like the following.
     *
     * ```antlr
     * grammar TA;
     * prog: statement* EOF;
     * statement: letterA | statement letterA 'b' ;
     * letterA: 'a';
     * ```
     *
     * If the above grammar, the ATN state immediately before the token
     * reference `'a'` in `letterA` is reachable from the left edge
     * of both the primary and closure blocks of the left-recursive rule
     * `statement`. The prediction context associated with each of these
     * configurations distinguishes between them, and prevents the alternative
     * which stepped out to `prog` (and then back in to `statement`
     * from being eliminated by the filter.
     *
     * @param configs The configuration set computed by
     * {@link #computeStartState} as the start state for the DFA.
     * @returns The transformed configuration set representing the start state
     * for a precedence DFA at a particular precedence level (determined by
     * calling {@link Parser#getPrecedence}).
     */
    protected applyPrecedenceFilter(configs: ATNConfigSet, globalContext: ParserRuleContext, contextCache: PredictionContextCache): ATNConfigSet;
    protected getReachableTarget(source: ATNConfig, trans: Transition, ttype: number): ATNState | undefined;
    /** collect and set D's semantic context */
    protected predicateDFAState(D: DFAState, configs: ATNConfigSet, nalts: number): DFAState.PredPrediction[] | undefined;
    protected getPredsForAmbigAlts(ambigAlts: BitSet, configs: ATNConfigSet, nalts: number): SemanticContext[] | undefined;
    protected getPredicatePredictions(ambigAlts: BitSet | undefined, altToPred: SemanticContext[]): DFAState.PredPrediction[] | undefined;
    /** Look through a list of predicate/alt pairs, returning alts for the
     *  pairs that win. An `undefined` predicate indicates an alt containing an
     *  unpredicated config which behaves as "always true."
     */
    protected evalSemanticContext(predPredictions: DFAState.PredPrediction[], outerContext: ParserRuleContext, complete: boolean): BitSet;
    /**
     * Evaluate a semantic context within a specific parser context.
     *
     * This method might not be called for every semantic context evaluated
     * during the prediction process. In particular, we currently do not
     * evaluate the following but it may change in the future:
     *
     * * Precedence predicates (represented by
     *   {@link SemanticContext.PrecedencePredicate}) are not currently evaluated
     *   through this method.
     * * Operator predicates (represented by {@link SemanticContext.AND} and
     *   {@link SemanticContext.OR}) are evaluated as a single semantic
     *   context, rather than evaluating the operands individually.
     *   Implementations which require evaluation results from individual
     *   predicates should override this method to explicitly handle evaluation of
     *   the operands within operator predicates.
     *
     * @param pred The semantic context to evaluate
     * @param parserCallStack The parser context in which to evaluate the
     * semantic context
     * @param alt The alternative which is guarded by `pred`
     *
     * @since 4.3
     */
    protected evalSemanticContextImpl(pred: SemanticContext, parserCallStack: ParserRuleContext, alt: number): boolean;
    protected closure(sourceConfigs: ATNConfigSet, configs: ATNConfigSet, collectPredicates: boolean, hasMoreContext: boolean, contextCache: PredictionContextCache, treatEofAsEpsilon: boolean): void;
    protected closureImpl(config: ATNConfig, configs: ATNConfigSet, intermediate: ATNConfigSet, closureBusy: Array2DHashSet<ATNConfig>, collectPredicates: boolean, hasMoreContexts: boolean, contextCache: PredictionContextCache, depth: number, treatEofAsEpsilon: boolean): void;
    getRuleName(index: number): string;
    protected getEpsilonTarget(config: ATNConfig, t: Transition, collectPredicates: boolean, inContext: boolean, contextCache: PredictionContextCache, treatEofAsEpsilon: boolean): ATNConfig | undefined;
    protected actionTransition(config: ATNConfig, t: ActionTransition): ATNConfig;
    protected precedenceTransition(config: ATNConfig, pt: PrecedencePredicateTransition, collectPredicates: boolean, inContext: boolean): ATNConfig;
    protected predTransition(config: ATNConfig, pt: PredicateTransition, collectPredicates: boolean, inContext: boolean): ATNConfig;
    protected ruleTransition(config: ATNConfig, t: RuleTransition, contextCache: PredictionContextCache): ATNConfig;
    private static STATE_ALT_SORT_COMPARATOR;
    private isConflicted;
    protected getConflictingAltsFromConfigSet(configs: ATNConfigSet): BitSet | undefined;
    getTokenName(t: number): string;
    getLookaheadName(input: TokenStream): string;
    dumpDeadEndConfigs(nvae: NoViableAltException): void;
    protected noViableAlt(input: TokenStream, outerContext: ParserRuleContext, configs: ATNConfigSet, startIndex: number): NoViableAltException;
    protected getUniqueAlt(configs: Iterable<ATNConfig>): number;
    protected configWithAltAtStopState(configs: Iterable<ATNConfig>, alt: number): boolean;
    protected addDFAEdge(dfa: DFA, fromState: DFAState, t: number, contextTransitions: IntegerList | undefined, toConfigs: ATNConfigSet, contextCache: PredictionContextCache): DFAState;
    protected setDFAEdge(p: DFAState, t: number, q: DFAState): void;
    /** See comment on LexerInterpreter.addDFAState. */
    protected addDFAContextState(dfa: DFA, configs: ATNConfigSet, returnContext: number, contextCache: PredictionContextCache): DFAState;
    /** See comment on LexerInterpreter.addDFAState. */
    protected addDFAState(dfa: DFA, configs: ATNConfigSet, contextCache: PredictionContextCache): DFAState;
    protected createDFAState(dfa: DFA, configs: ATNConfigSet): DFAState;
    protected reportAttemptingFullContext(dfa: DFA, conflictingAlts: BitSet | undefined, conflictState: SimulatorState, startIndex: number, stopIndex: number): void;
    protected reportContextSensitivity(dfa: DFA, prediction: number, acceptState: SimulatorState, startIndex: number, stopIndex: number): void;
    /** If context sensitive parsing, we know it's ambiguity not conflict */
    protected reportAmbiguity(dfa: DFA, D: DFAState, // the DFA state from execATN(): void that had SLL conflicts
    startIndex: number, stopIndex: number, exact: boolean, ambigAlts: BitSet, configs: ATNConfigSet): void;
    protected getReturnState(context: RuleContext): number;
    protected skipTailCalls(context: ParserRuleContext): ParserRuleContext;
    /**
     * @since 4.3
     */
    get parser(): Parser;
}
