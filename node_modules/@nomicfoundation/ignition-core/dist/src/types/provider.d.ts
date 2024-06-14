/**
 * Arguments for a request to an EIP-1193 Provider.
 *
 * @beta
 */
export interface RequestArguments {
    readonly method: string;
    readonly params?: readonly unknown[] | object;
}
/**
 * A provider for on-chain interactions.
 *
 * @beta
 */
export interface EIP1193Provider {
    request(args: RequestArguments): Promise<unknown>;
}
//# sourceMappingURL=provider.d.ts.map