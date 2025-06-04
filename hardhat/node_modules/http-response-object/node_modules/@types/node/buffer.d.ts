declare module "buffer" {
    export const INSPECT_MAX_BYTES: number;
    const BuffType: typeof Buffer;

    export type TranscodeEncoding = "ascii" | "utf8" | "utf16le" | "ucs2" | "latin1" | "binary";

    export function transcode(source: Buffer | Uint8Array, fromEnc: TranscodeEncoding, toEnc: TranscodeEncoding): Buffer;

    export const SlowBuffer: {
        /** @deprecated since v6.0.0, use Buffer.allocUnsafeSlow() */
        new(size: number): Buffer;
        prototype: Buffer;
    };

    export { BuffType as Buffer };
}
