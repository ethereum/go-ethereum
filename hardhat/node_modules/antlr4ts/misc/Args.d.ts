/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
/**
 * Validates that an argument is not `null` or `undefined`.
 *
 * @param parameterName The name of the parameter
 * @param value The argument value
 *
 * @throws `TypeError` if `value` is `null` or `undefined`.
 */
export declare function notNull(parameterName: string, value: any): void;
