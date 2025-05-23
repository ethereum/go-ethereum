export interface TypechainConfig {
    outDir: string;
    target: string;
    alwaysGenerateOverloads: boolean;
    discriminateTypes: boolean;
    tsNocheck: boolean;
    externalArtifacts?: string[];
    dontOverrideCompile: boolean;
    node16Modules: boolean;
}
export interface TypechainUserConfig extends Partial<TypechainConfig> {
}
