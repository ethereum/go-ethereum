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

import {
	Web3Context,
	Web3EventEmitter,
	Web3PromiEvent,
	Web3ConfigEvent,
	Web3SubscriptionManager,
	Web3SubscriptionConstructor,
} from 'web3-core';
import {
	ContractExecutionError,
	ContractTransactionDataAndInputError,
	SubscriptionError,
	Web3ContractError,
} from 'web3-errors';
import {
	createAccessList,
	call,
	estimateGas,
	getLogs,
	sendTransaction,
	decodeEventABI,
	NewHeadsSubscription,
	ALL_EVENTS,
	ALL_EVENTS_ABI,
	SendTransactionEvents,
	TransactionMiddleware,
} from 'web3-eth';
import {
	decodeFunctionCall,
	decodeFunctionReturn,
	encodeEventSignature,
	encodeFunctionSignature,
	decodeContractErrorData,
	isAbiErrorFragment,
	isAbiEventFragment,
	isAbiFunctionFragment,
	jsonInterfaceMethodToString,
} from 'web3-eth-abi';
import {
	AbiErrorFragment,
	AbiEventFragment,
	AbiFragment,
	AbiFunctionFragment,
	ContractAbi,
	ContractConstructorArgs,
	ContractEvent,
	ContractEvents,
	ContractMethod,
	ContractMethodInputParameters,
	ContractMethodOutputParameters,
	Address,
	BlockNumberOrTag,
	BlockTags,
	EthExecutionAPI,
	Filter,
	FilterAbis,
	HexString,
	LogsInput,
	Mutable,
	ContractInitOptions,
	NonPayableCallOptions,
	PayableCallOptions,
	DataFormat,
	DEFAULT_RETURN_FORMAT,
	Numbers,
	Web3ValidationErrorObject,
	EventLog,
	ContractAbiWithSignature,
	ContractOptions,
	TransactionReceipt,
	FormatType,
	DecodedParams,
} from 'web3-types';
import {
	format,
	isDataFormat,
	keccak256,
	toChecksumAddress,
	isContractInitOptions,
} from 'web3-utils';
import {
	isNullish,
	validator,
	utils as validatorUtils,
	ValidationSchemaInput,
	Web3ValidatorError,
} from 'web3-validator';
import { encodeEventABI, encodeMethodABI } from './encoding.js';
import { ContractLogsSubscription } from './contract_log_subscription.js';
import {
	ContractEventOptions,
	NonPayableMethodObject,
	NonPayableTxOptions,
	PayableMethodObject,
	PayableTxOptions,
	Web3ContractContext,
} from './types.js';
import {
	getCreateAccessListParams,
	getEstimateGasParams,
	getEthTxCallParams,
	getSendTxParams,
	isWeb3ContractContext,
} from './utils.js';
// eslint-disable-next-line import/no-cycle
import { DeployerMethodClass } from './contract-deployer-method-class.js';
// eslint-disable-next-line import/no-cycle
import { ContractSubscriptionManager } from './contract-subscription-manager.js';

type ContractBoundMethod<
	Abi extends AbiFunctionFragment,
	Method extends ContractMethod<Abi> = ContractMethod<Abi>,
> = (
	...args: Abi extends undefined
		? // eslint-disable-next-line @typescript-eslint/no-explicit-any
		  any[]
		: Method['Inputs'] extends never
		? // eslint-disable-next-line @typescript-eslint/no-explicit-any
		  any[]
		: Method['Inputs']
) => Method['Abi']['stateMutability'] extends 'payable' | 'pure'
	? PayableMethodObject<Method['Inputs'], Method['Outputs']>
	: NonPayableMethodObject<Method['Inputs'], Method['Outputs']>;

export type ContractOverloadedMethodInputs<AbiArr extends ReadonlyArray<unknown>> = NonNullable<
	AbiArr extends readonly []
		? undefined
		: AbiArr extends readonly [infer A, ...infer R]
		? A extends AbiFunctionFragment
			? ContractMethodInputParameters<A['inputs']> | ContractOverloadedMethodInputs<R>
			: undefined
		: undefined
>;

export type ContractOverloadedMethodOutputs<AbiArr extends ReadonlyArray<unknown>> = NonNullable<
	AbiArr extends readonly []
		? undefined
		: AbiArr extends readonly [infer A, ...infer R]
		? A extends AbiFunctionFragment
			? ContractMethodOutputParameters<A['outputs']> | ContractOverloadedMethodOutputs<R>
			: undefined
		: undefined
>;

// To avoid circular dependency between types and encoding, declared these types here.
export type ContractMethodsInterface<Abi extends ContractAbi> = {
	[MethodAbi in FilterAbis<
		Abi,
		AbiFunctionFragment & { type: 'function' }
	> as MethodAbi['name']]: ContractBoundMethod<MethodAbi>;
	// To allow users to use method signatures
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
} & { [key: string]: ContractBoundMethod<any> };

export type ContractMethodSend = Web3PromiEvent<
	FormatType<TransactionReceipt, DataFormat>,
	SendTransactionEvents<DataFormat>
>;

/**
 * @hidden
 * The event object can be accessed from `myContract.events.myEvent`.
 *
 * \> Remember: To subscribe to an event, your provider must have support for subscriptions.
 *
 * ```ts
 * const subscription = await myContract.events.MyEvent([options])
 * ```
 *
 * @param options - The options used to subscribe for the event
 * @returns - A Promise resolved with {@link ContractLogsSubscription} object
 */
export type ContractBoundEvent = (options?: ContractEventOptions) => ContractLogsSubscription;

// To avoid circular dependency between types and encoding, declared these types here.
export type ContractEventsInterface<
	Abi extends ContractAbi,
	Events extends ContractEvents<Abi> = ContractEvents<Abi>,
> = {
	[Name in keyof Events | 'allEvents']: ContractBoundEvent;
} & {
	[key: string]: ContractBoundEvent;
};

// To avoid circular dependency between types and encoding, declared these types here.
export type ContractEventEmitterInterface<Abi extends ContractAbi> = {
	[EventAbi in FilterAbis<
		Abi,
		AbiFunctionFragment & { type: 'event' }
	> as EventAbi['name']]: ContractEvent<EventAbi>['Inputs'];
};

type EventParameters = Parameters<typeof encodeEventABI>[2];

const contractSubscriptions = {
	logs: ContractLogsSubscription,
	newHeads: NewHeadsSubscription,
	newBlockHeaders: NewHeadsSubscription,
};

type ContractSubscriptions = typeof contractSubscriptions;

/**
 * The `web3.eth.Contract` makes it easy to interact with smart contracts on the ethereum blockchain.
 * For using contract package, first install Web3 package using: `npm i web3` or `yarn add web3` based on your package manager, after that contracts features can be used as mentioned in following snippet.
 * ```ts
 *
 * import { Web3 } from 'web3';
 *
 * const web3 = new Web3('https://127.0.0.1:4545');
 * const abi = [...] as const; // your contract ABI
 *
 * let contract = new web3.eth.Contract(abi,'0xdAC17F958D2ee523a2206206994597C13D831ec7');
 * await contract.methods.balanceOf('0xdAC17F958D2ee523a2206206994597C13D831ec7').call();
 * ```
 * For using individual package install `web3-eth-contract` and `web3-core` packages using: `npm i web3-eth-contract web3-core` or `yarn add web3-eth-contract web3-core`. This is more efficient approach for building lightweight applications.
 * ```ts
 *
 * import { Web3Context } from 'web3-core';
 * import { Contract } from 'web3-eth-contract';
 *
 * const abi = [...] as const; // your contract ABI
 *
 * let contract = new web3.eth.Contract(
 * 	abi,
 * 	'0xdAC17F958D2ee523a2206206994597C13D831ec7'
 * 	 new Web3Context('http://127.0.0.1:8545'));
 *
 * await contract.methods.balanceOf('0xdAC17F958D2ee523a2206206994597C13D831ec7').call();
 * ```
 * ## Generated Methods
 * Following methods are generated by web3.js contract object for each of contract functions by using its ABI.
 *
 * ### send
 * This is used to send a transaction to the smart contract and execute its method. Note this can alter the smart contract state.
 *
 * #### Parameters
 * options?: PayableTxOptions | NonPayableTxOptions
 *
 * #### Returns
 * [Web3PromiEvent](/api/web3/namespace/core#Web3PromiEvent) : Web3 Promi Event
 *
 * ```ts
 * // using the promise
 * myContract.methods.myMethod(123).send({from: '0xde0B295669a9FD93d5F28D9Ec85E40f4cb697BAe'})
 * 	.then(function(receipt){
 * 		// other parts of code to use receipt
 * 	});
 *
 *
 * // using the event emitter
 * myContract.methods.myMethod(123).send({from: '0xde0B295669a9FD93d5F28D9Ec85E40f4cb697BAe'})
 * 	.on('transactionHash', function(hash){
 * 		// ...
 * 	})
 * 	.on('confirmation', function(confirmationNumber, receipt){
 * 		// ...
 * 	})
 * 	.on('receipt', function(receipt){
 * 		// ...
 * 	})
 * 	.on('error', function(error, receipt) {
 * 		// ...
 * 	});
 *
 * ```
 *
 * ### call
 * This will execute smart contract method in the EVM without sending any transaction. Note calling cannot alter the smart contract state.
 *
 * #### Parameters
 * options?: PayableCallOptions | NonPayableCallOptions,
 * block?: BlockNumberOrTag,
 *
 * #### Returns
 * Promise : having results of call
 *
 * ```ts
 *
 * let myContract = new web3.eth.Contract(abi, address);
 *
 * myContract.methods.myFunction().call()
 * .then(console.log);
 *
 * ```
 * ### estimateGas
 * Returns the amount of gas consumed by executing the method in EVM without creating a new transaction on the blockchain. The returned amount can be used as a gas estimate for executing the transaction publicly. The actual gas used can be different when sending the transaction later, as the state of the smart contract can be different at that time.
 *
 * #### Parameters
 * options?: PayableCallOptions,
 * returnFormat: ReturnFormat = DEFAULT_RETURN_FORMAT as ReturnFormat,
 *
 * #### Returns
 * Promise: The gas amount estimated.
 *
 * ```ts
 * const estimatedGas = await contract.methods.approve('0xdAC17F958D2ee523a2206206994597C13D831ec7', 300)
 *     .estimateGas();
 *
 * ```
 *
 * ### encodeABI
 * Encodes the ABI for this method. The resulting hex string is 32-bit function signature hash plus the passed parameters in Solidity tightly packed format. This can be used to send a transaction, call a method, or pass it into another smart contract’s method as arguments. Set the data field on web3.eth.sendTransaction options as the encodeABI() result and it is the same as calling the contract method with contract.myMethod.send().
 *
 * Some use cases for encodeABI() include: preparing a smart contract transaction for a multisignature wallet, working with offline wallets and cold storage and creating transaction payload for complex smart contract proxy calls.
 *
 * #### Parameters
 * None
 *
 * #### Returns
 * String: The encoded ABI.
 *
 * ```ts
 * const encodedABI = await contract.methods.approve('0xdAC17F958D2ee523a2206206994597C13D831ec7', 300)
 *     .encodeABI();
 *
 * ```
 *

 * ### decodeMethodData
 * Decodes the given ABI-encoded data, revealing both the method name and the parameters used in the smart contract call.
 * This function reverses the encoding process happens at the method `encodeABI`.
 * It's particularly useful for debugging and understanding the interactions with and between smart contracts.
 *
 * #### Parameters
 *
 * - `data` **HexString**: The string of ABI-encoded data that needs to be decoded. This should include the method signature and the encoded parameters.
 *
 * #### Returns
 *
 * - **Object**: This object combines both the decoded parameters and the method name in a readable format. Specifically, the returned object contains:
 *   - `__method__` **String**: The name of the contract method, reconstructed from the ABI.
 *   - `__length__` **Number**: The number of parameters decoded.
 *   - Additional properties representing each parameter by name, as well as their position and values.
 *
 * #### Example
 *
 * Given an ABI-encoded string from a transaction, you can decode this data to identify the method called and the parameters passed.
 * Here's a simplified example:
 *
 *
 * ```typescript
 * const GreeterAbi = [
 * 	{
 * 		inputs: [
 * 			{
 * 				internalType: 'string',
 * 				name: '_greeting',
 * 				type: 'string',
 * 			},
 * 		],
 * 		name: 'setGreeting',
 * 		outputs: [],
 * 		type: 'function',
 * 	},
 * ];
 * const contract = new Contract(GreeterAbi); // Initialize with your contract's ABI
 *
 * // The ABI-encoded data string for "setGreeting('Hello World')"
 * const encodedData =
 * 	'0xa41368620000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000b48656c6c6f20576f726c64000000000000000000000000000000000000000000';
 *
 * try {
 * 	const decoded = contract.decodeMethodData(encodedData);
 * 	console.log(decoded.__method__); // Outputs: "setGreeting(string)"
 * 	console.log(decoded); // Outputs the detailed parameter data
 * 	// This tells that the method called was `setGreeting` with a single string parameter "Hello World":
 * 	// {
 * 	//   __method__: 'setGreeting(string)',
 * 	//   __length__: 1,
 * 	//   '0': 'Hello World',
 * 	//   _greeting: 'Hello World'
 * 	// }
 * } catch (error) {
 * 	console.error(error);
 * }
 * ```
 *

 * ### createAccessList
 * This will create an access list a method execution will access when executed in the EVM.
 * Note: You must specify a from address and gas if it’s not specified in options when instantiating parent contract object.
 *
 * #### Parameters
 * options?: PayableCallOptions | NonPayableCallOptions,
 * block?: BlockNumberOrTag,
 *
 * #### Returns
 * Promise: The generated access list for transaction.
 *
 * ```ts
 * const accessList = await contract.methods.approve('0xbEe634C21c16F05B03B704BaE071536121e6cFeA', 300)
 *     .createAccessList({
 *         from: "0x9992695e1053bb737d3cfae4743dcfc4b94f203d"
 *    });
 * ```
 *
 */
export class Contract<Abi extends ContractAbi>
	extends Web3Context<EthExecutionAPI, ContractSubscriptions>
	implements Web3EventEmitter<ContractEventEmitterInterface<Abi>>
{
	protected override _subscriptionManager: ContractSubscriptionManager<EthExecutionAPI>;

	public override get subscriptionManager(): ContractSubscriptionManager<EthExecutionAPI> {
		return this._subscriptionManager;
	}

	/**
	 * The options `object` for the contract instance. `from`, `gas` and `gasPrice` are used as fallback values when sending transactions.
	 *
	 * ```ts
	 * myContract.options;
	 * > {
	 *     address: '0x1234567890123456789012345678901234567891',
	 *     jsonInterface: [...],
	 *     from: '0xde0B295669a9FD93d5F28D9Ec85E40f4cb697BAe',
	 *     gasPrice: '10000000000000',
	 *     gas: 1000000
	 * }
	 *
	 * myContract.options.from = '0x1234567890123456789012345678901234567891'; // default from address
	 * myContract.options.gasPrice = '20000000000000'; // default gas price in wei
	 * myContract.options.gas = 5000000; // provide as fallback always 5M gas
	 * ```
	 */

	public readonly options: ContractOptions;
	private transactionMiddleware?: TransactionMiddleware;
	/**
	 * Set to true if you want contracts' defaults to sync with global defaults.
	 */
	public syncWithContext = false;

	private _errorsInterface!: AbiErrorFragment[];
	private _jsonInterface!: ContractAbiWithSignature;
	private _address?: Address;
	private _functions: Record<
		string,
		{
			signature: string;
			// eslint-disable-next-line @typescript-eslint/no-explicit-any
			method: ContractBoundMethod<any>;
		}
	> = {};
	private readonly _overloadedMethodAbis: Map<string, AbiFunctionFragment[]>;
	private _methods!: ContractMethodsInterface<Abi>;
	private _events!: ContractEventsInterface<Abi>;
	/**
	 * Set property to `data`, `input`, or `both` to change the property of the contract being sent to the
	 * RPC provider when using contract methods.
	 * Default is `input`
	 */

	private context?: Web3Context;
	/**
	 * Creates a new contract instance with all its methods and events defined in its ABI provided.
	 *
	 * ```ts
	 * new web3.eth.Contract(jsonInterface[, address][, options])
	 * ```
	 *
	 * @param jsonInterface - The JSON interface for the contract to instantiate.
	 * @param address - The address of the smart contract to call.
	 * @param options - The options of the contract. Some are used as fallbacks for calls and transactions.
	 * @param context - The context of the contract used for customizing the behavior of the contract.
	 * @returns - The contract instance with all its methods and events.
	 *
	 * ```ts title="Example"
	 * var myContract = new web3.eth.Contract([...], '0xde0B295669a9FD93d5F28D9Ec85E40f4cb697BAe', {
	 *   from: '0x1234567890123456789012345678901234567891', // default from address
	 *   gasPrice: '20000000000' // default gas price in wei, 20 gwei in this case
	 * });
	 * ```
	 *
	 * To use the type safe interface for these contracts you have to include the ABI definitions in your TypeScript project and then declare these as `const`.
	 *
	 * ```ts title="Example"
	 * const myContractAbi = [....] as const; // ABI definitions
	 * const myContract = new web3.eth.Contract(myContractAbi, '0xde0B295669a9FD93d5F28D9Ec85E40f4cb697BAe');
	 * ```
	 */
	public constructor(
		jsonInterface: Abi,
		context?: Web3ContractContext | Web3Context,
		returnFormat?: DataFormat,
	);
	public constructor(
		jsonInterface: Abi,
		address?: Address,
		contextOrReturnFormat?: Web3ContractContext | Web3Context | DataFormat,
		returnFormat?: DataFormat,
	);
	public constructor(
		jsonInterface: Abi,
		options?: ContractInitOptions,
		contextOrReturnFormat?: Web3ContractContext | Web3Context | DataFormat,
		returnFormat?: DataFormat,
	);
	public constructor(
		jsonInterface: Abi,
		address: Address | undefined,
		options: ContractInitOptions,
		contextOrReturnFormat?: Web3ContractContext | Web3Context | DataFormat,
		returnFormat?: DataFormat,
	);
	public constructor(
		jsonInterface: Abi,
		addressOrOptionsOrContext?:
			| Address
			| ContractInitOptions
			| Web3ContractContext
			| Web3Context,
		optionsOrContextOrReturnFormat?:
			| ContractInitOptions
			| Web3ContractContext
			| Web3Context
			| DataFormat,
		contextOrReturnFormat?: Web3ContractContext | Web3Context | DataFormat,
		returnFormat?: DataFormat,
	) {
		// eslint-disable-next-line no-nested-ternary
		const options = isContractInitOptions(addressOrOptionsOrContext)
			? addressOrOptionsOrContext
			: isContractInitOptions(optionsOrContextOrReturnFormat)
			? optionsOrContextOrReturnFormat
			: undefined;

		let contractContext;
		if (isWeb3ContractContext(addressOrOptionsOrContext)) {
			contractContext = addressOrOptionsOrContext;
		} else if (isWeb3ContractContext(optionsOrContextOrReturnFormat)) {
			contractContext = optionsOrContextOrReturnFormat;
		} else {
			contractContext = contextOrReturnFormat;
		}

		let provider;
		if (
			typeof addressOrOptionsOrContext === 'object' &&
			'provider' in addressOrOptionsOrContext
		) {
			provider = addressOrOptionsOrContext.provider;
		} else if (
			typeof optionsOrContextOrReturnFormat === 'object' &&
			'provider' in optionsOrContextOrReturnFormat
		) {
			provider = optionsOrContextOrReturnFormat.provider;
		} else if (
			typeof contextOrReturnFormat === 'object' &&
			'provider' in contextOrReturnFormat
		) {
			provider = contextOrReturnFormat.provider;
		} else {
			provider = Contract.givenProvider;
		}

		super({
			...contractContext,
			provider,
			registeredSubscriptions: contractSubscriptions,
		});

		this._subscriptionManager = new ContractSubscriptionManager<
			EthExecutionAPI,
			ContractSubscriptions
		>(super.subscriptionManager, this);

		// Init protected properties
		if ((contractContext as Web3Context)?.wallet) {
			this._wallet = (contractContext as Web3Context).wallet;
		}
		if ((contractContext as Web3Context)?.accountProvider) {
			this._accountProvider = (contractContext as Web3Context).accountProvider;
		}

		if (
			!isNullish(options) &&
			!isNullish(options.data) &&
			!isNullish(options.input) &&
			this.config.contractDataInputFill !== 'both'
		)
			throw new ContractTransactionDataAndInputError({
				data: options.data as HexString,
				input: options.input as HexString,
			});
		this._overloadedMethodAbis = new Map<string, AbiFunctionFragment[]>();

		// eslint-disable-next-line no-nested-ternary
		const returnDataFormat = isDataFormat(contextOrReturnFormat)
			? contextOrReturnFormat
			: isDataFormat(optionsOrContextOrReturnFormat)
			? optionsOrContextOrReturnFormat
			: returnFormat ?? this.defaultReturnFormat;
		const address =
			typeof addressOrOptionsOrContext === 'string' ? addressOrOptionsOrContext : undefined;
		this.config.contractDataInputFill =
			(options as ContractInitOptions)?.dataInputFill ?? this.config.contractDataInputFill;
		this._parseAndSetJsonInterface(jsonInterface, returnDataFormat);

		if (this.defaultReturnFormat !== returnDataFormat) {
			this.defaultReturnFormat = returnDataFormat;
		}

		if (!isNullish(address)) {
			this._parseAndSetAddress(address, returnDataFormat);
		}

		this.options = {
			address,
			jsonInterface: this._jsonInterface,
			gas: options?.gas ?? options?.gasLimit,
			gasPrice: options?.gasPrice,
			from: options?.from,
			input: options?.input,
			data: options?.data,
		};

		this.syncWithContext = (options as ContractInitOptions)?.syncWithContext ?? false;
		if (contractContext instanceof Web3Context) {
			this.subscribeToContextEvents(contractContext);
		}
		Object.defineProperty(this.options, 'address', {
			set: (value: Address) => this._parseAndSetAddress(value, returnDataFormat),
			get: () => this._address,
		});

		Object.defineProperty(this.options, 'jsonInterface', {
			set: (value: ContractAbi) => this._parseAndSetJsonInterface(value, returnDataFormat),
			get: () => this._jsonInterface,
		});

		if (contractContext instanceof Web3Context) {
			contractContext.on(Web3ConfigEvent.CONFIG_CHANGE, event => {
				// eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
				this.setConfig({ [event.name]: event.newValue });
			});
		}
	}

	public setTransactionMiddleware(transactionMiddleware: TransactionMiddleware) {
		this.transactionMiddleware = transactionMiddleware;
	}

	public getTransactionMiddleware() {
		return this.transactionMiddleware;
	}

	/**
	 * Subscribe to an event.
	 *
	 * ```ts
	 * await myContract.events.MyEvent([options])
	 * ```
	 *
	 * There is a special event `allEvents` that can be used to subscribe all events.
	 *
	 * ```ts
	 * await myContract.events.allEvents([options])
	 * ```
	 *
	 * @returns - When individual event is accessed will returns {@link ContractBoundEvent} object
	 */
	public get events() {
		return this._events;
	}

	/**
	 * Creates a transaction object for that method, which then can be `called`, `send`, `estimated`, `createAccessList` , or `ABI encoded`.
	 *
	 * The methods of this smart contract are available through:
	 *
	 * The name: `myContract.methods.myMethod(123)`
	 * The name with parameters: `myContract.methods['myMethod(uint256)'](123)`
	 * The signature `myContract.methods['0x58cf5f10'](123)`
	 *
	 * This allows calling functions with same name but different parameters from the JavaScript contract object.
	 *
	 * \> The method signature does not provide a type safe interface, so we recommend to use method `name` instead.
	 *
	 * ```ts
	 * // calling a method
	 * const result = await myContract.methods.myMethod(123).call({from: '0xde0B295669a9FD93d5F28D9Ec85E40f4cb697BAe'});
	 *
	 * // or sending and using a promise
	 * const receipt = await myContract.methods.myMethod(123).send({from: '0xde0B295669a9FD93d5F28D9Ec85E40f4cb697BAe'});
	 *
	 * // or sending and using the events
	 * const sendObject = myContract.methods.myMethod(123).send({from: '0xde0B295669a9FD93d5F28D9Ec85E40f4cb697BAe'});
	 * sendObject.on('transactionHash', function(hash){
	 *   ...
	 * });
	 * sendObject.on('receipt', function(receipt){
	 *   ...
	 * });
	 * sendObject.on('confirmation', function(confirmationNumber, receipt){
	 *   ...
	 * });
	 * sendObject.on('error', function(error, receipt) {
	 *   ...
	 * });
	 * ```
	 *
	 * @returns - Either returns {@link PayableMethodObject} or {@link NonPayableMethodObject} based on the definitions of the ABI of that contract.
	 */
	public get methods() {
		return this._methods;
	}

	/**
	 * Clones the current contract instance. This doesn't deploy contract on blockchain and only creates a local clone.
	 *
	 * @returns - The new contract instance.
	 *
	 * ```ts
	 * const contract1 = new web3.eth.Contract(abi, address, {gasPrice: '12345678', from: fromAddress});
	 *
	 * const contract2 = contract1.clone();
	 * contract2.options.address = '0xdAC17F958D2ee523a2206206994597C13D831ec7';
	 *
	 * (contract1.options.address !== contract2.options.address);
	 * > true
	 * ```
	 */
	public clone() {
		let newContract: Contract<Abi>;
		if (this.options.address) {
			newContract = new Contract<Abi>(
				[...this._jsonInterface, ...this._errorsInterface] as unknown as Abi,
				this.options.address,
				{
					gas: this.options.gas,
					gasPrice: this.options.gasPrice,
					from: this.options.from,
					input: this.options.input,
					data: this.options.data,
					provider: this.currentProvider,
					syncWithContext: this.syncWithContext,
					dataInputFill: this.config.contractDataInputFill,
				},
				this.getContextObject(),
			);
		} else {
			newContract = new Contract<Abi>(
				[...this._jsonInterface, ...this._errorsInterface] as unknown as Abi,
				{
					gas: this.options.gas,
					gasPrice: this.options.gasPrice,
					from: this.options.from,
					input: this.options.input,
					data: this.options.data,
					provider: this.currentProvider,
					syncWithContext: this.syncWithContext,
					dataInputFill: this.config.contractDataInputFill,
				},
				this.getContextObject(),
			);
		}
		if (this.context) newContract.subscribeToContextEvents(this.context);

		return newContract;
	}

	/**
	 * Call this function to deploy the contract to the blockchain. After successful deployment the promise will resolve with a new contract instance.
	 *
	 * ```ts
	 * myContract.deploy({
	 *   input: '0x12345...', // data keyword can be used, too.
	 *   arguments: [123, 'My String']
	 * })
	 * .send({
	 *   from: '0x1234567890123456789012345678901234567891',
	 *   gas: 1500000,
	 *   gasPrice: '30000000000000'
	 * }, function(error, transactionHash){ ... })
	 * .on('error', function(error){ ... })
	 * .on('transactionHash', function(transactionHash){ ... })
	 * .on('receipt', function(receipt){
	 *  console.log(receipt.contractAddress) // contains the new contract address
	 * })
	 * .on('confirmation', function(confirmationNumber, receipt){ ... })
	 * .then(function(newContractInstance){
	 *   console.log(newContractInstance.options.address) // instance with the new contract address
	 * });
	 *
	 *
	 * // When the data is already set as an option to the contract itself
	 * myContract.options.data = '0x12345...';
	 *
	 * myContract.deploy({
	 *   arguments: [123, 'My String']
	 * })
	 * .send({
	 *   from: '0x1234567890123456789012345678901234567891',
	 *   gas: 1500000,
	 *   gasPrice: '30000000000000'
	 * })
	 * .then(function(newContractInstance){
	 *   console.log(newContractInstance.options.address) // instance with the new contract address
	 * });
	 *
	 *
	 * // Simply encoding
	 * myContract.deploy({
	 *   input: '0x12345...',
	 *   arguments: [123, 'My String']
	 * })
	 * .encodeABI();
	 * > '0x12345...0000012345678765432'
	 *
	 *
	 * // decoding
	 * myContract.deploy({
	 *   input: '0x12345...',
	 *   // arguments: [123, 'My Greeting'] if you just need to decode the data, you can skip the arguments
	 * })
	 * .decodeData('0x12345...0000012345678765432');
	 * > {
	 *      __method__: 'constructor',
	 *      __length__: 2,
	 *      '0': '123',
	 *      _id: '123',
	 *      '1': 'My Greeting',
	 *      _greeting: 'My Greeting',
	 *   }
	 *
	 *
	 * // Gas estimation
	 * myContract.deploy({
	 *   input: '0x12345...',
	 *   arguments: [123, 'My String']
	 * })
	 * .estimateGas(function(err, gas){
	 *   console.log(gas);
	 * });
	 * ```
	 *
	 * @returns - The transaction object
	 */
	public deploy(deployOptions?: {
		/**
		 * The byte code of the contract.
		 */
		data?: HexString;
		input?: HexString;
		/**
		 * The arguments which get passed to the constructor on deployment.
		 */
		arguments?: ContractConstructorArgs<Abi>;
	}): DeployerMethodClass<Abi> {
		return new DeployerMethodClass(this, deployOptions);
	}

	/**
	 * Gets past events for this contract.
	 *
	 * ```ts
	 * const events = await myContract.getPastEvents('MyEvent', {
	 *   filter: {myIndexedParam: [20,23], myOtherIndexedParam: '0x123456789...'}, // Using an array means OR: e.g. 20 or 23
	 *   fromBlock: 0,
	 *   toBlock: 'latest'
	 * });
	 *
	 * > [{
	 *   returnValues: {
	 *       myIndexedParam: 20,
	 *       myOtherIndexedParam: '0x123456789...',
	 *       myNonIndexParam: 'My String'
	 *   },
	 *   raw: {
	 *       data: '0x7f9fade1c0d57a7af66ab4ead79fade1c0d57a7af66ab4ead7c2c2eb7b11a91385',
	 *       topics: ['0xfd43ade1c09fade1c0d57a7af66ab4ead7c2c2eb7b11a91ffdd57a7af66ab4ead7', '0x7f9fade1c0d57a7af66ab4ead79fade1c0d57a7af66ab4ead7c2c2eb7b11a91385']
	 *   },
	 *   event: 'MyEvent',
	 *   signature: '0xfd43ade1c09fade1c0d57a7af66ab4ead7c2c2eb7b11a91ffdd57a7af66ab4ead7',
	 *   logIndex: 0,
	 *   transactionIndex: 0,
	 *   transactionHash: '0x7f9fade1c0d57a7af66ab4ead79fade1c0d57a7af66ab4ead7c2c2eb7b11a91385',
	 *   blockHash: '0xfd43ade1c09fade1c0d57a7af66ab4ead7c2c2eb7b11a91ffdd57a7af66ab4ead7',
	 *   blockNumber: 1234,
	 *   address: '0xde0B295669a9FD93d5F28D9Ec85E40f4cb697BAe'
	 * },{
	 *   ...
	 * }]
	 * ```
	 *
	 * @param eventName - The name of the event in the contract, or `allEvents` to get all events.
	 * @param filter - The filter options used to get events.
	 * @param returnFormat - Return format
	 * @returns - An array with the past event `Objects`, matching the given event name and filter.
	 */
	public async getPastEvents<ReturnFormat extends DataFormat = typeof DEFAULT_RETURN_FORMAT>(
		returnFormat?: ReturnFormat,
	): Promise<(string | EventLog)[]>;
	public async getPastEvents<ReturnFormat extends DataFormat = typeof DEFAULT_RETURN_FORMAT>(
		eventName: keyof ContractEvents<Abi> | 'allEvents' | 'ALLEVENTS',
		returnFormat?: ReturnFormat,
	): Promise<(string | EventLog)[]>;
	public async getPastEvents<ReturnFormat extends DataFormat = typeof DEFAULT_RETURN_FORMAT>(
		filter: Omit<Filter, 'address'>,
		returnFormat?: ReturnFormat,
	): Promise<(string | EventLog)[]>;
	public async getPastEvents<ReturnFormat extends DataFormat = typeof DEFAULT_RETURN_FORMAT>(
		eventName: keyof ContractEvents<Abi> | 'allEvents' | 'ALLEVENTS',
		filter: Omit<Filter, 'address'>,
		returnFormat?: ReturnFormat,
	): Promise<(string | EventLog)[]>;
	public async getPastEvents<ReturnFormat extends DataFormat = typeof DEFAULT_RETURN_FORMAT>(
		param1?:
			| keyof ContractEvents<Abi>
			| 'allEvents'
			| 'ALLEVENTS'
			| Omit<Filter, 'address'>
			| ReturnFormat,
		param2?: Omit<Filter, 'address'> | ReturnFormat,
		param3?: ReturnFormat,
	): Promise<(string | EventLog)[]> {
		const eventName: string = typeof param1 === 'string' ? param1 : ALL_EVENTS;

		const options =
			// eslint-disable-next-line no-nested-ternary
			typeof param1 !== 'string' && !isDataFormat(param1)
				? (param1 as Omit<Filter, 'address'>)
				: !isDataFormat(param2)
				? param2
				: {};

		// eslint-disable-next-line no-nested-ternary
		const returnFormat = isDataFormat(param1)
			? param1
			: isDataFormat(param2)
			? param2
			: param3 ?? this.defaultReturnFormat;

		const abi =
			eventName === 'allEvents' || eventName === ALL_EVENTS
				? ALL_EVENTS_ABI
				: (this._jsonInterface.find(
						j => 'name' in j && j.name === eventName,
				  ) as AbiEventFragment & { signature: string });

		if (!abi) {
			throw new Web3ContractError(`Event ${String(eventName)} not found.`);
		}

		const { fromBlock, toBlock, topics, address } = encodeEventABI(
			this.options,
			abi,
			options ?? {},
		);

		const logs = await getLogs(this, { fromBlock, toBlock, topics, address }, returnFormat);
		const decodedLogs = logs
			? logs.map(log =>
					typeof log === 'string'
						? log
						: decodeEventABI(abi, log as LogsInput, this._jsonInterface, returnFormat),
			  )
			: [];

		const filter = options?.filter ?? {};
		const filterKeys = Object.keys(filter);

		if (filterKeys.length > 0) {
			return decodedLogs.filter(log => {
				if (typeof log === 'string') return true;

				return filterKeys.every((key: string) => {
					if (Array.isArray(filter[key])) {
						return (filter[key] as Numbers[]).some(
							(v: Numbers) =>
								String(log.returnValues[key]).toUpperCase() ===
								String(v).toUpperCase(),
						);
					}

					const inputAbi = abi.inputs?.filter(input => input.name === key)[0];
					if (inputAbi?.indexed && inputAbi.type === 'string') {
						const hashedIndexedString = keccak256(filter[key] as string);
						if (hashedIndexedString === String(log.returnValues[key])) return true;
					}

					return (
						String(log.returnValues[key]).toUpperCase() ===
						String(filter[key]).toUpperCase()
					);
				});
			});
		}

		return decodedLogs;
	}

	private _parseAndSetAddress(
		value?: Address,
		returnFormat: DataFormat = this.defaultReturnFormat,
	) {
		this._address = value
			? toChecksumAddress(format({ format: 'address' }, value, returnFormat))
			: value;
	}

	public decodeMethodData(data: HexString): DecodedParams & { __method__: string } {
		const methodSignature = data.slice(0, 10);
		const functionsAbis = this._jsonInterface.filter(j => j.type !== 'error');

		const abi = functionsAbis.find(
			a => methodSignature === encodeFunctionSignature(jsonInterfaceMethodToString(a)),
		);
		if (!abi) {
			throw new Web3ContractError(
				`The ABI for the provided method signature ${methodSignature} was not found.`,
			);
		}
		return decodeFunctionCall(abi, data);
	}

	private _parseAndSetJsonInterface(
		abis: ContractAbi,
		returnFormat: DataFormat = this.defaultReturnFormat,
	) {
		this._functions = {};
		this._methods = {} as ContractMethodsInterface<Abi>;
		this._events = {} as ContractEventsInterface<Abi>;

		let result: ContractAbi = [];

		const functionsAbi = abis.filter(abi => abi.type !== 'error');
		const errorsAbi = abis.filter(abi =>
			isAbiErrorFragment(abi),
		) as unknown as AbiErrorFragment[];

		for (const a of functionsAbi) {
			const abi: Mutable<AbiFragment & { signature: HexString }> = {
				...a,
				signature: '',
			};

			if (isAbiFunctionFragment(abi)) {
				const methodName = jsonInterfaceMethodToString(abi);
				const methodSignature = encodeFunctionSignature(methodName);
				abi.methodNameWithInputs = methodName;
				abi.signature = methodSignature;

				// make constant and payable backwards compatible
				abi.constant =
					abi.stateMutability === 'view' ||
					abi.stateMutability === 'pure' ||
					abi.constant;

				abi.payable = abi.stateMutability === 'payable' || abi.payable;
				this._overloadedMethodAbis.set(abi.name, [
					...(this._overloadedMethodAbis.get(abi.name) ?? []),
					abi,
				]);
				const abiFragment = this._overloadedMethodAbis.get(abi.name) ?? [];
				const contractMethod = this._createContractMethod<
					typeof abiFragment,
					AbiErrorFragment
				>(abiFragment, errorsAbi);

				const exactContractMethod = this._createContractMethod<
					typeof abiFragment,
					AbiErrorFragment
				>(abiFragment, errorsAbi, true);

				this._functions[methodName] = {
					signature: methodSignature,
					method: exactContractMethod,
				};

				// We don't know a particular type of the Abi method so can't type check
				this._methods[abi.name as keyof ContractMethodsInterface<Abi>] =
					contractMethod as never;

				// We don't know a particular type of the Abi method so can't type check
				this._methods[methodName as keyof ContractMethodsInterface<Abi>] =
					exactContractMethod as never;

				// We don't know a particular type of the Abi method so can't type check
				this._methods[methodSignature as keyof ContractMethodsInterface<Abi>] =
					exactContractMethod as never;
			} else if (isAbiEventFragment(abi)) {
				const eventName = jsonInterfaceMethodToString(abi);
				const eventSignature = encodeEventSignature(eventName);
				const event = this._createContractEvent(abi, returnFormat);
				abi.signature = eventSignature;

				if (!(eventName in this._events) || abi.name === 'bound') {
					// It's a private type and we don't want to expose it and no need to check
					this._events[eventName as keyof ContractEventsInterface<Abi>] = event as never;
				}
				// It's a private type and we don't want to expose it and no need to check
				this._events[abi.name as keyof ContractEventsInterface<Abi>] = event as never;
				// It's a private type and we don't want to expose it and no need to check
				this._events[eventSignature as keyof ContractEventsInterface<Abi>] = event as never;
			}

			result = [...result, abi];
		}

		this._events.allEvents = this._createContractEvent(ALL_EVENTS_ABI, returnFormat);
		this._jsonInterface = [...result] as unknown as ContractAbiWithSignature;
		this._errorsInterface = errorsAbi;
	}

	// eslint-disable-next-line class-methods-use-this
	private _getAbiParams(abi: AbiFunctionFragment, params: unknown[]): Array<unknown> {
		try {
			return validatorUtils.transformJsonDataToAbiFormat(abi.inputs ?? [], params);
		} catch (error) {
			throw new Web3ContractError(
				`Invalid parameters for method ${abi.name}: ${(error as Error).message}`,
			);
		}
	}

	private _createContractMethod<T extends AbiFunctionFragment[], E extends AbiErrorFragment>(
		abiArr: T,
		errorsAbis: E[],
		exact = false, // when true, it will only match the exact method signature
	): ContractBoundMethod<T[0]> {
		const abi = abiArr[abiArr.length - 1];
		return (...params: unknown[]) => {
			let abiParams!: Array<unknown>;
			const abis =
				(exact
					? this._overloadedMethodAbis
							.get(abi.name)
							?.filter(_abi => _abi.signature === abi.signature)
					: this._overloadedMethodAbis.get(abi.name)) ?? [];
			let methodAbi: AbiFunctionFragment = abis[0];
			const internalErrorsAbis = errorsAbis;

			const arrayOfAbis: AbiFunctionFragment[] = abis.filter(
				_abi => (_abi.inputs ?? []).length === params.length,
			);

			if (abis.length === 1 || arrayOfAbis.length === 0) {
				abiParams = this._getAbiParams(methodAbi, params);
				validator.validate(abi.inputs ?? [], abiParams);
			} else {
				const errors: Web3ValidationErrorObject[] = [];

				// all the methods that have is valid for the given inputs
				const applicableMethodAbi: AbiFunctionFragment[] = [];
				for (const _abi of arrayOfAbis) {
					try {
						abiParams = this._getAbiParams(_abi, params);
						validator.validate(
							_abi.inputs as unknown as ValidationSchemaInput,
							abiParams,
						);
						applicableMethodAbi.push(_abi);
					} catch (e) {
						errors.push(e as Web3ValidationErrorObject);
					}
				}
				if (applicableMethodAbi.length === 1) {
					[methodAbi] = applicableMethodAbi; // take the first item that is the only item in the array
				} else if (applicableMethodAbi.length > 1) {
					[methodAbi] = applicableMethodAbi; // take the first item in the array
					console.warn(
						`Multiple methods found that is compatible with the given inputs.\n\tFound ${
							applicableMethodAbi.length
						} compatible methods: ${JSON.stringify(
							applicableMethodAbi.map(
								m =>
									`${
										(m as { methodNameWithInputs: string }).methodNameWithInputs
									} (signature: ${(m as { signature: string }).signature})`,
							),
						)} \n\tThe first one will be used: ${
							(methodAbi as { methodNameWithInputs: string }).methodNameWithInputs
						}`,
					);
					// TODO: 5.x Should throw a new error with the list of methods found.
					// Related issue: https://github.com/web3/web3.js/issues/6923
					// This is in order to provide an error message when there is more than one method found that fits the inputs.
					// To do that, replace the pervious line of code with something like the following line:
					// throw new Web3ValidatorError({ message: 'Multiple methods found',  ... list of applicable methods }));
				}
				if (errors.length === arrayOfAbis.length) {
					throw new Web3ValidatorError(errors);
				}
			}
			const methods = {
				arguments: abiParams,

				call: async (
					options?: PayableCallOptions | NonPayableCallOptions,
					block?: BlockNumberOrTag,
				) =>
					this._contractMethodCall(
						methodAbi,
						abiParams,
						internalErrorsAbis,
						options,
						block,
					),

				send: (options?: PayableTxOptions | NonPayableTxOptions): ContractMethodSend =>
					this._contractMethodSend(methodAbi, abiParams, internalErrorsAbis, options),
				populateTransaction: (
					options?: PayableTxOptions | NonPayableTxOptions,
					contractOptions?: ContractOptions,
				) => {
					let modifiedContractOptions = contractOptions ?? this.options;
					modifiedContractOptions = {
						...modifiedContractOptions,
						input: undefined,
						from: modifiedContractOptions?.from ?? this.defaultAccount ?? undefined,
					};
					const tx = getSendTxParams({
						abi,
						params,
						options: { ...options, dataInputFill: this.config.contractDataInputFill },
						contractOptions: modifiedContractOptions,
					});
					// @ts-expect-error remove unnecessary field
					if (tx.dataInputFill) {
						// @ts-expect-error remove unnecessary field
						delete tx.dataInputFill;
					}
					return tx;
				},
				estimateGas: async <ReturnFormat extends DataFormat = typeof DEFAULT_RETURN_FORMAT>(
					options?: PayableCallOptions | NonPayableCallOptions,
					returnFormat: ReturnFormat = this
						.defaultReturnFormat as unknown as ReturnFormat,
				) =>
					this.contractMethodEstimateGas({
						abi: methodAbi,
						params: abiParams,
						returnFormat,
						options,
					}),

				encodeABI: () => encodeMethodABI(methodAbi, abiParams),
				decodeData: (data: HexString) => decodeFunctionCall(methodAbi, data),

				createAccessList: async (
					options?: PayableCallOptions | NonPayableCallOptions,
					block?: BlockNumberOrTag,
				) =>
					this._contractMethodCreateAccessList(
						methodAbi,
						abiParams,
						internalErrorsAbis,
						options,
						block,
					),
			};

			if (methodAbi.stateMutability === 'payable') {
				return methods as PayableMethodObject<
					ContractOverloadedMethodInputs<T>,
					ContractOverloadedMethodOutputs<T>
				>;
			}
			return methods as NonPayableMethodObject<
				ContractOverloadedMethodInputs<T>,
				ContractOverloadedMethodOutputs<T>
			>;
		};
	}

	private async _contractMethodCall<Options extends PayableCallOptions | NonPayableCallOptions>(
		abi: AbiFunctionFragment,
		params: unknown[],
		errorsAbi: AbiErrorFragment[],
		options?: Options,
		block?: BlockNumberOrTag,
	) {
		const tx = getEthTxCallParams({
			abi,
			params,
			options: {
				...options,
				dataInputFill: this.config.contractDataInputFill,
			},
			contractOptions: {
				...this.options,
				from: this.options.from ?? this.config.defaultAccount,
			},
		});
		try {
			const result = await call(
				this,
				tx,
				block,
				this.defaultReturnFormat as typeof DEFAULT_RETURN_FORMAT,
			);
			return decodeFunctionReturn(abi, result);
		} catch (error: unknown) {
			if (error instanceof ContractExecutionError) {
				// this will parse the error data by trying to decode the ABI error inputs according to EIP-838
				decodeContractErrorData(errorsAbi, error.cause);
			}
			throw error;
		}
	}

	private async _contractMethodCreateAccessList<
		Options extends PayableCallOptions | NonPayableCallOptions,
	>(
		abi: AbiFunctionFragment,
		params: unknown[],
		errorsAbi: AbiErrorFragment[],
		options?: Options,
		block?: BlockNumberOrTag,
	) {
		const tx = getCreateAccessListParams({
			abi,
			params,
			options: { ...options, dataInputFill: this.config.contractDataInputFill },
			contractOptions: {
				...this.options,
				from: this.options.from ?? this.config.defaultAccount,
			},
		});

		try {
			return createAccessList(this, tx, block, this.defaultReturnFormat);
		} catch (error: unknown) {
			if (error instanceof ContractExecutionError) {
				// this will parse the error data by trying to decode the ABI error inputs according to EIP-838
				decodeContractErrorData(errorsAbi, error.cause);
			}
			throw error;
		}
	}

	private _contractMethodSend<Options extends PayableCallOptions | NonPayableCallOptions>(
		abi: AbiFunctionFragment,
		params: unknown[],
		errorsAbi: AbiErrorFragment[],
		options?: Options,
		contractOptions?: ContractOptions,
	) {
		let modifiedContractOptions = contractOptions ?? this.options;
		modifiedContractOptions = {
			...modifiedContractOptions,
			input: undefined,
			from: modifiedContractOptions.from ?? this.defaultAccount ?? undefined,
		};
		const tx = getSendTxParams({
			abi,
			params,
			options: { ...options, dataInputFill: this.config.contractDataInputFill },
			contractOptions: modifiedContractOptions,
		});

		const transactionToSend = isNullish(this.transactionMiddleware)
			? sendTransaction(this, tx, this.defaultReturnFormat, {
					// TODO Should make this configurable by the user
					checkRevertBeforeSending: false,
					contractAbi: this._jsonInterface, // explicitly not passing middleware so if some one is using old eth package it will not break
			  })
			: sendTransaction(
					this,
					tx,
					this.defaultReturnFormat,
					{
						// TODO Should make this configurable by the user
						checkRevertBeforeSending: false,
						contractAbi: this._jsonInterface,
					},
					this.transactionMiddleware,
			  );

		// eslint-disable-next-line no-void
		void transactionToSend.on('error', (error: unknown) => {
			if (error instanceof ContractExecutionError) {
				// this will parse the error data by trying to decode the ABI error inputs according to EIP-838
				decodeContractErrorData(errorsAbi, error.cause);
			}
		});
		return transactionToSend;
	}

	public async contractMethodEstimateGas<
		Options extends PayableCallOptions | NonPayableCallOptions,
		ReturnFormat extends DataFormat,
	>({
		abi,
		params,
		returnFormat,
		options,
		contractOptions,
	}: {
		abi: AbiFunctionFragment;
		params: unknown[];
		returnFormat: ReturnFormat;
		options?: Options;
		contractOptions?: ContractOptions;
	}) {
		const tx = getEstimateGasParams({
			abi,
			params,
			options: { ...options, dataInputFill: this.config.contractDataInputFill },
			contractOptions: contractOptions ?? this.options,
		});
		return estimateGas(this, tx, BlockTags.LATEST, returnFormat ?? this.defaultReturnFormat);
	}

	// eslint-disable-next-line class-methods-use-this
	private _createContractEvent(
		abi: AbiEventFragment & { signature: HexString },
		returnFormat: DataFormat = this.defaultReturnFormat,
	): ContractBoundEvent {
		return (...params: unknown[]) => {
			const { topics, fromBlock } = encodeEventABI(
				this.options,
				abi,
				params[0] as EventParameters,
			);
			const sub = new ContractLogsSubscription(
				{
					address: this.options.address,
					topics,
					abi,
					jsonInterface: this._jsonInterface,
				},
				{
					subscriptionManager: this.subscriptionManager as Web3SubscriptionManager<
						unknown,
						{
							[key: string]: Web3SubscriptionConstructor<unknown>;
						}
					>,
					returnFormat,
				},
			);
			if (!isNullish(fromBlock)) {
				// emit past events when fromBlock is defined
				this.getPastEvents(abi.name, { fromBlock, topics }, returnFormat)
					.then(logs => {
						if (logs) {
							logs.forEach(log => sub.emit('data', log as EventLog));
						}
					})
					.catch((error: Error) => {
						sub.emit(
							'error',
							new SubscriptionError('Failed to get past events.', error),
						);
					});
			}
			this.subscriptionManager?.addSubscription(sub).catch((error: Error) => {
				sub.emit('error', new SubscriptionError('Failed to subscribe.', error));
			});

			return sub;
		};
	}

	protected subscribeToContextEvents<T extends Web3Context>(context: T): void {
		// eslint-disable-next-line @typescript-eslint/no-this-alias
		const contractThis = this;
		this.context = context;

		if (contractThis.syncWithContext) {
			context.on(Web3ConfigEvent.CONFIG_CHANGE, event => {
				contractThis.setConfig({ [event.name]: event.newValue });
			});
		}
	}
}
