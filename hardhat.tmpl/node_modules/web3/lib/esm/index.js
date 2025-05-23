/*
This file is part of web3.js.

web3.js is free software: you can redistribute it and/or modify
it under the terms of the GNU Lesser General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

web3.js is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Lesser General Public License for more details.

You should have received a copy of the GNU Lesser General Public License
along with web3.js.  If not, see <http://www.gnu.org/licenses/>.
*/
/**
 * This is the main (or 'umbrella') class of the web3.js library.
 *
 * ```ts
 * import Web3 from 'web3';
 *
 * > Web3.utils
 * > Web3.version
 * > Web3.givenProvider
 * > Web3.providers
 * > Web3.modules
 * ```
 *
 * # Web3.modules
 *
 * ```ts
 * Web3.modules
 * ```
 *
 * Will return an object with the classes of all major sub modules, to be able to instantiate them manually.
 *
 * #### Returns
 *
 *  `Object` A list of module constructors:
 *
 *
 *  + `Eth` - `Constructor`: The Eth module for interacting with the Ethereum network
 *
 *
 *  + `Net` - `Constructor`: The Net module for interacting with network properties.
 *
 *
 *  + `Personal` - `constructor`: The Personal module for interacting with the Ethereum accounts (web3.eth.personal).
 *
 * #### Example
 *
 * ```ts
 * Web3.modules
 * > {
 *   	Eth: Eth(provider),
 *   	Net: Net(provider),
 *   	Personal: Personal(provider),
 *   }
 * ```
 *
 * See details: {@link Web3.modules}
 *
 * # Web3 Instance
 *
 * The Web3 class is an umbrella package to house all Ethereum related modules.
 *
 * ```ts
 * import Web3 from 'web3';
 *
 * // "Web3.givenProvider" will be set if in an Ethereum supported browser.
 * const web3 = new Web3(Web3.givenProvider || 'ws://some.local-or-remote.node:8546');
 *
 * > web3.eth
 * > web3.utils
 * > web3.version
 * ```
 *
 * ### version
 *
 * Contains the current package version of the web3.js library.
 *
 * #### Returns
 * //todo enable when functionality added
 * // @see Web3.version
 *
 * ### utils
 *
 * Static accessible property of the Web3 class and property of the instance as well.
 *
 * ```ts
 * Web3.utils
 * web3.utils
 * ```
 *
 * Utility functions are also exposed on the `Web3` class object diretly.
 *
 * //todo enable when implemented
 * //See details: {@link Web3.utils}
 *
 * ### setProvider
 *
 * ```ts
 * web3.setProvider(myProvider)
 * web3.eth.setProvider(myProvider)
 * ...
 * ```
 *
 * Will change the provider for its module.
 *
 * **_NOTE:_** When called on the umbrella package web3 it will also set the provider for all sub modules web3.eth  etc.
 *
 * #### Parameters
 *  `Object`  - `myProvider`: a valid provider.
 *
 * #### Returns
 * `Boolean`
 *
 * See details: {@link Web3.setProvider}
 *
 * #### Example: Local Geth Node
 * ```ts
 * import Web3 from "web3";
 * let web3: Web3 = new Web3('http://localhost:8545');
 * // or
 * let web3: Web3 = new Web3(new Web3.providers.HttpProvider('http://localhost:8545'));
 *
 * // change provider
 * web3.setProvider('ws://localhost:8546');
 * // or
 * web3.setProvider(new Web3.providers.WebsocketProvider('ws://localhost:8546'));
 *
 * //todo add IPC provider
 * ```
 *
 * #### Example: Remote Geth Node
 *
 * ```ts
 * // Using a remote node provider, like Alchemy (https://www.alchemyapi.io/supernode), is simple.
 * import Web3 from "web3";
 * let web3: Web3 = new Web3("https://eth-mainnet.alchemyapi.io/v2/your-api-key");
 * ```
 *
 * ### providers
 *
 * ```ts
 * web3.providers
 * web3.eth.providers
 * ```
 * Contains the current available providers.
 *
 * #### Returns
 *  `Object` with the following providers:
 *
 *
 *  + `Object` - `HttpProvider`: HTTP provider, does not support subscriptions.
 *
 *
 *  + `Object` - `WebSocketProvider`: The WebSocket provider is the standard for usage in legacy browsers.
 *
 *
 *  + `Object` - `IpcProvider`: The IPC provider is used in node.js dapps when running a local node. Gives the most secure connection.
 *
 *
 * #### Example
 * ```ts
 * import { Web3 } from 'web3';
 * // use the given Provider or instantiate a new websocket provider
 * let web3 = new Web3(Web3.givenProvider || 'ws://remotenode.com:8546');
 * // or
 * let web3 = new Web3(Web3.givenProvider || new Web3.providers.WebsocketProvider('ws://remotenode.com:8546'));
 *
 * // Using the IPC provider in node.js
 * import { Web3 } from 'web3';
 * import { IpcProvider } from 'web3-providers-ipc';
 * var web3 = new Web3(new IpcProvider('/Users/myuser/Library/Ethereum/geth.ipc')); // mac os path
 * // on windows the path is: "\\\\.\\pipe\\geth.ipc"
 * // on linux the path is: "/users/myuser/.ethereum/geth.ipc"
 * ```
 * #### Configuration
 *
 * ```ts
 *
 * //===
 * //Http
 * //===
 *
 * import Web3HttpProvider, { HttpProviderOptions } from "web3-providers-http";
 *
 * let options: HttpProviderOptions = {
 * 	providerOptions: {
 * 		keepalive: true,
 * 		credentials: "omit",
 * 		headers: {
 * 			"Access-Control-Allow-Origin": "*",
 * 		},
 * 	},
 * };
 *
 *
 * var provider = new Web3HttpProvider("http://localhost:8545", options);
 * web3.setProvider(provider);
 *
 * //===
 * //WebSockets
 * //===
 * import Web3WsProvider, {
 * 	ClientOptions,
 * 	ClientRequestArgs,
 * 	ReconnectOptions,
 * } from "web3-providers-ws";
 *
 *
 * let clientOptions: ClientOptions = {
 * 	// Useful for credentialed urls, e.g: ws://username:password@localhost:8546
 * 	headers: {
 * 		authorization: "Basic username:password",
 * 	},
 * 	maxPayload: 100000000,
 * };
 *
 * // Enable auto reconnection
 * let reconnectOptions: ReconnectOptions = {
 * 	autoReconnect: true,
 * 	delay: 5000, // ms
 * 	maxAttempts: 5,
 * };
 *
 * //clientOptions and reconnectOptions are optional
 * //clientOptions: ClientOptions | ClientRequestArgs
 * let ws = new Web3WsProvider(
 * "ws://localhost:8546",
 * clientOptions,
 * reconnectOptions
 * );
 * web3.setProvider(ws);
 *
 * ```
 * More information for the Http and Websocket provider modules can be found here:
 *
 *
 * - {@link HttpProvider}
 *
 *
 * - {@link WebSocketProvider}
 *
 * See details: {@link Web3.providers}
 *
 *
 * ### givenProvider
 *
 * ```ts
 * web3.givenProvider
 * web3.eth.givenProvider
 * ...
 * ```
 * When using web3.js in an Ethereum compatible browser, it will set with the current native provider by that browser.
 * Will return the given provider by the (browser) environment, otherwise `undefined`.
 *
 * #### Returns
 * `Object` -  The given provider set or `undefined`.
 *
 * See details: {@link Web3.givenProvider}
 *
 * ### currentProvider
 *
 * ```ts
 * web3.currentProvider
 * web3.eth.currentProvider
 * ...
 * ```
 * Will return the current provider, otherwise `undefined`.
 *
 * #### Returns
 * `Object`: The current provider, otherwise `undefined`.
 *
 * See details: {@link Web3.currentProvider}
 *
 * ### BatchRequest
 *
 * ```ts
 * new web3.BatchRequest()
 * new web3.BatchRequest()
 * ...
 * ```
 * Class to create and execute batch requests.
 *
 *  #### Parameters
 *  none
 *
 * #### Returns
 * `Object`: With the following methods:
 *
 * + `add(request)`: To add a request object to the batch call.
 *
 * + `execute()` : To execute the batch request.
 *
 * #### Example
 * ```ts
 * let request1: JsonRpcOptionalRequest = {
 * 	id: 10,
 * 	method: 'eth_getBalance',
 * 	params: ["0xdc6bad79dab7ea733098f66f6c6f9dd008da3258", 'latest'],
 * };
 * let request2: JsonRpcOptionalRequest = {
 * 	id: 11,
 * 	method: 'eth_getBalance',
 * 	params: ["0x962f9a9c2a6c092474d24def35eccb3d9363265e", 'latest'],
 * };
 *
 * const batch = new web3.BatchRequest();
 *
 *  batch.add(request1);
 *  batch.add(request2);
 * // add returns a deferred promise which can be used to run specific code after completion of each respective request.
 * //const request2Promise = batch.add(request2);
 *
 * const response = await batch.execute();
 * ```
 * See details: {@link Web3.BatchRequest}
 */
/**
 * This comment _supports3_ [Markdown](https://marked.js.org/)
 */
import Web3 from './web3.js';
export * from './types.js';
export * from './web3_eip6963.js';
export default Web3;
/**
 * Named exports for all objects which are the default-exported-object in their packages
 */
export { Web3 };
export { Web3Context, Web3PluginBase, Web3EthPluginBase, Web3PromiEvent } from 'web3-core';
export { Web3Eth } from 'web3-eth';
export { Contract } from 'web3-eth-contract';
export { Iban } from 'web3-eth-iban';
export { Personal } from 'web3-eth-personal';
export { Net } from 'web3-net';
export { HttpProvider } from 'web3-providers-http';
export { WebSocketProvider } from 'web3-providers-ws';
export { Web3Validator } from 'web3-validator';
/**
 * Export all packages grouped by name spaces
 */
export * as core from 'web3-core';
export * as errors from 'web3-errors';
export * as eth from './eth.exports.js';
export * as net from 'web3-net';
export * as providers from './providers.exports.js';
export * as rpcMethods from 'web3-rpc-methods';
export * as types from 'web3-types';
export * as utils from 'web3-utils';
export * as validator from 'web3-validator';
/**
 * Export all types from `web3-types` without a namespace (in addition to being available at `types` namespace).
 * To enable the user to write: `function something(): Web3Api` without the need for `types.Web3Api`.
 * And the same for `web3-errors`. Because this package contains error classes and constants.
 */
export * from 'web3-errors';
export * from 'web3-types';
//# sourceMappingURL=index.js.map