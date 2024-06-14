/// <reference types="chai" />
/**
 * A predicate for use with .withArgs(...), to induce chai to accept any value
 * as a positive match with the argument.
 *
 * Example: expect(contract.emitInt()).to.emit(contract, "Int").withArgs(anyValue)
 */
export declare function anyValue(): boolean;
/**
 * A predicate for use with .withArgs(...), to induce chai to accept any
 * unsigned integer as a positive match with the argument.
 *
 * Example: expect(contract.emitUint()).to.emit(contract, "Uint").withArgs(anyUint)
 */
export declare function anyUint(i: any): boolean;
export declare function supportWithArgs(Assertion: Chai.AssertionStatic, chaiUtils: Chai.ChaiUtils): void;
//# sourceMappingURL=withArgs.d.ts.map