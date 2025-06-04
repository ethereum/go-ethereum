/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { Comparable } from "../misc/Stubs";
import { Equatable } from "../misc/Stubs";
import { Recognizer } from "../Recognizer";
import { RuleContext } from "../RuleContext";
/** A tree structure used to record the semantic context in which
 *  an ATN configuration is valid.  It's either a single predicate,
 *  a conjunction `p1&&p2`, or a sum of products `p1||p2`.
 *
 *  I have scoped the {@link AND}, {@link OR}, and {@link Predicate} subclasses of
 *  {@link SemanticContext} within the scope of this outer class.
 */
export declare abstract class SemanticContext implements Equatable {
    private static _NONE;
    /**
     * The default {@link SemanticContext}, which is semantically equivalent to
     * a predicate of the form `{true}?`.
     */
    static get NONE(): SemanticContext;
    /**
     * For context independent predicates, we evaluate them without a local
     * context (i.e., unedfined context). That way, we can evaluate them without
     * having to create proper rule-specific context during prediction (as
     * opposed to the parser, which creates them naturally). In a practical
     * sense, this avoids a cast exception from RuleContext to myruleContext.
     *
     * For context dependent predicates, we must pass in a local context so that
     * references such as $arg evaluate properly as _localctx.arg. We only
     * capture context dependent predicates in the context in which we begin
     * prediction, so we passed in the outer context here in case of context
     * dependent predicate evaluation.
     */
    abstract eval<T>(parser: Recognizer<T, any>, parserCallStack: RuleContext): boolean;
    /**
     * Evaluate the precedence predicates for the context and reduce the result.
     *
     * @param parser The parser instance.
     * @param parserCallStack
     * @returns The simplified semantic context after precedence predicates are
     * evaluated, which will be one of the following values.
     *
     * * {@link #NONE}: if the predicate simplifies to `true` after
     *   precedence predicates are evaluated.
     * * `undefined`: if the predicate simplifies to `false` after
     *   precedence predicates are evaluated.
     * * `this`: if the semantic context is not changed as a result of
     *   precedence predicate evaluation.
     * * A non-`undefined` {@link SemanticContext}: the new simplified
     *   semantic context after precedence predicates are evaluated.
     */
    evalPrecedence(parser: Recognizer<any, any>, parserCallStack: RuleContext): SemanticContext | undefined;
    abstract hashCode(): number;
    abstract equals(obj: any): boolean;
    static and(a: SemanticContext | undefined, b: SemanticContext): SemanticContext;
    /**
     *
     *  @see ParserATNSimulator#getPredsForAmbigAlts
     */
    static or(a: SemanticContext | undefined, b: SemanticContext): SemanticContext;
}
export declare namespace SemanticContext {
    class Predicate extends SemanticContext {
        ruleIndex: number;
        predIndex: number;
        isCtxDependent: boolean;
        constructor();
        constructor(ruleIndex: number, predIndex: number, isCtxDependent: boolean);
        eval<T>(parser: Recognizer<T, any>, parserCallStack: RuleContext): boolean;
        hashCode(): number;
        equals(obj: any): boolean;
        toString(): string;
    }
    class PrecedencePredicate extends SemanticContext implements Comparable<PrecedencePredicate> {
        precedence: number;
        constructor(precedence: number);
        eval<T>(parser: Recognizer<T, any>, parserCallStack: RuleContext): boolean;
        evalPrecedence(parser: Recognizer<any, any>, parserCallStack: RuleContext): SemanticContext | undefined;
        compareTo(o: PrecedencePredicate): number;
        hashCode(): number;
        equals(obj: any): boolean;
        toString(): string;
    }
    /**
     * This is the base class for semantic context "operators", which operate on
     * a collection of semantic context "operands".
     *
     * @since 4.3
     */
    abstract class Operator extends SemanticContext {
        /**
         * Gets the operands for the semantic context operator.
         *
         * @returns a collection of {@link SemanticContext} operands for the
         * operator.
         *
         * @since 4.3
         */
        abstract readonly operands: Iterable<SemanticContext>;
    }
    /**
     * A semantic context which is true whenever none of the contained contexts
     * is false.
     */
    class AND extends Operator {
        opnds: SemanticContext[];
        constructor(a: SemanticContext, b: SemanticContext);
        get operands(): Iterable<SemanticContext>;
        equals(obj: any): boolean;
        hashCode(): number;
        /**
         * {@inheritDoc}
         *
         * The evaluation of predicates by this context is short-circuiting, but
         * unordered.
         */
        eval<T>(parser: Recognizer<T, any>, parserCallStack: RuleContext): boolean;
        evalPrecedence(parser: Recognizer<any, any>, parserCallStack: RuleContext): SemanticContext | undefined;
        toString(): string;
    }
    /**
     * A semantic context which is true whenever at least one of the contained
     * contexts is true.
     */
    class OR extends Operator {
        opnds: SemanticContext[];
        constructor(a: SemanticContext, b: SemanticContext);
        get operands(): Iterable<SemanticContext>;
        equals(obj: any): boolean;
        hashCode(): number;
        /**
         * {@inheritDoc}
         *
         * The evaluation of predicates by this context is short-circuiting, but
         * unordered.
         */
        eval<T>(parser: Recognizer<T, any>, parserCallStack: RuleContext): boolean;
        evalPrecedence(parser: Recognizer<any, any>, parserCallStack: RuleContext): SemanticContext | undefined;
        toString(): string;
    }
}
