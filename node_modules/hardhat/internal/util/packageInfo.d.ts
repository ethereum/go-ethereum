export declare function getPackageJsonPath(): string;
export declare function getPackageRoot(): string;
export interface PackageJson {
    name: string;
    version: string;
    type?: "commonjs" | "module";
    engines: {
        node: string;
    };
}
export declare function findClosestPackageJson(file: string): string | null;
export declare function getPackageName(file: string): Promise<string>;
export declare function getPackageJson(): Promise<PackageJson>;
export declare function getHardhatVersion(): string;
/**
 * Return the contents of the package.json in the user's project
 */
export declare function getProjectPackageJson(): Promise<any>;
//# sourceMappingURL=packageInfo.d.ts.map