/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
/**
 * Represents the serialization type of a {@link LexerAction}.
 *
 * @author Sam Harwell
 * @since 4.2
 */
export declare const enum LexerActionType {
    /**
     * The type of a {@link LexerChannelAction} action.
     */
    CHANNEL = 0,
    /**
     * The type of a {@link LexerCustomAction} action.
     */
    CUSTOM = 1,
    /**
     * The type of a {@link LexerModeAction} action.
     */
    MODE = 2,
    /**
     * The type of a {@link LexerMoreAction} action.
     */
    MORE = 3,
    /**
     * The type of a {@link LexerPopModeAction} action.
     */
    POP_MODE = 4,
    /**
     * The type of a {@link LexerPushModeAction} action.
     */
    PUSH_MODE = 5,
    /**
     * The type of a {@link LexerSkipAction} action.
     */
    SKIP = 6,
    /**
     * The type of a {@link LexerTypeAction} action.
     */
    TYPE = 7
}
