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

import Typography from 'material-ui/Typography';
import {styles} from '../common';

// multiplier multiplies a number by another.
export const multiplier = <T>(by: number = 1) => (x: number) => x * by;

// valuePlotter renders a tooltip, which shows the value of the payload.
export const valuePlotter = <T>(text: string, mapper: (T => T) = multiplier(1)) => (payload: T) => {
	const p = mapper(payload);
	return (
		<Typography type='caption' color='inherit'>
			<span style={styles.light}>{text}</span> {p}
		</Typography>
	);
};

// unit contains the units for the bytePlotter.
const unit = ['B', 'kB', 'MB', 'GB', 'TB', 'PB'];
// bytePlotter renders a tooltip, which shows the simplified byte value of the payload followed by the unit.
export const bytePlotter = <T>(text: string, mapper: (T => T) = multiplier(1)) => (payload: T) => {
	let p = mapper(payload);
	if (typeof p !== 'number') {
		return null;
	}

	let i = 0;
	for (; p > 1024 && i < 5; i++) {
		p /= 1024;
	}

	return (
		<Typography type='caption' color='inherit'>
			<span style={styles.light}>{text}</span> {p.toFixed(2)} {unit[i]}
		</Typography>
	);
};

export type Props = {
	active: boolean,
	payload: Object,
	tooltip: <T>(text: string, mapper?: T => T) => (payload: mixed) => null | React$Element<any>,
};

// CustomTooltip takes a tooltip function, and uses it to plot the active value of the chart.
class CustomTooltip extends Component<Props> {
	render() {
		const {active, payload, tooltip} = this.props;
		if (!active || typeof tooltip !== 'function') {
			return null;
		}
		return tooltip(payload[0].value);
	}
}

export default CustomTooltip;
