import { Reporter } from './Reporter';
import { ValidationError } from '.';
/**
 * @since 1.0.0
 */
export declare function failure(es: Array<ValidationError>): Array<string>;
/**
 * @since 1.0.0
 */
export declare function success(): Array<string>;
/**
 * @since 1.0.0
 */
export declare const PathReporter: Reporter<Array<string>>;
