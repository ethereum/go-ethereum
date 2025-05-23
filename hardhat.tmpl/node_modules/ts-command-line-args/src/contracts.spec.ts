/* eslint-disable @typescript-eslint/ban-ts-comment */
/* eslint-disable @typescript-eslint/no-unused-vars */

import { ArgumentConfig } from './contracts';

/**
 * This file is just used for testing type checking at compile time using the // @ts-expect-error feature
 */
describe('contracts', () => {
    describe('ArgumentConfig', () => {
        describe('simple properties', () => {
            interface AllRequired {
                name: string;
                age: number;
                member: boolean;
            }

            it('should allow object with sample values', () => {
                const sampleConfig: ArgumentConfig<AllRequired> = {
                    name: String,
                    age: Number,
                    member: Boolean,
                };
            });

            it('should allow an object with type option definitions', () => {
                const config: ArgumentConfig<AllRequired> = {
                    name: { type: String },
                    age: { type: Number },
                    member: { type: Boolean },
                };
            });

            it('should not allow missing properties', () => {
                // @ts-expect-error
                const config: ArgumentConfig<AllRequired> = {
                    name: String,
                    age: Number,
                };
            });

            it('sample values alone should not allow optional properties', () => {
                interface OptionalProperties {
                    name?: string;
                    age?: number;
                    member?: boolean;
                }

                const config: ArgumentConfig<OptionalProperties> = {
                    // @ts-expect-error
                    name: String,
                    // @ts-expect-error
                    age: Number,
                    // @ts-expect-error
                    member: Boolean,
                };
            });

            it('should not allow arrays', () => {
                interface ArrayProperties {
                    name: string[];
                    age: number[];
                    member: boolean[];
                }

                const config: ArgumentConfig<ArrayProperties> = {
                    // @ts-expect-error
                    name: String,
                    // @ts-expect-error
                    age: Number,
                    // @ts-expect-error
                    member: Boolean,
                };
            });

            it('should not allow wrong type constructor', () => {
                const configSample: ArgumentConfig<AllRequired> = {
                    // @ts-expect-error
                    name: Number,
                    age: Number,
                    member: Boolean,
                };

                const configTypeOption: ArgumentConfig<AllRequired> = {
                    // @ts-expect-error
                    name: { type: Number },
                    age: { type: Number },
                    member: { type: Boolean },
                };
            });
        });

        describe('complex properties', () => {
            interface ComplexProperties {
                requiredStringOne: string;
                requiredStringTwo: string;
                optionalString?: string;
                requiredArray: string[];
                optionalArray?: string[];
            }

            it('should not allow object with sample values', () => {
                const config: ArgumentConfig<ComplexProperties> = {
                    // @ts-expect-error
                    optionalString: String,
                    // @ts-expect-error
                    requiredArray: String,
                    // @ts-expect-error
                    optionalArray: String,
                };
            });

            it('should allow an object with type option definitions', () => {
                const config: ArgumentConfig<ComplexProperties> = {
                    requiredStringOne: String,
                    requiredStringTwo: { type: String },
                    optionalString: { type: String, optional: true },
                    requiredArray: { type: String, multiple: true },
                    optionalArray: { type: String, lazyMultiple: true, optional: true },
                };
            });

            it('should not allow missing properties', () => {
                // @ts-expect-error
                const config: ArgumentConfig<ComplexProperties> = {
                    requiredStringOne: String,
                    requiredStringTwo: { type: String },
                    requiredArray: { type: String, multiple: true },
                    optionalArray: { type: String, multiple: true, optional: true },
                };
            });

            it('should not allow wrong type constructor', () => {
                const configTypeOption: ArgumentConfig<ComplexProperties> = {
                    // @ts-expect-error
                    requiredStringOne: Number,
                    // @ts-expect-error
                    requiredStringTwo: { type: Number },
                    optionalString: { type: String, optional: true },
                    requiredArray: { type: String, multiple: true },
                    optionalArray: { type: String, multiple: true, optional: true },
                };
            });

            it('should allow a complex type with an associated constructor', () => {
                interface IMyComplexType {
                    name: string;
                }

                interface IExpectedArgs {
                    complex: IMyComplexType;
                }

                const configTypeOption: ArgumentConfig<IExpectedArgs> = {
                    complex: { type: (value) => (value ? { name: value } : undefined) },
                };
            });
        });
    });
});
