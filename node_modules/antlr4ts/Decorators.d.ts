/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
export declare function NotNull(target: any, propertyKey: PropertyKey, propertyDescriptor?: PropertyDescriptor | number): void;
export declare function Nullable(target: any, propertyKey: PropertyKey, propertyDescriptor?: PropertyDescriptor | number): void;
export declare function Override(target: any, propertyKey: PropertyKey, propertyDescriptor?: PropertyDescriptor): void;
export declare function SuppressWarnings(options: string): (target: any, propertyKey: PropertyKey, descriptor?: PropertyDescriptor | undefined) => void;
