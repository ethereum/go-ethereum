/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { Token } from "./Token";
export interface WritableToken extends Token {
    text: string | undefined;
    type: number;
    line: number;
    charPositionInLine: number;
    channel: number;
    tokenIndex: number;
}
