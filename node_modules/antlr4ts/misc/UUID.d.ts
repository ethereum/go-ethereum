/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { Equatable } from "./Stubs";
export declare class UUID implements Equatable {
    private readonly data;
    constructor(mostSigBits: number, moreSigBits: number, lessSigBits: number, leastSigBits: number);
    static fromString(data: string): UUID;
    hashCode(): number;
    equals(obj: any): boolean;
    toString(): string;
}
