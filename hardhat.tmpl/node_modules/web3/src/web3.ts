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
// eslint-disable-next-line max-classes-per-file
import {
	Web3Context,
	Web3ContextInitOptions,
	Web3ContextObject,
	Web3SubscriptionConstructor,
	isSupportedProvider,
} from 'web3-core';
import { Web3Eth, RegisteredSubscription, registeredSubscriptions } from 'web3-eth';
import Contract from 'web3-eth-contract';
import { ENS, registryAddresses } from 'web3-eth-ens';
import { Iban } from 'web3-eth-iban';
import { Personal } from 'web3-eth-personal';
import { Net } from 'web3-net';
import * as utils from 'web3-utils';
import { isNullish, isDataFormat, isContractInitOptions } from 'web3-utils';
import { mainnet } from 'web3-rpc-providers';
import {
	Address,
	ContractAbi,
	ContractInitOptions,
	EthExecutionAPI,
	SupportedProviders,
	DataFormat,
} from 'web3-types';
import { InvalidMethodParamsError } from 'web3-errors';
import abi from './abi.js';
import { initAccountsForContext } from './accounts.js';
import { Web3EthInterface } from './types.js';
import { Web3PkgInfo } from './version.js';
import { onNewProviderDiscovered, requestEIP6963Providers } from './web3_eip6963.js';

export class Web3<
	CustomRegisteredSubscription extends {
		[key: string]: Web3SubscriptionConstructor<EthExecutionAPI>;
	} = RegisteredSubscription,
> extends Web3Context<EthExecutionAPI, CustomRegisteredSubscription & RegisteredSubscription> {
	public static version = Web3PkgInfo.version;
	public static utils = utils;
	public static requestEIP6963Providers = requestEIP6963Providers;
	public static onNewProviderDiscovered = onNewProviderDiscovered;
	public static modules = {
		Web3Eth,
		Iban,
		Net,
		ENS,
		Personal,
	};

	public utils: typeof utils;

	public eth: Web3EthInterface;

	public constructor(
		providerOrContext:
			| string
			| SupportedProviders<EthExecutionAPI>
			| Web3ContextInitOptions<EthExecutionAPI, CustomRegisteredSubscription> = mainnet,
	) {
		if (
			isNullish(providerOrContext) ||
			(typeof providerOrContext === 'string' && providerOrContext.trim() === '') ||
			(typeof providerOrContext !== 'string' &&
				!isSupportedProvider(providerOrContext as SupportedProviders<EthExecutionAPI>) &&
				!(providerOrContext as Web3ContextInitOptions).provider)
		) {
			console.warn(
				'NOTE: web3.js is running without provider. You need to pass a provider in order to interact with the network!',
			);
		}

		let contextInitOptions: Web3ContextInitOptions<EthExecutionAPI> = {};
		if (
			typeof providerOrContext === 'string' ||
			isSupportedProvider(providerOrContext as SupportedProviders)
		) {
			contextInitOptions.provider = providerOrContext as
				| undefined
				| string
				| SupportedProviders;
		} else if (providerOrContext) {
			contextInitOptions = providerOrContext as Web3ContextInitOptions;
		} else {
			contextInitOptions = {};
		}

		contextInitOptions.registeredSubscriptions = {
			// all the Eth standard subscriptions
			...registeredSubscriptions,
			// overridden and combined with any custom subscriptions
			...(contextInitOptions.registeredSubscriptions ?? {}),
		} as CustomRegisteredSubscription;

		super(contextInitOptions);
		const accounts = initAccountsForContext(this);

		// Init protected properties
		this._wallet = accounts.wallet;
		this._accountProvider = accounts;

		this.utils = utils;

		// Have to use local alias to initiate contract context
		// eslint-disable-next-line @typescript-eslint/no-this-alias
		const self = this;

		class ContractBuilder<Abi extends ContractAbi> extends Contract<Abi> {
			public constructor(jsonInterface: Abi);
			public constructor(
				jsonInterface: Abi,
				addressOrOptionsOrContext?: Address | ContractInitOptions | Web3Context,
			);
			public constructor(
				jsonInterface: Abi,
				addressOrOptionsOrContext?: Address | ContractInitOptions | Web3Context,
				optionsOrContextOrReturnFormat?: ContractInitOptions | Web3Context | DataFormat,
			);
			public constructor(
				jsonInterface: Abi,
				addressOrOptionsOrContext?: Address | ContractInitOptions,
				optionsOrContextOrReturnFormat?: ContractInitOptions,
				contextOrReturnFormat?: Web3Context | DataFormat,
			);
			public constructor(
				jsonInterface: Abi,
				addressOrOptionsOrContext?: Address | ContractInitOptions,
				optionsOrContextOrReturnFormat?: ContractInitOptions,
				contextOrReturnFormat?: Web3Context | DataFormat,
				returnFormat?: DataFormat,
			) {
				if (
					isContractInitOptions(addressOrOptionsOrContext) &&
					isContractInitOptions(optionsOrContextOrReturnFormat)
				) {
					throw new InvalidMethodParamsError(
						'Should not provide options at both 2nd and 3rd parameters',
					);
				}
				let address: string | undefined;
				let options: object = {};
				let context: Web3ContextObject;
				let dataFormat: DataFormat | undefined;

				// add validation so its not a breaking change
				if (
					!isNullish(addressOrOptionsOrContext) &&
					typeof addressOrOptionsOrContext !== 'object' &&
					typeof addressOrOptionsOrContext !== 'string'
				) {
					throw new InvalidMethodParamsError();
				}

				if (typeof addressOrOptionsOrContext === 'string') {
					address = addressOrOptionsOrContext;
				}
				if (isContractInitOptions(addressOrOptionsOrContext)) {
					options = addressOrOptionsOrContext as object;
				} else if (isContractInitOptions(optionsOrContextOrReturnFormat)) {
					options = optionsOrContextOrReturnFormat as object;
				} else {
					options = {};
				}

				if (addressOrOptionsOrContext instanceof Web3Context) {
					context = addressOrOptionsOrContext;
				} else if (optionsOrContextOrReturnFormat instanceof Web3Context) {
					context = optionsOrContextOrReturnFormat;
				} else if (contextOrReturnFormat instanceof Web3Context) {
					context = contextOrReturnFormat;
				} else {
					context = self.getContextObject() as Web3ContextObject;
				}

				if (returnFormat) {
					dataFormat = returnFormat;
				} else if (isDataFormat(optionsOrContextOrReturnFormat)) {
					dataFormat = optionsOrContextOrReturnFormat as DataFormat;
				} else if (isDataFormat(contextOrReturnFormat)) {
					dataFormat = contextOrReturnFormat;
				}

				super(jsonInterface, address, options, context, dataFormat);
				super.subscribeToContextEvents(self);

				// eslint-disable-next-line no-use-before-define
				if (!isNullish(eth)) {
					// eslint-disable-next-line no-use-before-define
					const TxMiddleware = eth.getTransactionMiddleware();
					if (!isNullish(TxMiddleware)) {
						super.setTransactionMiddleware(TxMiddleware);
					}
				}
			}
		}

		const eth = self.use(Web3Eth);

		// Eth Module
		this.eth = Object.assign(eth, {
			// ENS module
			ens: self.use(ENS, registryAddresses.main), // registry address defaults to main network

			// Iban helpers
			Iban,

			net: self.use(Net),
			personal: self.use(Personal),

			// Contract helper and module
			Contract: ContractBuilder,

			// ABI Helpers
			abi,

			// Accounts helper
			accounts,
		});
	}
}
export default Web3;
