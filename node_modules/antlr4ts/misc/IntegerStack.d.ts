/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { IntegerList } from "./IntegerList";
/**
 *
 * @author Sam Harwell
 */
export declare class IntegerStack extends IntegerList {
    constructor(arg?: number | IntegerStack);
    push(value: number): void;
    pop(): number;
    peek(): number;
}
