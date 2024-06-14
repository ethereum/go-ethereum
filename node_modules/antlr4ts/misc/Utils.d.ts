/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { Equatable } from "./Stubs";
import { IntegerList } from "./IntegerList";
export declare function escapeWhitespace(s: string, escapeSpaces: boolean): string;
export declare function join(collection: Iterable<any>, separator: string): string;
export declare function equals(x: Equatable | undefined, y: Equatable | undefined): boolean;
/** Convert array of strings to string&rarr;index map. Useful for
 *  converting rulenames to name&rarr;ruleindex map.
 */
export declare function toMap(keys: string[]): Map<string, number>;
export declare function toCharArray(str: string): Uint16Array;
export declare function toCharArray(data: IntegerList): Uint16Array;
