import { Event, Exception, ExtendedError, StackFrame } from '@sentry/types';
import { NodeOptions } from './backend';
import * as stacktrace from './stacktrace';
/**
 * Resets the file cache. Exists for testing purposes.
 * @hidden
 */
export declare function resetFileContentCache(): void;
/**
 * @hidden
 */
export declare function extractStackFromError(error: Error): stacktrace.StackFrame[];
/**
 * @hidden
 */
export declare function parseStack(stack: stacktrace.StackFrame[], options?: NodeOptions): PromiseLike<StackFrame[]>;
/**
 * @hidden
 */
export declare function getExceptionFromError(error: Error, options?: NodeOptions): PromiseLike<Exception>;
/**
 * @hidden
 */
export declare function parseError(error: ExtendedError, options?: NodeOptions): PromiseLike<Event>;
/**
 * @hidden
 */
export declare function prepareFramesForEvent(stack: StackFrame[]): StackFrame[];
//# sourceMappingURL=parsers.d.ts.map