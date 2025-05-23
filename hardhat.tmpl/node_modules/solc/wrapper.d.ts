declare function wrapper(soljson: any): {
    version: any;
    semver: any;
    license: any;
    lowlevel: {
        compileSingle: any;
        compileMulti: any;
        compileCallback: (input: any, optimize: any, readCallback: any) => any;
        compileStandard: any;
    };
    features: {
        legacySingleInput: boolean;
        multipleInputs: boolean;
        importCallback: boolean;
        nativeStandardJSON: boolean;
    };
    compile: any;
    loadRemoteVersion: typeof loadRemoteVersion;
    setupMethods: typeof wrapper;
};
declare function loadRemoteVersion(versionString: any, callback: any): void;
export = wrapper;
