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

import { format } from 'web3-utils';

import {
	AbiEventFragment,
	LogsInput,
	DataFormat,
	DEFAULT_RETURN_FORMAT,
	EventLog,
	ContractAbiWithSignature,
} from 'web3-types';

import { decodeLog } from 'web3-eth-abi';

import { logSchema } from '../schemas.js';
import { ALL_EVENTS } from '../constants.js';

export const decodeEventABI = (
	event: AbiEventFragment & { signature: string },
	data: LogsInput,
	jsonInterface: ContractAbiWithSignature,
	returnFormat: DataFormat = DEFAULT_RETURN_FORMAT,
): EventLog => {
	let modifiedEvent = { ...event };

	const result = format(logSchema, data, returnFormat);

	// if allEvents get the right event
	if ([ALL_EVENTS, 'allEvents'].includes(modifiedEvent.name)) {
		const matchedEvent = jsonInterface.find(j => j.signature === data.topics[0]);
		if (matchedEvent) {
			modifiedEvent = matchedEvent as AbiEventFragment & { signature: string };
		} else {
			modifiedEvent = { anonymous: true } as unknown as AbiEventFragment & {
				signature: string;
			};
		}
	}

	// create empty inputs if none are present (e.g. anonymous events on allEvents)
	modifiedEvent.inputs = modifiedEvent.inputs ?? event.inputs ?? [];

	// Handle case where an event signature shadows the current ABI with non-identical
	// arg indexing. If # of topics doesn't match, event is anon.
	if (!modifiedEvent.anonymous) {
		let indexedInputs = 0;
		(modifiedEvent.inputs ?? []).forEach(input => {
			if (input.indexed) {
				indexedInputs += 1;
			}
		});

		if (indexedInputs > 0 && data?.topics && data?.topics.length !== indexedInputs + 1) {
			// checks if event is anonymous
			modifiedEvent = {
				...modifiedEvent,
				anonymous: true,
				inputs: [],
			};
		}
	}

	const argTopics = modifiedEvent.anonymous ? data.topics : (data.topics ?? []).slice(1);

	return {
		...result,
		returnValues: decodeLog([...(modifiedEvent.inputs ?? [])], data.data, argTopics),
		event: modifiedEvent.name,
		signature:
			!modifiedEvent.anonymous && data.topics?.length > 0 && data.topics[0]
				? data.topics[0]
				: undefined,

		raw: {
			data: data.data,
			topics: data.topics,
		},
	};
};
