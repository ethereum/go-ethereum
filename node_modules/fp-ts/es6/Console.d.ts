/**
 * @file Adapted from https://github.com/purescript/purescript-console
 */
import { IO } from './IO';
/**
 * @since 1.0.0
 */
export declare const log: (s: unknown) => IO<void>;
/**
 * @since 1.0.0
 */
export declare const warn: (s: unknown) => IO<void>;
/**
 * @since 1.0.0
 */
export declare const error: (s: unknown) => IO<void>;
/**
 * @since 1.0.0
 */
export declare const info: (s: unknown) => IO<void>;
