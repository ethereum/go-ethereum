export declare function fromBig(n: bigint, le?: boolean): {
    h: number;
    l: number;
};
export declare function split(lst: bigint[], le?: boolean): Uint32Array[];
export declare const toBig: (h: number, l: number) => bigint;
export declare function add(Ah: number, Al: number, Bh: number, Bl: number): {
    h: number;
    l: number;
};
declare const u64: {
    fromBig: typeof fromBig;
    split: typeof split;
    toBig: (h: number, l: number) => bigint;
    shrSH: (h: number, l: number, s: number) => number;
    shrSL: (h: number, l: number, s: number) => number;
    rotrSH: (h: number, l: number, s: number) => number;
    rotrSL: (h: number, l: number, s: number) => number;
    rotrBH: (h: number, l: number, s: number) => number;
    rotrBL: (h: number, l: number, s: number) => number;
    rotr32H: (h: number, l: number) => number;
    rotr32L: (h: number, l: number) => number;
    rotlSH: (h: number, l: number, s: number) => number;
    rotlSL: (h: number, l: number, s: number) => number;
    rotlBH: (h: number, l: number, s: number) => number;
    rotlBL: (h: number, l: number, s: number) => number;
    add: typeof add;
    add3L: (Al: number, Bl: number, Cl: number) => number;
    add3H: (low: number, Ah: number, Bh: number, Ch: number) => number;
    add4L: (Al: number, Bl: number, Cl: number, Dl: number) => number;
    add4H: (low: number, Ah: number, Bh: number, Ch: number, Dh: number) => number;
    add5H: (low: number, Ah: number, Bh: number, Ch: number, Dh: number, Eh: number) => number;
    add5L: (Al: number, Bl: number, Cl: number, Dl: number, El: number) => number;
};
export default u64;
