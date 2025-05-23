/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { Chunk } from "./Chunk";
import { Lexer } from "../../Lexer";
import { MultiMap } from "../../misc/MultiMap";
import { Parser } from "../../Parser";
import { ParseTree } from "../ParseTree";
import { ParseTreeMatch } from "./ParseTreeMatch";
import { ParseTreePattern } from "./ParseTreePattern";
import { RuleTagToken } from "./RuleTagToken";
import { Token } from "../../Token";
/**
 * A tree pattern matching mechanism for ANTLR {@link ParseTree}s.
 *
 * Patterns are strings of source input text with special tags representing
 * token or rule references such as:
 *
 * ```
 * <ID> = <expr>;
 * ```
 *
 * Given a pattern start rule such as `statement`, this object constructs
 * a {@link ParseTree} with placeholders for the `ID` and `expr`
 * subtree. Then the {@link #match} routines can compare an actual
 * {@link ParseTree} from a parse with this pattern. Tag `<ID>` matches
 * any `ID` token and tag `<expr>` references the result of the
 * `expr` rule (generally an instance of `ExprContext`.
 *
 * Pattern `x = 0;` is a similar pattern that matches the same pattern
 * except that it requires the identifier to be `x` and the expression to
 * be `0`.
 *
 * The {@link #matches} routines return `true` or `false` based
 * upon a match for the tree rooted at the parameter sent in. The
 * {@link #match} routines return a {@link ParseTreeMatch} object that
 * contains the parse tree, the parse tree pattern, and a map from tag name to
 * matched nodes (more below). A subtree that fails to match, returns with
 * {@link ParseTreeMatch#mismatchedNode} set to the first tree node that did not
 * match.
 *
 * For efficiency, you can compile a tree pattern in string form to a
 * {@link ParseTreePattern} object.
 *
 * See `TestParseTreeMatcher` for lots of examples.
 * {@link ParseTreePattern} has two static helper methods:
 * {@link ParseTreePattern#findAll} and {@link ParseTreePattern#match} that
 * are easy to use but not super efficient because they create new
 * {@link ParseTreePatternMatcher} objects each time and have to compile the
 * pattern in string form before using it.
 *
 * The lexer and parser that you pass into the {@link ParseTreePatternMatcher}
 * constructor are used to parse the pattern in string form. The lexer converts
 * the `<ID> = <expr>;` into a sequence of four tokens (assuming lexer
 * throws out whitespace or puts it on a hidden channel). Be aware that the
 * input stream is reset for the lexer (but not the parser; a
 * {@link ParserInterpreter} is created to parse the input.). Any user-defined
 * fields you have put into the lexer might get changed when this mechanism asks
 * it to scan the pattern string.
 *
 * Normally a parser does not accept token `<expr>` as a valid
 * `expr` but, from the parser passed in, we create a special version of
 * the underlying grammar representation (an {@link ATN}) that allows imaginary
 * tokens representing rules (`<expr>`) to match entire rules. We call
 * these *bypass alternatives*.
 *
 * Delimiters are `<`} and `>`}, with `\` as the escape string
 * by default, but you can set them to whatever you want using
 * {@link #setDelimiters}. You must escape both start and stop strings
 * `\<` and `\>`.
 */
export declare class ParseTreePatternMatcher {
    /**
     * This is the backing field for `lexer`.
     */
    private _lexer;
    /**
     * This is the backing field for `parser`.
     */
    private _parser;
    protected start: string;
    protected stop: string;
    protected escape: string;
    /**
     * Regular expression corresponding to escape, for global replace
     */
    protected escapeRE: RegExp;
    /**
     * Constructs a {@link ParseTreePatternMatcher} or from a {@link Lexer} and
     * {@link Parser} object. The lexer input stream is altered for tokenizing
     * the tree patterns. The parser is used as a convenient mechanism to get
     * the grammar name, plus token, rule names.
     */
    constructor(lexer: Lexer, parser: Parser);
    /**
     * Set the delimiters used for marking rule and token tags within concrete
     * syntax used by the tree pattern parser.
     *
     * @param start The start delimiter.
     * @param stop The stop delimiter.
     * @param escapeLeft The escape sequence to use for escaping a start or stop delimiter.
     *
     * @throws {@link Error} if `start` is not defined or empty.
     * @throws {@link Error} if `stop` is not defined or empty.
     */
    setDelimiters(start: string, stop: string, escapeLeft: string): void;
    /** Does `pattern` matched as rule `patternRuleIndex` match `tree`? */
    matches(tree: ParseTree, pattern: string, patternRuleIndex: number): boolean;
    /** Does `pattern` matched as rule patternRuleIndex match tree? Pass in a
     *  compiled pattern instead of a string representation of a tree pattern.
     */
    matches(tree: ParseTree, pattern: ParseTreePattern): boolean;
    /**
     * Compare `pattern` matched as rule `patternRuleIndex` against
     * `tree` and return a {@link ParseTreeMatch} object that contains the
     * matched elements, or the node at which the match failed.
     */
    match(tree: ParseTree, pattern: string, patternRuleIndex: number): ParseTreeMatch;
    /**
     * Compare `pattern` matched against `tree` and return a
     * {@link ParseTreeMatch} object that contains the matched elements, or the
     * node at which the match failed. Pass in a compiled pattern instead of a
     * string representation of a tree pattern.
     */
    match(tree: ParseTree, pattern: ParseTreePattern): ParseTreeMatch;
    /**
     * For repeated use of a tree pattern, compile it to a
     * {@link ParseTreePattern} using this method.
     */
    compile(pattern: string, patternRuleIndex: number): ParseTreePattern;
    /**
     * Used to convert the tree pattern string into a series of tokens. The
     * input stream is reset.
     */
    get lexer(): Lexer;
    /**
     * Used to collect to the grammar file name, token names, rule names for
     * used to parse the pattern into a parse tree.
     */
    get parser(): Parser;
    /**
     * Recursively walk `tree` against `patternTree`, filling
     * `match.`{@link ParseTreeMatch#labels labels}.
     *
     * @returns the first node encountered in `tree` which does not match
     * a corresponding node in `patternTree`, or `undefined` if the match
     * was successful. The specific node returned depends on the matching
     * algorithm used by the implementation, and may be overridden.
     */
    protected matchImpl(tree: ParseTree, patternTree: ParseTree, labels: MultiMap<string, ParseTree>): ParseTree | undefined;
    /** Is `t` `(expr <expr>)` subtree? */
    protected getRuleTagToken(t: ParseTree): RuleTagToken | undefined;
    tokenize(pattern: string): Token[];
    /** Split `<ID> = <e:expr> ;` into 4 chunks for tokenizing by {@link #tokenize}. */
    split(pattern: string): Chunk[];
}
export declare namespace ParseTreePatternMatcher {
    class CannotInvokeStartRule extends Error {
        error: Error;
        constructor(error: Error);
    }
    class StartRuleDoesNotConsumeFullPattern extends Error {
        constructor();
    }
}
