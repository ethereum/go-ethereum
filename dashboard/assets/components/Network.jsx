// @flow

// Copyright 2018 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

import React, {Component} from 'react';

import type {Content, Network as NetworkType} from '../types/content';


// inserter is a state updater function for the main component, which inserts the new log chunk into the chunk array.
// limit is the maximum length of the chunk array, used in order to prevent the browser from OOM.
export const inserter = (update: NetworkType, prev: LogsType) => prev;

// styles contains the constant styles of the component.
const styles = {};

export type Props = {
    container:    Object,
    content:      Content,
    shouldUpdate: Object,
};

type State = {
    peers: List<string>,
};

// Network renders the network page.
class Network extends Component<Props, State> {
	constructor(props: Props) {
		super(props);
		this.content = React.createRef();
		this.state = {
			peers: [],
		};
	}

	render() {
		return (
			<div>
				Alma
			</div>
		);
	}
}

export default Network;
