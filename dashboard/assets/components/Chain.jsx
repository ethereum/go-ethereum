// @flow

// Copyright 2019 The go-ethereum Authors
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
import type {Chain as ChainType} from '../types/content';

export const inserter = () => (update: ChainType, prev: ChainType) => {
	if (!update.currentBlock) {
		return;
	}
	if (!prev.currentBlock) {
		prev.currentBlock = {};
	}
	prev.currentBlock.number = update.currentBlock.number;
	prev.currentBlock.timestamp = update.currentBlock.timestamp;
	return prev;
};

// styles contains the constant styles of the component.
const styles = {};

// themeStyles returns the styles generated from the theme for the component.
const themeStyles = theme => ({});

export type Props = {
	content: Content,
};

type State = {};

// Logs renders the log page.
class Chain extends Component<Props, State> {
	render() {
		return <></>;
	}
}

export default Chain;
