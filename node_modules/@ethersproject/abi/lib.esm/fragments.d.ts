import { BigNumber } from "@ethersproject/bignumber";
export interface JsonFragmentType {
    readonly name?: string;
    readonly indexed?: boolean;
    readonly type?: string;
    readonly internalType?: any;
    readonly components?: ReadonlyArray<JsonFragmentType>;
}
export interface JsonFragment {
    readonly name?: string;
    readonly type?: string;
    readonly anonymous?: boolean;
    readonly payable?: boolean;
    readonly constant?: boolean;
    readonly stateMutability?: string;
    readonly inputs?: ReadonlyArray<JsonFragmentType>;
    readonly outputs?: ReadonlyArray<JsonFragmentType>;
    readonly gas?: string;
}
export declare const FormatTypes: {
    [name: string]: string;
};
export declare class ParamType {
    readonly name: string;
    readonly type: string;
    readonly baseType: string;
    readonly indexed: boolean;
    readonly components: Array<ParamType>;
    readonly arrayLength: number;
    readonly arrayChildren: ParamType;
    readonly _isParamType: boolean;
    constructor(constructorGuard: any, params: any);
    format(format?: string): string;
    static from(value: string | JsonFragmentType | ParamType, allowIndexed?: boolean): ParamType;
    static fromObject(value: JsonFragmentType | ParamType): ParamType;
    static fromString(value: string, allowIndexed?: boolean): ParamType;
    static isParamType(value: any): value is ParamType;
}
export declare abstract class Fragment {
    readonly type: string;
    readonly name: string;
    readonly inputs: Array<ParamType>;
    readonly _isFragment: boolean;
    constructor(constructorGuard: any, params: any);
    abstract format(format?: string): string;
    static from(value: Fragment | JsonFragment | string): Fragment;
    static fromObject(value: Fragment | JsonFragment): Fragment;
    static fromString(value: string): Fragment;
    static isFragment(value: any): value is Fragment;
}
export declare class EventFragment extends Fragment {
    readonly anonymous: boolean;
    format(format?: string): string;
    static from(value: EventFragment | JsonFragment | string): EventFragment;
    static fromObject(value: JsonFragment | EventFragment): EventFragment;
    static fromString(value: string): EventFragment;
    static isEventFragment(value: any): value is EventFragment;
}
export declare class ConstructorFragment extends Fragment {
    stateMutability: string;
    payable: boolean;
    gas?: BigNumber;
    format(format?: string): string;
    static from(value: ConstructorFragment | JsonFragment | string): ConstructorFragment;
    static fromObject(value: ConstructorFragment | JsonFragment): ConstructorFragment;
    static fromString(value: string): ConstructorFragment;
    static isConstructorFragment(value: any): value is ConstructorFragment;
}
export declare class FunctionFragment extends ConstructorFragment {
    constant: boolean;
    outputs?: Array<ParamType>;
    format(format?: string): string;
    static from(value: FunctionFragment | JsonFragment | string): FunctionFragment;
    static fromObject(value: FunctionFragment | JsonFragment): FunctionFragment;
    static fromString(value: string): FunctionFragment;
    static isFunctionFragment(value: any): value is FunctionFragment;
}
export declare class ErrorFragment extends Fragment {
    format(format?: string): string;
    static from(value: ErrorFragment | JsonFragment | string): ErrorFragment;
    static fromObject(value: ErrorFragment | JsonFragment): ErrorFragment;
    static fromString(value: string): ErrorFragment;
    static isErrorFragment(value: any): value is ErrorFragment;
}
//# sourceMappingURL=fragments.d.ts.map