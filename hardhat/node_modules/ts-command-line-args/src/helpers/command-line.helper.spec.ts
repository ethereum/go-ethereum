import { createCommandLineConfig, mergeConfig, normaliseConfig } from './command-line.helper';
import { ArgumentConfig, ArgumentOptions } from '../contracts';

describe('command-line.helper', () => {
    interface ComplexProperties {
        requiredStringOne: string;
        requiredStringTwo: string;
        optionalString?: string;
        requiredArray: string[];
        optionalArray?: string[];
    }

    function getConfig(): ArgumentConfig<ComplexProperties> {
        return {
            requiredStringOne: String,
            requiredStringTwo: { type: String },
            optionalString: { type: String, optional: true },
            requiredArray: { type: String, multiple: true },
            optionalArray: { type: String, lazyMultiple: true, optional: true },
        };
    }

    describe('normaliseConfig', () => {
        it('should replace type constructors with objects', () => {
            const normalised = normaliseConfig(getConfig());

            expect(normalised).toEqual({
                requiredStringOne: { type: String },
                requiredStringTwo: { type: String },
                optionalString: { type: String, optional: true },
                requiredArray: { type: String, multiple: true },
                optionalArray: { type: String, lazyMultiple: true, optional: true },
            });
        });
    });

    describe('createCommandLineConfig', () => {
        it('should create expected config', () => {
            const commandLineConfig = createCommandLineConfig(normaliseConfig(getConfig()));

            expect(commandLineConfig).toEqual([
                { name: 'requiredStringOne', type: String },
                { name: 'requiredStringTwo', type: String },
                { name: 'optionalString', type: String, optional: true },
                { name: 'requiredArray', type: String, multiple: true },
                { name: 'optionalArray', type: String, lazyMultiple: true, optional: true },
            ]);
        });
    });

    describe('mergeConfig', () => {
        interface ISampleInterface {
            stringOne: string;
            stringTwo: string;
            strings: string[];
            number: number;
            boolean: boolean;
            dates: Date[];
            optionalObject?: { value: string };
            configPath?: string;
        }

        let options: ArgumentOptions<ISampleInterface>;

        beforeEach(() => {
            options = {
                stringOne: { type: String },
                stringTwo: { type: String },
                strings: { type: String, multiple: true },
                number: { type: Number },
                boolean: { type: Boolean },
                dates: { type: (value) => new Date(Date.parse(value)), multiple: true },
                optionalObject: { type: (value) => (typeof value === 'string' ? { value } : value), optional: true },
                configPath: { type: String, optional: true },
            };
        });

        type FileConfigTest = {
            description: string;
            parsedArgs: Partial<Record<keyof ISampleInterface, any>>;
            parsedArgsNoDefaults?: Partial<Record<keyof ISampleInterface, any>>;
            fileContent: Record<string, unknown>;
            expected: Partial<ISampleInterface>;
            jsonPath?: keyof ISampleInterface;
        };
        const fileConfigTests: FileConfigTest[] = [
            {
                description: 'no arguments passed',
                parsedArgs: {},
                fileContent: {
                    stringOne: 'stringOneFromFile',
                    stringTwo: 'stringTwoFromFile',
                },
                expected: {
                    stringOne: 'stringOneFromFile',
                    stringTwo: 'stringTwoFromFile',
                },
            },
            {
                description: 'file content is empty',
                parsedArgs: {
                    stringOne: 'stringOneFromArgs',
                    stringTwo: 'stringTwoFromArgs',
                    number: 36,
                    boolean: false,
                    dates: [new Date()],
                },
                fileContent: {},
                expected: {
                    stringOne: 'stringOneFromArgs',
                    stringTwo: 'stringTwoFromArgs',
                    number: 36,
                    boolean: false,
                    dates: [new Date()],
                },
            },
            {
                description: 'both file content and parsed args have values',
                parsedArgs: {
                    stringOne: 'stringOneFromArgs',
                    boolean: false,
                },
                fileContent: {
                    stringTwo: 'stringTwoFromFile',
                    number: 55,
                },
                expected: {
                    stringOne: 'stringOneFromArgs',
                    boolean: false,
                    stringTwo: 'stringTwoFromFile',
                    number: 55,
                },
            },
            {
                description: 'file content and parsed args have conflicting values',
                parsedArgs: {
                    stringOne: 'stringOneFromArgs',
                    number: 55,
                    boolean: false,
                    dates: [new Date(2020, 5, 1)],
                },
                fileContent: {
                    stringOne: 'stringOneFromFile',
                    stringTwo: 'stringTwoFromFile',
                    number: 36,
                    boolean: true,
                    dates: 'March 1 2020',
                    randomOtherProp: '',
                },
                expected: {
                    stringOne: 'stringOneFromArgs',
                    stringTwo: 'stringTwoFromFile',
                    number: 55,
                    boolean: false,
                    dates: [new Date(2020, 5, 1)],
                },
            },
            {
                description: 'config file overrides default',
                parsedArgs: { optionalObject: { value: 'parsedValue' } },
                parsedArgsNoDefaults: {},
                fileContent: {
                    optionalObject: { value: 'valueFromFile' },
                },
                expected: {
                    optionalObject: { value: 'valueFromFile' },
                },
            },
            {
                description: 'parsed args overrides config file and default',
                parsedArgs: { optionalObject: { value: 'parsedValue' } },
                parsedArgsNoDefaults: { optionalObject: { value: 'parsedValue' } },
                fileContent: {
                    optionalObject: { value: 'valueFromFile' },
                },
                expected: {
                    optionalObject: { value: 'parsedValue' },
                },
            },
            {
                description: 'jsonPath set',
                parsedArgs: {
                    configPath: 'configs.cmdLineConfig.example',
                },
                fileContent: {
                    configs: {
                        cmdLineConfig: {
                            example: {
                                stringOne: 'stringOneFromFile',
                                stringTwo: 'stringTwoFromFile',
                            },
                        },
                    },
                },
                expected: {
                    stringOne: 'stringOneFromFile',
                    stringTwo: 'stringTwoFromFile',
                    configPath: 'configs.cmdLineConfig.example',
                },
                jsonPath: 'configPath',
            },
        ];

        fileConfigTests.forEach((test) => {
            it(`should return configFromFile when ${test.description}`, () => {
                const result = mergeConfig<ISampleInterface>(
                    test.parsedArgs,
                    test.parsedArgsNoDefaults || test.parsedArgs,
                    test.fileContent,
                    options,
                    test.jsonPath,
                );

                expect(result).toEqual(test.expected);
            });
        });

        type ConversionTest = {
            fromFile: Partial<Record<keyof ISampleInterface, any>>;
            expected: Partial<ISampleInterface>;
        };

        const typeConversionTests: ConversionTest[] = [
            { fromFile: { stringOne: 'stringOne' }, expected: { stringOne: 'stringOne' } },
            { fromFile: { strings: 'stringOne' }, expected: { strings: ['stringOne'] } },
            { fromFile: { strings: ['stringOne', 'stringTwo'] }, expected: { strings: ['stringOne', 'stringTwo'] } },
            { fromFile: { number: '1' }, expected: { number: 1 } },
            { fromFile: { number: 1 }, expected: { number: 1 } },
            { fromFile: { number: 'one' }, expected: { number: NaN } },
            { fromFile: { boolean: true }, expected: { boolean: true } },
            { fromFile: { boolean: false }, expected: { boolean: false } },
            { fromFile: { boolean: 1 }, expected: { boolean: true } },
            { fromFile: { boolean: 0 }, expected: { boolean: false } },
            { fromFile: { boolean: 'true' }, expected: { boolean: true } },
            { fromFile: { boolean: 'false' }, expected: { boolean: false } },
            { fromFile: { dates: '2020/03/04' }, expected: { dates: [new Date(2020, 2, 4)] } },
            {
                fromFile: { dates: ['2020/03/04', '2020/05/06'] },
                expected: { dates: [new Date(2020, 2, 4), new Date(2020, 4, 6)] },
            },
        ];

        typeConversionTests.forEach((test) => {
            it(`should convert all configfromFile properties with type conversion function with input: '${JSON.stringify(
                test.fromFile,
            )}'`, () => {
                const result = mergeConfig<ISampleInterface>({}, {}, test.fromFile, options, undefined);

                expect(result).toEqual(test.expected);
            });
        });
    });
});
