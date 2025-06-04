/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { Interval } from "./misc/Interval";
import { Token } from "./Token";
import { TokenStream } from "./TokenStream";
/**
 * Useful for rewriting out a buffered input token stream after doing some
 * augmentation or other manipulations on it.
 *
 * You can insert stuff, replace, and delete chunks. Note that the operations
 * are done lazily--only if you convert the buffer to a {@link String} with
 * {@link TokenStream#getText()}. This is very efficient because you are not
 * moving data around all the time. As the buffer of tokens is converted to
 * strings, the {@link #getText()} method(s) scan the input token stream and
 * check to see if there is an operation at the current index. If so, the
 * operation is done and then normal {@link String} rendering continues on the
 * buffer. This is like having multiple Turing machine instruction streams
 * (programs) operating on a single input tape. :)
 *
 * This rewriter makes no modifications to the token stream. It does not ask the
 * stream to fill itself up nor does it advance the input cursor. The token
 * stream `TokenStream.index` will return the same value before and
 * after any {@link #getText()} call.
 *
 * The rewriter only works on tokens that you have in the buffer and ignores the
 * current input cursor. If you are buffering tokens on-demand, calling
 * {@link #getText()} halfway through the input will only do rewrites for those
 * tokens in the first half of the file.
 *
 * Since the operations are done lazily at {@link #getText}-time, operations do
 * not screw up the token index values. That is, an insert operation at token
 * index `i` does not change the index values for tokens
 * `i`+1..n-1.
 *
 * Because operations never actually alter the buffer, you may always get the
 * original token stream back without undoing anything. Since the instructions
 * are queued up, you can easily simulate transactions and roll back any changes
 * if there is an error just by removing instructions. For example,
 *
 * ```
 * CharStream input = new ANTLRFileStream("input");
 * TLexer lex = new TLexer(input);
 * CommonTokenStream tokens = new CommonTokenStream(lex);
 * T parser = new T(tokens);
 * TokenStreamRewriter rewriter = new TokenStreamRewriter(tokens);
 * parser.startRule();
 * ```
 *
 * Then in the rules, you can execute (assuming rewriter is visible):
 *
 * ```
 * Token t,u;
 * ...
 * rewriter.insertAfter(t, "text to put after t");}
 * rewriter.insertAfter(u, "text after u");}
 * System.out.println(rewriter.getText());
 * ```
 *
 * You can also have multiple "instruction streams" and get multiple rewrites
 * from a single pass over the input. Just name the instruction streams and use
 * that name again when printing the buffer. This could be useful for generating
 * a C file and also its header file--all from the same buffer:
 *
 * ```
 * rewriter.insertAfter("pass1", t, "text to put after t");}
 * rewriter.insertAfter("pass2", u, "text after u");}
 * System.out.println(rewriter.getText("pass1"));
 * System.out.println(rewriter.getText("pass2"));
 * ```
 *
 * If you don't use named rewrite streams, a "default" stream is used as the
 * first example shows.
 */
export declare class TokenStreamRewriter {
    static readonly DEFAULT_PROGRAM_NAME: string;
    static readonly PROGRAM_INIT_SIZE: number;
    static readonly MIN_TOKEN_INDEX: number;
    /** Our source stream */
    protected tokens: TokenStream;
    /** You may have multiple, named streams of rewrite operations.
     *  I'm calling these things "programs."
     *  Maps String (name) &rarr; rewrite (List)
     */
    protected programs: Map<string, RewriteOperation[]>;
    /** Map String (program name) &rarr; Integer index */
    protected lastRewriteTokenIndexes: Map<string, number>;
    constructor(tokens: TokenStream);
    getTokenStream(): TokenStream;
    rollback(instructionIndex: number): void;
    /** Rollback the instruction stream for a program so that
     *  the indicated instruction (via instructionIndex) is no
     *  longer in the stream. UNTESTED!
     */
    rollback(instructionIndex: number, programName: string): void;
    deleteProgram(): void;
    /** Reset the program so that no instructions exist */
    deleteProgram(programName: string): void;
    insertAfter(t: Token, text: {}): void;
    insertAfter(index: number, text: {}): void;
    insertAfter(t: Token, text: {}, programName: string): void;
    insertAfter(index: number, text: {}, programName: string): void;
    insertBefore(t: Token, text: {}): void;
    insertBefore(index: number, text: {}): void;
    insertBefore(t: Token, text: {}, programName: string): void;
    insertBefore(index: number, text: {}, programName: string): void;
    replaceSingle(index: number, text: {}): void;
    replaceSingle(indexT: Token, text: {}): void;
    replace(from: number, to: number, text: {}): void;
    replace(from: Token, to: Token, text: {}): void;
    replace(from: number, to: number, text: {}, programName: string): void;
    replace(from: Token, to: Token, text: {}, programName: string): void;
    delete(index: number): void;
    delete(from: number, to: number): void;
    delete(indexT: Token): void;
    delete(from: Token, to: Token): void;
    delete(from: number, to: number, programName: string): void;
    delete(from: Token, to: Token, programName: string): void;
    protected getLastRewriteTokenIndex(): number;
    protected getLastRewriteTokenIndex(programName: string): number;
    protected setLastRewriteTokenIndex(programName: string, i: number): void;
    protected getProgram(name: string): RewriteOperation[];
    private initializeProgram;
    /** Return the text from the original tokens altered per the
     *  instructions given to this rewriter.
     */
    getText(): string;
    /** Return the text from the original tokens altered per the
     *  instructions given to this rewriter in programName.
     *
     * @since 4.5
     */
    getText(programName: string): string;
    /** Return the text associated with the tokens in the interval from the
     *  original token stream but with the alterations given to this rewriter.
     *  The interval refers to the indexes in the original token stream.
     *  We do not alter the token stream in any way, so the indexes
     *  and intervals are still consistent. Includes any operations done
     *  to the first and last token in the interval. So, if you did an
     *  insertBefore on the first token, you would get that insertion.
     *  The same is true if you do an insertAfter the stop token.
     */
    getText(interval: Interval): string;
    getText(interval: Interval, programName: string): string;
    /** We need to combine operations and report invalid operations (like
     *  overlapping replaces that are not completed nested). Inserts to
     *  same index need to be combined etc...  Here are the cases:
     *
     *  I.i.u I.j.v								leave alone, nonoverlapping
     *  I.i.u I.i.v								combine: Iivu
     *
     *  R.i-j.u R.x-y.v	| i-j in x-y			delete first R
     *  R.i-j.u R.i-j.v							delete first R
     *  R.i-j.u R.x-y.v	| x-y in i-j			ERROR
     *  R.i-j.u R.x-y.v	| boundaries overlap	ERROR
     *
     *  Delete special case of replace (text==undefined):
     *  D.i-j.u D.x-y.v	| boundaries overlap	combine to max(min)..max(right)
     *
     *  I.i.u R.x-y.v | i in (x+1)-y			delete I (since insert before
     * 											we're not deleting i)
     *  I.i.u R.x-y.v | i not in (x+1)-y		leave alone, nonoverlapping
     *  R.x-y.v I.i.u | i in x-y				ERROR
     *  R.x-y.v I.x.u 							R.x-y.uv (combine, delete I)
     *  R.x-y.v I.i.u | i not in x-y			leave alone, nonoverlapping
     *
     *  I.i.u = insert u before op @ index i
     *  R.x-y.u = replace x-y indexed tokens with u
     *
     *  First we need to examine replaces. For any replace op:
     *
     * 		1. wipe out any insertions before op within that range.
     * 		2. Drop any replace op before that is contained completely within
     * 	 that range.
     * 		3. Throw exception upon boundary overlap with any previous replace.
     *
     *  Then we can deal with inserts:
     *
     * 		1. for any inserts to same index, combine even if not adjacent.
     * 		2. for any prior replace with same left boundary, combine this
     * 	 insert with replace and delete this replace.
     * 		3. throw exception if index in same range as previous replace
     *
     *  Don't actually delete; make op undefined in list. Easier to walk list.
     *  Later we can throw as we add to index &rarr; op map.
     *
     *  Note that I.2 R.2-2 will wipe out I.2 even though, technically, the
     *  inserted stuff would be before the replace range. But, if you
     *  add tokens in front of a method body '{' and then delete the method
     *  body, I think the stuff before the '{' you added should disappear too.
     *
     *  Return a map from token index to operation.
     */
    protected reduceToSingleOperationPerIndex(rewrites: Array<RewriteOperation | undefined>): Map<number, RewriteOperation>;
    protected catOpText(a: {}, b: {}): string;
    /** Get all operations before an index of a particular kind */
    protected getKindOfOps<T extends RewriteOperation>(rewrites: Array<RewriteOperation | undefined>, kind: {
        new (...args: any[]): T;
    }, before: number): T[];
}
export declare class RewriteOperation {
    protected readonly tokens: TokenStream;
    /** What index into rewrites List are we? */
    readonly instructionIndex: number;
    /** Token buffer index. */
    index: number;
    text: {};
    constructor(tokens: TokenStream, index: number, instructionIndex: number);
    constructor(tokens: TokenStream, index: number, instructionIndex: number, text: {});
    /** Execute the rewrite operation by possibly adding to the buffer.
     *  Return the index of the next token to operate on.
     */
    execute(buf: string[]): number;
    toString(): string;
}
