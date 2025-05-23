import wrapper from './wrapper';
declare const _default: {
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
    loadRemoteVersion: (versionString: any, callback: any) => void;
    setupMethods: typeof wrapper;
};
export = _default;
