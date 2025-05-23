/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { Lexer } from "../Lexer";
import { LexerAction } from "./LexerAction";
import { LexerActionType } from "./LexerActionType";
/**
 * Implements the `mode` lexer action by calling {@link Lexer#mode} with
 * the assigned mode.
 *
 * @author Sam Harwell
 * @since 4.2
 */
export declare class LexerModeAction implements LexerAction {
    private readonly _mode;
    /**
     * Constructs a new `mode` action with the specified mode value.
     * @param mode The mode value to pass to {@link Lexer#mode}.
     */
    constructor(mode: number);
    /**
     * Get the lexer mode this action should transition the lexer to.
     *
     * @returns The lexer mode for this `mode` command.
     */
    get mode(): number;
    /**
     * {@inheritDoc}
     * @returns This method returns {@link LexerActionType#MODE}.
     */
    get actionType(): LexerActionType;
    /**
     * {@inheritDoc}
     * @returns This method returns `false`.
     */
    get isPositionDependent(): boolean;
    /**
     * {@inheritDoc}
     *
     * This action is implemented by calling {@link Lexer#mode} with the
     * value provided by {@link #getMode}.
     */
    execute(lexer: Lexer): void;
    hashCode(): number;
    equals(obj: any): boolean;
    toString(): string;
}
