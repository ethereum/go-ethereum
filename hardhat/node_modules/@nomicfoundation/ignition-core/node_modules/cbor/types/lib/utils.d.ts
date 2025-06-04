/// <reference types="node" />
export function utf8(buf: any): string;
export namespace utf8 {
    const checksUTF8: boolean;
}
export function isBufferish(b: any): boolean;
export function bufferishToBuffer(b: any): Buffer;
export function parseCBORint(ai: any, buf: any): any;
export function writeHalf(buf: any, half: any): boolean;
export function parseHalf(buf: any): number;
export function parseCBORfloat(buf: any): any;
export function hex(s: any): Buffer;
export function bin(s: any): Buffer;
export function arrayEqual(a: any, b: any): any;
export function bufferToBigInt(buf: any): bigint;
export function cborValueToString(val: any, float_bytes?: number): any;
export function guessEncoding(input: any, encoding: any): any;
export function base64url(buf: Buffer | Uint8Array | Uint8ClampedArray | ArrayBuffer | DataView): string;
export function base64(buf: Buffer | Uint8Array | Uint8ClampedArray | ArrayBuffer | DataView): string;
export function isBigEndian(): boolean;
import { Buffer } from "buffer";
