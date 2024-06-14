/**
 * The `node:test` module provides a standalone testing module.
 * @see [source](https://github.com/nodejs/node/blob/v18.x/lib/test.js)
 */
declare module 'node:test' {
    /**
     * Programmatically start the test runner.
     * @since v18.9.0
     * @param options Configuration options for running tests.
     * @returns A {@link TestsStream} that emits events about the test execution.
     */
    function run(options?: RunOptions): TestsStream;

    /**
     * The `test()` function is the value imported from the test module. Each invocation of this
     * function results in reporting the test to the {@link TestsStream}.
     *
     * The {@link TestContext} object passed to the fn argument can be used to perform actions
     * related to the current test. Examples include skipping the test, adding additional
     * diagnostic information, or creating subtests.
     *
     * `test()` returns a {@link Promise} that resolves once the test completes. The return value
     * can usually be discarded for top level tests. However, the return value from subtests should
     * be used to prevent the parent test from finishing first and cancelling the subtest as shown
     * in the following example.
     *
     * ```js
     * test('top level test', async (t) => {
     *   // The setTimeout() in the following subtest would cause it to outlive its
     *   // parent test if 'await' is removed on the next line. Once the parent test
     *   // completes, it will cancel any outstanding subtests.
     *   await t.test('longer running subtest', async (t) => {
     *     return new Promise((resolve, reject) => {
     *       setTimeout(resolve, 1000);
     *     });
     *   });
     * });
     * ```
     * @since v18.0.0
     * @param name The name of the test, which is displayed when reporting test results.
     *    Default: The `name` property of fn, or `'<anonymous>'` if `fn` does not have a name.
     * @param options Configuration options for the test
     * @param fn The function under test. The first argument to this function is a
     *    {@link TestContext} object. If the test uses callbacks, the callback function is
     *    passed as the second argument. Default: A no-op function.
     * @returns A {@link Promise} resolved with `undefined` once the test completes.
     */
    function test(name?: string, fn?: TestFn): Promise<void>;
    function test(name?: string, options?: TestOptions, fn?: TestFn): Promise<void>;
    function test(options?: TestOptions, fn?: TestFn): Promise<void>;
    function test(fn?: TestFn): Promise<void>;

    /**
     * @since v18.6.0
     * @param name The name of the suite, which is displayed when reporting suite results.
     *    Default: The `name` property of fn, or `'<anonymous>'` if `fn` does not have a name.
     * @param options Configuration options for the suite
     * @param fn The function under suite. Default: A no-op function.
     */
    function describe(name?: string, options?: TestOptions, fn?: SuiteFn): void;
    function describe(name?: string, fn?: SuiteFn): void;
    function describe(options?: TestOptions, fn?: SuiteFn): void;
    function describe(fn?: SuiteFn): void;
    namespace describe {
        // Shorthand for skipping a suite, same as `describe([name], { skip: true }[, fn])`.
        function skip(name?: string, options?: TestOptions, fn?: SuiteFn): void;
        function skip(name?: string, fn?: SuiteFn): void;
        function skip(options?: TestOptions, fn?: SuiteFn): void;
        function skip(fn?: SuiteFn): void;

        // Shorthand for marking a suite as `TODO`, same as `describe([name], { todo: true }[, fn])`.
        function todo(name?: string, options?: TestOptions, fn?: SuiteFn): void;
        function todo(name?: string, fn?: SuiteFn): void;
        function todo(options?: TestOptions, fn?: SuiteFn): void;
        function todo(fn?: SuiteFn): void;
    }

    /**
     * @since v18.6.0
     * @param name The name of the test, which is displayed when reporting test results.
     *    Default: The `name` property of fn, or `'<anonymous>'` if `fn` does not have a name.
     * @param options Configuration options for the test
     * @param fn The function under test. If the test uses callbacks, the callback function is
     *    passed as the second argument. Default: A no-op function.
     */
    function it(name?: string, options?: TestOptions, fn?: ItFn): void;
    function it(name?: string, fn?: ItFn): void;
    function it(options?: TestOptions, fn?: ItFn): void;
    function it(fn?: ItFn): void;
    namespace it {
        // Shorthand for skipping a test, same as `it([name], { skip: true }[, fn])`.
        function skip(name?: string, options?: TestOptions, fn?: ItFn): void;
        function skip(name?: string, fn?: ItFn): void;
        function skip(options?: TestOptions, fn?: ItFn): void;
        function skip(fn?: ItFn): void;

        // Shorthand for marking a test as `TODO`, same as `it([name], { todo: true }[, fn])`.
        function todo(name?: string, options?: TestOptions, fn?: ItFn): void;
        function todo(name?: string, fn?: ItFn): void;
        function todo(options?: TestOptions, fn?: ItFn): void;
        function todo(fn?: ItFn): void;
    }

    /**
     * The type of a function under test. The first argument to this function is a
     * {@link TestContext} object. If the test uses callbacks, the callback function is passed as
     * the second argument.
     */
    type TestFn = (t: TestContext, done: (result?: any) => void) => any;

    /**
     * The type of a function under Suite.
     * If the test uses callbacks, the callback function is passed as an argument
     */
    type SuiteFn = (done: (result?: any) => void) => void;

    /**
     * The type of a function under test.
     * If the test uses callbacks, the callback function is passed as an argument
     */
    type ItFn = (done: (result?: any) => void) => any;

    interface RunOptions {
        /**
         * If a number is provided, then that many files would run in parallel.
         * If truthy, it would run (number of cpu cores - 1) files in parallel.
         * If falsy, it would only run one file at a time.
         * If unspecified, subtests inherit this value from their parent.
         * @default true
         */
        concurrency?: number | boolean | undefined;

        /**
         * An array containing the list of files to run.
         * If unspecified, the test runner execution model will be used.
         */
        files?: readonly string[] | undefined;

        /**
         * Allows aborting an in-progress test execution.
         * @default undefined
         */
        signal?: AbortSignal | undefined;

        /**
         * A number of milliseconds the test will fail after.
         * If unspecified, subtests inherit this value from their parent.
         * @default Infinity
         */
        timeout?: number | undefined;

        /**
         * Sets inspector port of test child process.
         * If a nullish value is provided, each process gets its own port,
         * incremented from the primary's `process.debugPort`.
         */
        inspectPort?: number | (() => number) | undefined;
    }

    /**
     * A successful call of the `run()` method will return a new `TestsStream` object,
     * streaming a series of events representing the execution of the tests.
     * `TestsStream` will emit events in the order of the tests' definitions.
     * @since v18.9.0
     */
    interface TestsStream extends NodeJS.ReadableStream {
        addListener(event: 'test:diagnostic', listener: (data: DiagnosticData) => void): this;
        addListener(event: 'test:fail', listener: (data: TestFail) => void): this;
        addListener(event: 'test:pass', listener: (data: TestPass) => void): this;
        addListener(event: 'test:plan', listener: (data: TestPlan) => void): this;
        addListener(event: 'test:start', listener: (data: TestStart) => void): this;
        addListener(event: string, listener: (...args: any[]) => void): this;
        emit(event: 'test:diagnostic', data: DiagnosticData): boolean;
        emit(event: 'test:fail', data: TestFail): boolean;
        emit(event: 'test:pass', data: TestPass): boolean;
        emit(event: 'test:plan', data: TestPlan): boolean;
        emit(event: 'test:start', data: TestStart): boolean;
        emit(event: string | symbol, ...args: any[]): boolean;
        on(event: 'test:diagnostic', listener: (data: DiagnosticData) => void): this;
        on(event: 'test:fail', listener: (data: TestFail) => void): this;
        on(event: 'test:pass', listener: (data: TestPass) => void): this;
        on(event: 'test:plan', listener: (data: TestPlan) => void): this;
        on(event: 'test:start', listener: (data: TestStart) => void): this;
        on(event: string, listener: (...args: any[]) => void): this;
        once(event: 'test:diagnostic', listener: (data: DiagnosticData) => void): this;
        once(event: 'test:fail', listener: (data: TestFail) => void): this;
        once(event: 'test:pass', listener: (data: TestPass) => void): this;
        once(event: 'test:plan', listener: (data: TestPlan) => void): this;
        once(event: 'test:start', listener: (data: TestStart) => void): this;
        once(event: string, listener: (...args: any[]) => void): this;
        prependListener(event: 'test:diagnostic', listener: (data: DiagnosticData) => void): this;
        prependListener(event: 'test:fail', listener: (data: TestFail) => void): this;
        prependListener(event: 'test:pass', listener: (data: TestPass) => void): this;
        prependListener(event: 'test:plan', listener: (data: TestPlan) => void): this;
        prependListener(event: 'test:start', listener: (data: TestStart) => void): this;
        prependListener(event: string, listener: (...args: any[]) => void): this;
        prependOnceListener(event: 'test:diagnostic', listener: (data: DiagnosticData) => void): this;
        prependOnceListener(event: 'test:fail', listener: (data: TestFail) => void): this;
        prependOnceListener(event: 'test:pass', listener: (data: TestPass) => void): this;
        prependOnceListener(event: 'test:plan', listener: (data: TestPlan) => void): this;
        prependOnceListener(event: 'test:start', listener: (data: TestStart) => void): this;
        prependOnceListener(event: string, listener: (...args: any[]) => void): this;
    }

    interface DiagnosticData {
        /**
         * The diagnostic message.
         */
        message: string;

        /**
         * The nesting level of the test.
         */
        nesting: number;
    }

    interface TestFail {
        /**
         * Additional execution metadata.
         */
        details: {
            /**
             * The duration of the test in milliseconds.
             */
            duration: number;

            /**
             * The error thrown by the test.
             */
            error: Error;
        };

        /**
         * The test name.
         */
        name: string;

        /**
         * The nesting level of the test.
         */
        nesting: number;

        /**
         * The ordinal number of the test.
         */
        testNumber: number;

        /**
         * Present if `context.todo` is called.
         */
        todo?: string | boolean;

        /**
         * Present if `context.skip` is called.
         */
        skip?: string | boolean;
    }

    interface TestPass {
        /**
         * Additional execution metadata.
         */
        details: {
            /**
             * The duration of the test in milliseconds.
             */
            duration: number;
        };

        /**
         * The test name.
         */
        name: string;

        /**
         * The nesting level of the test.
         */
        nesting: number;

        /**
         * The ordinal number of the test.
         */
        testNumber: number;

        /**
         * Present if `context.todo` is called.
         */
        todo?: string | boolean;

        /**
         * Present if `context.skip` is called.
         */
        skip?: string | boolean;
    }

    interface TestPlan {
        /**
         * The nesting level of the test.
         */
        nesting: number;

        /**
         * The number of subtests that have ran.
         */
        count: number;
    }

    interface TestStart {
        /**
         * The test name.
         */
        name: string;

        /**
         * The nesting level of the test.
         */
        nesting: number;
    }

    /**
     * An instance of `TestContext` is passed to each test function in order to interact with the
     * test runner. However, the `TestContext` constructor is not exposed as part of the API.
     * @since v18.0.0
     */
    interface TestContext {
        /**
         * This function is used to create a hook running before each subtest of the current test.
         * @param fn The hook function. If the hook uses callbacks, the callback function is passed as
         *    the second argument. Default: A no-op function.
         * @param options Configuration options for the hook.
         * @since v18.8.0
         */
        beforeEach: typeof beforeEach;

        /**
         * This function is used to create a hook that runs after the current test finishes.
         * @param fn The hook function. If the hook uses callbacks, the callback function is passed as
         *    the second argument. Default: A no-op function.
         * @param options Configuration options for the hook.
         * @since v18.13.0
         */
        after: typeof after;

        /**
         * This function is used to create a hook running after each subtest of the current test.
         * @param fn The hook function. If the hook uses callbacks, the callback function is passed as
         *    the second argument. Default: A no-op function.
         * @param options Configuration options for the hook.
         * @since v18.8.0
         */
        afterEach: typeof afterEach;

        /**
         * This function is used to write diagnostics to the output. Any diagnostic information is
         * included at the end of the test's results. This function does not return a value.
         * @param message Message to be reported.
         * @since v18.0.0
         */
        diagnostic(message: string): void;

        /**
         * The name of the test.
         * @since v18.8.0
         */
        readonly name: string;

        /**
         * If `shouldRunOnlyTests` is truthy, the test context will only run tests that have the `only`
         * option set. Otherwise, all tests are run. If Node.js was not started with the `--test-only`
         * command-line option, this function is a no-op.
         * @param shouldRunOnlyTests Whether or not to run `only` tests.
         * @since v18.0.0
         */
        runOnly(shouldRunOnlyTests: boolean): void;

        /**
         * Can be used to abort test subtasks when the test has been aborted.
         * @since v18.7.0
         */
        readonly signal: AbortSignal;

        /**
         * This function causes the test's output to indicate the test as skipped. If `message` is
         * provided, it is included in the output. Calling `skip()` does not terminate execution of
         * the test function. This function does not return a value.
         * @param message Optional skip message.
         * @since v18.0.0
         */
        skip(message?: string): void;

        /**
         * This function adds a `TODO` directive to the test's output. If `message` is provided, it is
         * included in the output. Calling `todo()` does not terminate execution of the test
         * function. This function does not return a value.
         * @param message Optional `TODO` message.
         * @since v18.0.0
         */
        todo(message?: string): void;

        /**
         * This function is used to create subtests under the current test. This function behaves in
         * the same fashion as the top level {@link test} function.
         * @since v18.0.0
         * @param name The name of the test, which is displayed when reporting test results.
         *    Default: The `name` property of fn, or `'<anonymous>'` if `fn` does not have a name.
         * @param options Configuration options for the test
         * @param fn The function under test. This first argument to this function is a
         *    {@link TestContext} object. If the test uses callbacks, the callback function is
         *    passed as the second argument. Default: A no-op function.
         * @returns A {@link Promise} resolved with `undefined` once the test completes.
         */
        test: typeof test;
        /**
         * Each test provides its own MockTracker instance.
         */
        readonly mock: MockTracker;
    }

    interface TestOptions {
        /**
         * If a number is provided, then that many tests would run in parallel.
         * If truthy, it would run (number of cpu cores - 1) tests in parallel.
         * For subtests, it will be `Infinity` tests in parallel.
         * If falsy, it would only run one test at a time.
         * If unspecified, subtests inherit this value from their parent.
         * @default false
         */
        concurrency?: number | boolean | undefined;

        /**
         * If truthy, and the test context is configured to run `only` tests, then this test will be
         * run. Otherwise, the test is skipped.
         * @default false
         */
        only?: boolean | undefined;

        /**
         * Allows aborting an in-progress test.
         * @since v18.8.0
         */
        signal?: AbortSignal | undefined;

        /**
         * If truthy, the test is skipped. If a string is provided, that string is displayed in the
         * test results as the reason for skipping the test.
         * @default false
         */
        skip?: boolean | string | undefined;

        /**
         * A number of milliseconds the test will fail after. If unspecified, subtests inherit this
         * value from their parent.
         * @default Infinity
         * @since v18.7.0
         */
        timeout?: number | undefined;

        /**
         * If truthy, the test marked as `TODO`. If a string is provided, that string is displayed in
         * the test results as the reason why the test is `TODO`.
         * @default false
         */
        todo?: boolean | string | undefined;
    }

    /**
     * This function is used to create a hook running before running a suite.
     * @param fn The hook function. If the hook uses callbacks, the callback function is passed as
     *    the second argument. Default: A no-op function.
     * @param options Configuration options for the hook.
     * @since v18.8.0
     */
    function before(fn?: HookFn, options?: HookOptions): void;

    /**
     * This function is used to create a hook running after running a suite.
     * @param fn The hook function. If the hook uses callbacks, the callback function is passed as
     *    the second argument. Default: A no-op function.
     * @param options Configuration options for the hook.
     * @since v18.8.0
     */
    function after(fn?: HookFn, options?: HookOptions): void;

    /**
     * This function is used to create a hook running before each subtest of the current suite.
     * @param fn The hook function. If the hook uses callbacks, the callback function is passed as
     *    the second argument. Default: A no-op function.
     * @param options Configuration options for the hook.
     * @since v18.8.0
     */
    function beforeEach(fn?: HookFn, options?: HookOptions): void;

    /**
     * This function is used to create a hook running after each subtest of the current test.
     * @param fn The hook function. If the hook uses callbacks, the callback function is passed as
     *    the second argument. Default: A no-op function.
     * @param options Configuration options for the hook.
     * @since v18.8.0
     */
    function afterEach(fn?: HookFn, options?: HookOptions): void;

    /**
     * The hook function. If the hook uses callbacks, the callback function is passed as the
     * second argument.
     */
    type HookFn = (done: (result?: any) => void) => any;

    /**
     * Configuration options for hooks.
     * @since v18.8.0
     */
    interface HookOptions {
        /**
         * Allows aborting an in-progress hook.
         */
        signal?: AbortSignal | undefined;

        /**
         * A number of milliseconds the hook will fail after. If unspecified, subtests inherit this
         * value from their parent.
         * @default Infinity
         */
        timeout?: number | undefined;
    }

    interface MockFunctionOptions {
        /**
         * The number of times that the mock will use the behavior of `implementation`.
         * Once the mock function has been called `times` times,
         * it will automatically restore the behavior of `original`.
         * This value must be an integer greater than zero.
         * @default Infinity
         */
        times?: number | undefined;
    }

    interface MockMethodOptions extends MockFunctionOptions {
        /**
         * If `true`, `object[methodName]` is treated as a getter.
         * This option cannot be used with the `setter` option.
         */
        getter?: boolean | undefined;

        /**
         * If `true`, `object[methodName]` is treated as a setter.
         * This option cannot be used with the `getter` option.
         */
        setter?: boolean | undefined;
    }

    type Mock<F extends Function> = F & {
        mock: MockFunctionContext<F>;
    };

    type NoOpFunction = (...args: any[]) => undefined;

    type FunctionPropertyNames<T> = {
        [K in keyof T]: T[K] extends Function ? K : never;
    }[keyof T];

    interface MockTracker {
        /**
         * This function is used to create a mock function.
         * @param original An optional function to create a mock on.
         * @param implementation An optional function used as the mock implementation for `original`.
         *  This is useful for creating mocks that exhibit one behavior for a specified number of calls and then restore the behavior of `original`.
         * @param options Optional configuration options for the mock function.
         */
        fn<F extends Function = NoOpFunction>(original?: F, options?: MockFunctionOptions): Mock<F>;
        fn<F extends Function = NoOpFunction, Implementation extends Function = F>(original?: F, implementation?: Implementation, options?: MockFunctionOptions): Mock<F | Implementation>;

        /**
         * This function is used to create a mock on an existing object method.
         * @param object The object whose method is being mocked.
         * @param methodName The identifier of the method on `object` to mock. If `object[methodName]` is not a function, an error is thrown.
         * @param implementation An optional function used as the mock implementation for `object[methodName]`.
         * @param options Optional configuration options for the mock method.
         */
        method<
            MockedObject extends object,
            MethodName extends FunctionPropertyNames<MockedObject>,
        >(
            object: MockedObject,
            methodName: MethodName,
            options?: MockFunctionOptions,
        ): MockedObject[MethodName] extends Function
            ? Mock<MockedObject[MethodName]>
            : never;
        method<
            MockedObject extends object,
            MethodName extends FunctionPropertyNames<MockedObject>,
            Implementation extends Function,
        >(
            object: MockedObject,
            methodName: MethodName,
            implementation: Implementation,
            options?: MockFunctionOptions,
        ): MockedObject[MethodName] extends Function
            ? Mock<MockedObject[MethodName] | Implementation>
            : never;
        method<MockedObject extends object>(
            object: MockedObject,
            methodName: keyof MockedObject,
            options: MockMethodOptions,
        ): Mock<Function>;
        method<MockedObject extends object>(
            object: MockedObject,
            methodName: keyof MockedObject,
            implementation: Function,
            options: MockMethodOptions,
        ): Mock<Function>;

        /**
         * This function is syntax sugar for {@link MockTracker.method} with `options.getter` set to `true`.
         */
        getter<
            MockedObject extends object,
            MethodName extends keyof MockedObject,
        >(
            object: MockedObject,
            methodName: MethodName,
            options?: MockFunctionOptions,
        ): Mock<() => MockedObject[MethodName]>;
        getter<
            MockedObject extends object,
            MethodName extends keyof MockedObject,
            Implementation extends Function,
        >(
            object: MockedObject,
            methodName: MethodName,
            implementation?: Implementation,
            options?: MockFunctionOptions,
        ): Mock<(() => MockedObject[MethodName]) | Implementation>;

        /**
         * This function is syntax sugar for {@link MockTracker.method} with `options.setter` set to `true`.
         */
        setter<
            MockedObject extends object,
            MethodName extends keyof MockedObject,
        >(
            object: MockedObject,
            methodName: MethodName,
            options?: MockFunctionOptions,
        ): Mock<(value: MockedObject[MethodName]) => void>;
        setter<
            MockedObject extends object,
            MethodName extends keyof MockedObject,
            Implementation extends Function,
        >(
            object: MockedObject,
            methodName: MethodName,
            implementation?: Implementation,
            options?: MockFunctionOptions,
        ): Mock<((value: MockedObject[MethodName]) => void) | Implementation>;

        /**
         * This function restores the default behavior of all mocks that were previously created by this `MockTracker`
         * and disassociates the mocks from the `MockTracker` instance. Once disassociated, the mocks can still be used,
         * but the `MockTracker` instance can no longer be used to reset their behavior or otherwise interact with them.
         *
         * After each test completes, this function is called on the test context's `MockTracker`.
         * If the global `MockTracker` is used extensively, calling this function manually is recommended.
         */
        reset(): void;

        /**
         * This function restores the default behavior of all mocks that were previously created by this `MockTracker`.
         * Unlike `mock.reset()`, `mock.restoreAll()` does not disassociate the mocks from the `MockTracker` instance.
         */
        restoreAll(): void;
    }

    const mock: MockTracker;

    interface MockFunctionCall<
        F extends Function,
        ReturnType = F extends (...args: any) => infer T
            ? T
            : F extends abstract new (...args: any) => infer T
                ? T
                : unknown,
        Args = F extends (...args: infer Y) => any
            ? Y
            : F extends abstract new (...args: infer Y) => any
                ? Y
                : unknown[],
    > {
        /**
         * An array of the arguments passed to the mock function.
         */
        arguments: Args;
        /**
         * If the mocked function threw then this property contains the thrown value.
         */
        error: unknown | undefined;
        /**
         * The value returned by the mocked function.
         *
         * If the mocked function threw, it will be `undefined`.
         */
        result: ReturnType | undefined;
        /**
         * An `Error` object whose stack can be used to determine the callsite of the mocked function invocation.
         */
        stack: Error;
        /**
         * If the mocked function is a constructor, this field contains the class being constructed.
         * Otherwise this will be `undefined`.
         */
        target: F extends abstract new (...args: any) => any ? F : undefined;
        /**
         * The mocked function's `this` value.
         */
        this: unknown;
    }

    interface MockFunctionContext<F extends Function> {
        /**
         * A getter that returns a copy of the internal array used to track calls to the mock.
         */
        readonly calls: Array<MockFunctionCall<F>>;

        /**
         * This function returns the number of times that this mock has been invoked.
         * This function is more efficient than checking `ctx.calls.length`
         * because `ctx.calls` is a getter that creates a copy of the internal call tracking array.
         */
        callCount(): number;

        /**
         * This function is used to change the behavior of an existing mock.
         * @param implementation The function to be used as the mock's new implementation.
         */
        mockImplementation(implementation: Function): void;

        /**
         * This function is used to change the behavior of an existing mock for a single invocation.
         * Once invocation `onCall` has occurred, the mock will revert to whatever behavior
         * it would have used had `mockImplementationOnce()` not been called.
         * @param implementation The function to be used as the mock's implementation for the invocation number specified by `onCall`.
         * @param onCall The invocation number that will use `implementation`.
         *  If the specified invocation has already occurred then an exception is thrown.
         */
        mockImplementationOnce(implementation: Function, onCall?: number): void;

        /**
         * Resets the call history of the mock function.
         */
        resetCalls(): void;

        /**
         * Resets the implementation of the mock function to its original behavior.
         * The mock can still be used after calling this function.
         */
        restore(): void;
    }

    export { test as default, run, test, describe, it, before, after, beforeEach, afterEach, mock };
}
