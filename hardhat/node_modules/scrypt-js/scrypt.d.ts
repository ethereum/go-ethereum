
export as namespace scrypt;

export type ProgressCallback = (progress: number) => boolean | void;

export function scrypt(
    password: ArrayLike<number>,
    salt: ArrayLike<number>,
    N: number,
    r: number,
    p: number,
    dkLen: number,
    callback?: ProgressCallback
): Promise<Uint8Array>;

export function syncScrypt(
    password: ArrayLike<number>,
    salt: ArrayLike<number>,
    N: number,
    r: number,
    p: number,
    dkLen: number
): Uint8Array;
