/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { Lexer } from "../Lexer";
import { LexerAction } from "./LexerAction";
import { LexerActionType } from "./LexerActionType";
/**
 * This implementation of {@link LexerAction} is used for tracking input offsets
 * for position-dependent actions within a {@link LexerActionExecutor}.
 *
 * This action is not serialized as part of the ATN, and is only required for
 * position-dependent lexer actions which appear at a location other than the
 * end of a rule. For more information about DFA optimizations employed for
 * lexer actions, see {@link LexerActionExecutor#append} and
 * {@link LexerActionExecutor#fixOffsetBeforeMatch}.
 *
 * @author Sam Harwell
 * @since 4.2
 */
export declare class LexerIndexedCustomAction implements LexerAction {
    private readonly _offset;
    private readonly _action;
    /**
     * Constructs a new indexed custom action by associating a character offset
     * with a {@link LexerAction}.
     *
     * Note: This class is only required for lexer actions for which
     * {@link LexerAction#isPositionDependent} returns `true`.
     *
     * @param offset The offset into the input {@link CharStream}, relative to
     * the token start index, at which the specified lexer action should be
     * executed.
     * @param action The lexer action to execute at a particular offset in the
     * input {@link CharStream}.
     */
    constructor(offset: number, action: LexerAction);
    /**
     * Gets the location in the input {@link CharStream} at which the lexer
     * action should be executed. The value is interpreted as an offset relative
     * to the token start index.
     *
     * @returns The location in the input {@link CharStream} at which the lexer
     * action should be executed.
     */
    get offset(): number;
    /**
     * Gets the lexer action to execute.
     *
     * @returns A {@link LexerAction} object which executes the lexer action.
     */
    get action(): LexerAction;
    /**
     * {@inheritDoc}
     *
     * @returns This method returns the result of calling {@link #getActionType}
     * on the {@link LexerAction} returned by {@link #getAction}.
     */
    get actionType(): LexerActionType;
    /**
     * {@inheritDoc}
     * @returns This method returns `true`.
     */
    get isPositionDependent(): boolean;
    /**
     * {@inheritDoc}
     *
     * This method calls {@link #execute} on the result of {@link #getAction}
     * using the provided `lexer`.
     */
    execute(lexer: Lexer): void;
    hashCode(): number;
    equals(obj: any): boolean;
}
