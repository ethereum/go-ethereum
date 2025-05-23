/* eslint-disable @typescript-eslint/no-var-requires */
import { ArgumentConfig, ProcessExitCodeFunction } from './contracts';
import {
    IMocked,
    Mock,
    setupFunction,
    replacePropertiesBeforeEach,
    addMatchers,
    registerMock,
    reset,
    any,
} from '@morgan-stanley/ts-mocking-bird';
import { parse } from './parse';
import * as fsImport from 'fs';
import * as pathImport from 'path';
import * as helpersImport from './helpers';

jest.mock('fs', () => require('@morgan-stanley/ts-mocking-bird').proxyJestModule(require.resolve('fs')));
jest.mock('path', () => require('@morgan-stanley/ts-mocking-bird').proxyJestModule(require.resolve('path')));
jest.mock('./helpers', () => require('@morgan-stanley/ts-mocking-bird').proxyJestModule(require.resolve('./helpers')));

describe('parse', () => {
    let mockConsole: IMocked<typeof console>;
    let mockProcess: IMocked<typeof process>;
    let mockFs: IMocked<typeof fsImport>;
    let mockPath: IMocked<typeof pathImport>;
    let mockHelper: IMocked<typeof helpersImport>;

    interface ComplexProperties {
        requiredString: string;
        defaultedString: string;
        optionalString?: string;
        requiredBoolean: boolean;
        optionalBoolean?: boolean;
        requiredArray: string[];
        optionalArray?: string[];
    }

    interface PropertiesWithFileConfig extends ComplexProperties {
        optionalFileArg?: string;
        optionalPathArg?: string;
    }

    interface PropertiesWithHelp extends ComplexProperties {
        optionalHelpArg?: boolean;
    }

    function getConfig(): ArgumentConfig<ComplexProperties> {
        return {
            requiredString: String,
            defaultedString: { type: String, defaultValue: defaultFromOption },
            optionalString: { type: String, optional: true },
            requiredBoolean: { type: Boolean, alias: 'b' },
            optionalBoolean: { type: Boolean, optional: true },
            requiredArray: { type: String, alias: 'o', multiple: true },
            optionalArray: { type: String, lazyMultiple: true, optional: true },
        };
    }

    function getHelpConfig(): ArgumentConfig<PropertiesWithHelp> {
        return {
            ...getConfig(),
            optionalHelpArg: { type: Boolean, optional: true, alias: 'h', description: 'This help guide' },
        };
    }

    interface IOptionalArgs {
        path?: string;
        optionalHelpArg?: boolean;
    }

    function getAllOptionalHelpConfig(): ArgumentConfig<IOptionalArgs> {
        return {
            path: { type: String, optional: true },
            optionalHelpArg: { type: Boolean, optional: true, alias: 'h', description: 'This help guide' },
        };
    }

    function getFileConfig(): ArgumentConfig<PropertiesWithFileConfig> {
        return {
            ...getConfig(),
            optionalFileArg: { type: String, optional: true },
            optionalPathArg: { type: String, optional: true },
        };
    }

    const requiredStringValue = 'requiredStringValue';
    const requiredString = ['--requiredString', requiredStringValue];
    const defaultedStringValue = 'defaultedStringValue';
    const defaultFromOption = 'defaultFromOption';
    const defaultedString = ['--defaultedString', defaultedStringValue];
    const optionalStringValue = 'optionalStringValue';
    const optionalString = ['--optionalString', optionalStringValue];
    const requiredBoolean = ['--requiredBoolean'];
    const optionalBoolean = ['--optionalBoolean'];
    const requiredArrayValue = ['requiredArray'];
    const requiredArray = ['--requiredArray', ...requiredArrayValue];
    const optionalArrayValue = ['optionalArrayValueOne', 'optionalArrayValueTwo'];
    const optionalArray = ['--optionalArray', optionalArrayValue[0], '--optionalArray', optionalArrayValue[1]];
    const optionalHelpArg = ['--optionalHelpArg'];
    const optionalFileArg = ['--optionalFileArg=configFilePath'];
    const optionalPathArg = ['--optionalPathArg=configPath'];
    const unknownStringValue = 'unknownStringValue';
    const unknownString = ['--unknownOption', unknownStringValue];

    let jsonFromFile: Record<string, unknown>;

    replacePropertiesBeforeEach(() => {
        jsonFromFile = {
            requiredString: 'requiredStringFromFile',
            defaultedString: 'defaultedStringFromFile',
        };
        const configFromFile = Mock.create<Buffer>().setup(
            setupFunction('toString', () => JSON.stringify(jsonFromFile)),
        ).mock;
        addMatchers();
        mockConsole = Mock.create<typeof console>().setup(setupFunction('error'), setupFunction('log'));
        mockProcess = Mock.create<typeof process>().setup(setupFunction('exit'));
        mockFs = Mock.create<typeof fsImport>().setup(setupFunction('readFileSync', () => configFromFile as any));
        mockPath = Mock.create<typeof pathImport>().setup(setupFunction('resolve', (path) => `${path}_resolved`));
        mockHelper = Mock.create<typeof helpersImport>().setup(setupFunction('mergeConfig'));

        registerMock(fsImport, mockFs.mock);
        registerMock(pathImport, mockPath.mock);
        registerMock(helpersImport, mockHelper.mock);

        return [{ package: process, mocks: mockProcess.mock }];
    });

    afterEach(() => {
        reset(fsImport);
        reset(pathImport);
        reset(helpersImport);
    });

    describe('should create the expected argument value object', () => {
        it('when all options are populated', () => {
            const result = parse(getConfig(), {
                logger: mockConsole.mock,
                argv: [
                    ...requiredString,
                    ...defaultedString,
                    ...optionalString,
                    ...requiredBoolean,
                    ...optionalBoolean,
                    ...requiredArray,
                    ...optionalArray,
                ],
            });

            expect(result).toEqual({
                requiredString: requiredStringValue,
                defaultedString: defaultedStringValue,
                optionalString: optionalStringValue,
                requiredArray: requiredArrayValue,
                optionalArray: optionalArrayValue,
                requiredBoolean: true,
                optionalBoolean: true,
            });
        });

        it('when optional values are ommitted', () => {
            const result = parse(getHelpConfig(), {
                logger: mockConsole.mock,
                argv: [...requiredString, ...requiredArray],
                helpArg: 'optionalHelpArg',
            });

            expect(result).toEqual({
                requiredString: requiredStringValue,
                defaultedString: defaultFromOption,
                requiredArray: requiredArrayValue,
                requiredBoolean: false,
            });

            expect(mockConsole.withFunction('log')).wasNotCalled();
            expect(mockConsole.withFunction('error')).wasNotCalled();
        });

        it('should not load config from file when not specified', () => {
            const result = parse(getFileConfig(), {
                logger: mockConsole.mock,
                argv: [...requiredString, ...requiredArray],
                loadFromFileArg: 'optionalFileArg',
            });

            expect(result).toEqual({
                requiredString: requiredStringValue,
                defaultedString: defaultFromOption,
                requiredArray: requiredArrayValue,
                requiredBoolean: false,
            });

            expect(mockPath.withFunction('resolve')).wasNotCalled();
            expect(mockFs.withFunction('readFileSync')).wasNotCalled();
        });

        it('should load config from file when specified', () => {
            const mergedConfig = {
                requiredString: 'requiredStringFromFile',
                defaultedString: 'defaultedStringFromFile',
                requiredArray: requiredArrayValue,
                requiredBoolean: false,
            };
            mockHelper.setupFunction('mergeConfig', () => mergedConfig as any);

            const argv = [...requiredString, ...requiredArray, ...optionalFileArg, ...optionalPathArg];

            const result = parse(getFileConfig(), {
                logger: mockConsole.mock,
                argv,
                loadFromFileArg: 'optionalFileArg',
                loadFromFileJsonPathArg: 'optionalPathArg',
            });

            expect(result).toEqual(mergedConfig);

            const expectedParsedArgs = {
                defaultedString: 'defaultFromOption',
                requiredString: 'requiredStringValue',
                requiredArray: ['requiredArray'],
                optionalFileArg: 'configFilePath',
                optionalPathArg: 'configPath',
            };

            const expectedParsedArgsWithoutDefaults = {
                requiredString: 'requiredStringValue',
                requiredArray: ['requiredArray'],
                optionalFileArg: 'configFilePath',
                optionalPathArg: 'configPath',
            };

            expect(mockPath.withFunction('resolve').withParameters('configFilePath')).wasCalledOnce();
            expect(mockFs.withFunction('readFileSync').withParameters('configFilePath_resolved')).wasCalledOnce();
            expect(
                mockHelper
                    .withFunction('mergeConfig')
                    .withParametersEqualTo(
                        expectedParsedArgs,
                        expectedParsedArgsWithoutDefaults,
                        jsonFromFile,
                        any(),
                        'optionalPathArg' as any,
                    ),
            ).wasCalledOnce();
        });

        type OveriddeBooleanTest = {
            args: string[];
            configFromFile: Partial<PropertiesWithFileConfig>;
            expected: Partial<PropertiesWithFileConfig>;
        };

        const overrideBooleanTests: OveriddeBooleanTest[] = [
            {
                args: ['--requiredBoolean'],
                configFromFile: { requiredBoolean: false },
                expected: { requiredBoolean: true },
            },
            {
                args: ['--requiredBoolean', '--optionalPathArg=optionalPath'],
                configFromFile: { requiredBoolean: false },
                expected: { requiredBoolean: true, optionalPathArg: 'optionalPath' },
            },
            {
                args: ['--requiredBoolean=false'],
                configFromFile: { requiredBoolean: true },
                expected: { requiredBoolean: false },
            },
            {
                args: ['--requiredBoolean=true'],
                configFromFile: { requiredBoolean: false },
                expected: { requiredBoolean: true },
            },
            {
                args: ['--requiredBoolean', 'false'],
                configFromFile: { requiredBoolean: true },
                expected: { requiredBoolean: false },
            },
            {
                args: ['--requiredBoolean', 'true'],
                configFromFile: { requiredBoolean: false },
                expected: { requiredBoolean: true },
            },
            { args: ['-b'], configFromFile: { requiredBoolean: false }, expected: { requiredBoolean: true } },
            { args: ['-b=false'], configFromFile: { requiredBoolean: true }, expected: { requiredBoolean: false } },
            { args: ['-b=true'], configFromFile: { requiredBoolean: false }, expected: { requiredBoolean: true } },
            { args: ['-b', 'false'], configFromFile: { requiredBoolean: true }, expected: { requiredBoolean: false } },
            { args: ['-b', 'true'], configFromFile: { requiredBoolean: false }, expected: { requiredBoolean: true } },
        ];

        overrideBooleanTests.forEach((test) => {
            it(`should correctly override boolean value in config file when ${test.args.join()} passed on command line`, () => {
                jsonFromFile = test.configFromFile;

                const mergedConfig = {
                    requiredString: 'requiredStringFromFile',
                    defaultedString: 'defaultedStringFromFile',
                    requiredArray: requiredArrayValue,
                    ...test.expected,
                };
                mockHelper.setupFunction('mergeConfig', () => mergedConfig as any);

                const argv = [...optionalFileArg, ...test.args];

                parse(getFileConfig(), {
                    logger: mockConsole.mock,
                    argv,
                    loadFromFileArg: 'optionalFileArg',
                    loadFromFileJsonPathArg: 'optionalPathArg',
                });

                const expectedParsedArgs = {
                    defaultedString: 'defaultFromOption',
                    optionalFileArg: 'configFilePath',
                    ...test.expected,
                };

                const expectedParsedArgsWithoutDefaults = {
                    optionalFileArg: 'configFilePath',
                    ...test.expected,
                };

                expect(
                    mockHelper
                        .withFunction('mergeConfig')
                        .withParametersEqualTo(
                            expectedParsedArgs,
                            expectedParsedArgsWithoutDefaults,
                            jsonFromFile,
                            any(),
                            'optionalPathArg' as any,
                        ),
                ).wasCalledOnce();
            });
        });
    });

    it(`should print errors and exit process when required arguments are missing and no baseCommand or help arg are passed`, () => {
        const result = parse(getConfig(), {
            logger: mockConsole.mock,
            argv: [...defaultedString],
        });

        expect(
            mockConsole
                .withFunction('error')
                .withParameters(
                    `Required parameter 'requiredString' was not passed. Please provide a value by passing '--requiredString=passedValue' in command line arguments`,
                ),
        ).wasCalledOnce();
        expect(
            mockConsole
                .withFunction('error')
                .withParameters(
                    `Required parameter 'requiredArray' was not passed. Please provide a value by passing '--requiredArray=passedValue' or '-o passedValue' in command line arguments`,
                ),
        ).wasCalledOnce();
        expect(mockConsole.withFunction('log')).wasNotCalled();

        expect(mockProcess.withFunction('exit')).wasCalledOnce();

        expect(result).toBeUndefined();
    });

    it(`should print errors and exit process when required arguments are missing and baseCommand is present`, () => {
        const result = parse(getConfig(), {
            logger: mockConsole.mock,
            argv: [...defaultedString],
            baseCommand: 'runMyScript',
        });

        expect(
            mockConsole
                .withFunction('error')
                .withParameters(
                    `Required parameter 'requiredString' was not passed. Please provide a value by running 'runMyScript --requiredString=passedValue'`,
                ),
        ).wasCalledOnce();
        expect(
            mockConsole
                .withFunction('error')
                .withParameters(
                    `Required parameter 'requiredArray' was not passed. Please provide a value by running 'runMyScript --requiredArray=passedValue' or 'runMyScript -o passedValue'`,
                ),
        ).wasCalledOnce();
        expect(mockConsole.withFunction('log')).wasNotCalled();

        expect(mockProcess.withFunction('exit')).wasCalledOnce();

        expect(result).toBeUndefined();
    });

    it(`should print errors and exit process when required arguments are missing and help arg is present`, () => {
        const mockExitFunction = Mock.create<{ processExitCode: ProcessExitCodeFunction<any> }>().setup(
            setupFunction('processExitCode', () => 5),
        );
        const result = parse(getHelpConfig(), {
            logger: mockConsole.mock,
            argv: [...defaultedString],
            helpArg: 'optionalHelpArg',
            processExitCode: mockExitFunction.mock.processExitCode,
        });

        expect(
            mockConsole
                .withFunction('error')
                .withParameters(
                    `Required parameter 'requiredString' was not passed. Please provide a value by passing '--requiredString=passedValue' in command line arguments`,
                ),
        ).wasCalledOnce();
        expect(
            mockConsole
                .withFunction('error')
                .withParameters(
                    `Required parameter 'requiredArray' was not passed. Please provide a value by passing '--requiredArray=passedValue' or '-o passedValue' in command line arguments`,
                ),
        ).wasCalledOnce();
        expect(mockConsole.withFunction('log').withParameters(`To view the help guide pass '-h'`)).wasCalledOnce();
        expect(
            mockExitFunction
                .withFunction('processExitCode')
                .withParametersEqualTo(
                    'missingArgs',
                    { defaultedString: 'defaultedStringValue', requiredBoolean: false },
                    [
                        { name: 'requiredString', type: String },
                        { name: 'requiredArray', type: String, alias: 'o', multiple: true },
                    ] as any,
                ),
        ).wasCalledOnce();
        expect(mockProcess.withFunction('exit').withParameters(5)).wasCalledOnce();

        expect(result).toBeUndefined();
    });

    it(`should print errors and exit process when required arguments are missing and help arg and baseCommand are present`, () => {
        const result = parse(getHelpConfig(), {
            logger: mockConsole.mock,
            argv: [...defaultedString],
            baseCommand: 'runMyScript',
            helpArg: 'optionalHelpArg',
        });

        expect(
            mockConsole
                .withFunction('error')
                .withParameters(
                    `Required parameter 'requiredString' was not passed. Please provide a value by running 'runMyScript --requiredString=passedValue'`,
                ),
        ).wasCalledOnce();
        expect(
            mockConsole
                .withFunction('error')
                .withParameters(
                    `Required parameter 'requiredArray' was not passed. Please provide a value by running 'runMyScript --requiredArray=passedValue' or 'runMyScript -o passedValue'`,
                ),
        ).wasCalledOnce();
        expect(
            mockConsole.withFunction('log').withParameters(`To view the help guide run 'runMyScript -h'`),
        ).wasCalledOnce();

        expect(mockProcess.withFunction('exit').withParameters()).wasCalledOnce();

        expect(result).toBeUndefined();
    });

    it(`should print warnings, return an incomplete result when arguments are missing and exitProcess is false`, () => {
        const result = parse(
            getConfig(),
            {
                logger: mockConsole.mock,
                argv: [...defaultedString],
            },
            false,
        );
        expect(
            mockConsole
                .withFunction('error')
                .withParameters(
                    `Required parameter 'requiredString' was not passed. Please provide a value by passing '--requiredString=passedValue' in command line arguments`,
                ),
        ).wasCalledOnce();
        expect(
            mockConsole
                .withFunction('error')
                .withParameters(
                    `Required parameter 'requiredArray' was not passed. Please provide a value by passing '--requiredArray=passedValue' or '-o passedValue' in command line arguments`,
                ),
        ).wasCalledOnce();

        expect(mockProcess.withFunction('exit')).wasNotCalled();

        expect(result).toEqual({
            defaultedString: defaultedStringValue,
            requiredBoolean: false,
        });
    });

    describe(`should print help messages`, () => {
        it(`and exit when help arg is passed`, () => {
            const mockExitFunction = Mock.create<{ processExitCode: ProcessExitCodeFunction<any> }>().setup(
                setupFunction('processExitCode', () => 5),
            );
            const result = parse(getAllOptionalHelpConfig(), {
                logger: mockConsole.mock,
                argv: [...optionalHelpArg],
                helpArg: 'optionalHelpArg',
                headerContentSections: [
                    { header: 'Header', content: 'Header Content' },
                    { header: 'Header both', content: 'Header Content both', includeIn: 'both' },
                    { header: 'Header cli', content: 'Header Content cli', includeIn: 'cli' },
                    { header: 'Header markdown', content: 'Header markdown markdown', includeIn: 'markdown' },
                ],
                footerContentSections: [
                    { header: 'Footer', content: 'Footer Content' },
                    { header: 'Footer both', content: 'Footer Content both', includeIn: 'both' },
                    { header: 'Footer cli', content: 'Footer Content cli', includeIn: 'cli' },
                    { header: 'Footer markdown', content: 'Footer markdown markdown', includeIn: 'markdown' },
                ],
                processExitCode: mockExitFunction.mock.processExitCode,
            });

            function verifyHelpContent(content: string): string | boolean {
                let currentIndex = 0;

                function verifyNextContent(segment: string) {
                    const segmentIndex = content.indexOf(segment);
                    if (segmentIndex < 0) {
                        return `Expected help content '${segment}' not found`;
                    }
                    if (segmentIndex < currentIndex) {
                        return `Help content '${segment}' not in expected place`;
                    }

                    currentIndex = segmentIndex;
                    return true;
                }

                const helpContentSegments = [
                    'Header',
                    'Header Content',
                    'Header both',
                    'Header Content both',
                    'Header cli',
                    'Header Content cli',
                    '-h',
                    '--optionalHelpArg',
                    'This help guide',
                    'Footer',
                    'Footer Content',
                    'Footer both',
                    'Footer Content both',
                    'Footer cli',
                    'Footer Content cli',
                ];

                const failures = helpContentSegments.map(verifyNextContent).filter((result) => result !== true);

                if (content.indexOf('markdown') >= 0) {
                    failures.push(`'markdown' found in usage guide`);
                }

                if (failures.length > 0) {
                    return failures[0];
                }

                return true;
            }

            expect(result).toBeUndefined();
            expect(
                mockExitFunction
                    .withFunction('processExitCode')
                    .withParametersEqualTo('usageGuide', { optionalHelpArg: true }, []),
            ).wasCalledOnce();
            expect(mockProcess.withFunction('exit').withParameters(5)).wasCalledOnce();
            expect(mockConsole.withFunction('error')).wasNotCalled();
            expect(mockConsole.withFunction('log').withParameters(verifyHelpContent)).wasCalledOnce();
        });
    });

    it(`it should print help messages and return partial arguments when help arg passed and exitProcess is false`, () => {
        const result = parse(
            getHelpConfig(),
            {
                logger: mockConsole.mock,
                argv: [...defaultedString, ...optionalHelpArg],
                helpArg: 'optionalHelpArg',
                headerContentSections: [{ header: 'Header', content: 'Header Content' }],
                footerContentSections: [{ header: 'Footer', content: 'Footer Content' }],
            },
            false,
        );

        expect(result).toEqual({
            defaultedString: defaultedStringValue,
            optionalHelpArg: true,
            requiredBoolean: false,
        });
        expect(mockProcess.withFunction('exit')).wasNotCalled();
        expect(mockConsole.withFunction('log')).wasCalledOnce();
    });

    it(`it should not print help messages and return unknown arguments when stopAtFirstUnknown is true`, () => {
        const result = parse(getConfig(), {
            logger: mockConsole.mock,
            argv: [...requiredString, ...requiredArray, ...unknownString],
            stopAtFirstUnknown: true,
        });

        expect(result).toEqual({
            requiredString: requiredStringValue,
            defaultedString: defaultFromOption,
            requiredBoolean: false,
            requiredArray: requiredArrayValue,
            _unknown: [...unknownString],
        });
        expect(mockProcess.withFunction('exit')).wasNotCalled();
        expect(mockConsole.withFunction('log')).wasNotCalled();
    });

    it(`it should not print help messages and return unknown arguments when stopAtFirstUnknown is true and using option groups`, () => {
        interface GroupedArguments {
            group1Arg: string;
            group2Arg: string;
        }

        const result = parse<GroupedArguments>(
            {
                group1Arg: { type: String, group: 'Group1' },
                group2Arg: { type: String, group: 'Group2' },
            },
            {
                logger: mockConsole.mock,
                argv: ['--group1Arg', 'group1', '--group2Arg', 'group2', ...unknownString],
                stopAtFirstUnknown: true,
            },
        );

        expect(result).toEqual({
            group1Arg: 'group1',
            group2Arg: 'group2',
            _unknown: [...unknownString],
        });
        expect(mockProcess.withFunction('exit')).wasNotCalled();
        expect(mockConsole.withFunction('log')).wasNotCalled();
    });
});
