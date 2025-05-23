import { EthExecutionAPI, Web3APISpec, Web3BaseProvider } from 'web3-types';
import { EventEmitter } from 'eventemitter3';
/**
 * This is an abstract class, which extends {@link Web3BaseProvider} class. This class is used to implement a provider that adheres to the EIP-1193 standard for Ethereum providers.
 */
export declare abstract class Eip1193Provider<API extends Web3APISpec = EthExecutionAPI> extends Web3BaseProvider<API> {
    protected readonly _eventEmitter: EventEmitter;
    private _chainId;
    private _accounts;
    private _getChainId;
    private _getAccounts;
    protected _onConnect(): void;
    protected _onDisconnect(code: number, data?: unknown): void;
    private _onAccountsChanged;
}
