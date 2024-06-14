"use strict";
/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
Object.defineProperty(exports, "__esModule", { value: true });
exports.LexerActionType = void 0;
// ConvertTo-TS run at 2016-10-04T11:26:29.0172086-07:00
/**
 * Represents the serialization type of a {@link LexerAction}.
 *
 * @author Sam Harwell
 * @since 4.2
 */
var LexerActionType;
(function (LexerActionType) {
    /**
     * The type of a {@link LexerChannelAction} action.
     */
    LexerActionType[LexerActionType["CHANNEL"] = 0] = "CHANNEL";
    /**
     * The type of a {@link LexerCustomAction} action.
     */
    LexerActionType[LexerActionType["CUSTOM"] = 1] = "CUSTOM";
    /**
     * The type of a {@link LexerModeAction} action.
     */
    LexerActionType[LexerActionType["MODE"] = 2] = "MODE";
    /**
     * The type of a {@link LexerMoreAction} action.
     */
    LexerActionType[LexerActionType["MORE"] = 3] = "MORE";
    /**
     * The type of a {@link LexerPopModeAction} action.
     */
    LexerActionType[LexerActionType["POP_MODE"] = 4] = "POP_MODE";
    /**
     * The type of a {@link LexerPushModeAction} action.
     */
    LexerActionType[LexerActionType["PUSH_MODE"] = 5] = "PUSH_MODE";
    /**
     * The type of a {@link LexerSkipAction} action.
     */
    LexerActionType[LexerActionType["SKIP"] = 6] = "SKIP";
    /**
     * The type of a {@link LexerTypeAction} action.
     */
    LexerActionType[LexerActionType["TYPE"] = 7] = "TYPE";
})(LexerActionType = exports.LexerActionType || (exports.LexerActionType = {}));
//# sourceMappingURL=LexerActionType.js.map