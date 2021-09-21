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

// percentPlotter renders a tooltip, which displays the value of the payload followed by a percent sign.
export const percentPlotter = <T>(text: string, mapper: (T => T) = multiplier(1)) => (payload: T) => {
	const p = mapper(payload);
	if (typeof p !== 'number') {
		return null;
	}
	return (
		<Typography type='caption' color='inherit'>
			<span style={styles.light}>{text}</span> {p.toFixed(2)} %
		</Typography>
	);
};

// unit contains the units for the bytePlotter.
const unit = ['', 'Ki', 'Mi', 'Gi', 'Ti', 'Pi', 'Ei', 'Zi', 'Yi'];

// simplifyBytes returns the simplified version of the given value followed by the unit.
const simplifyBytes = (x: number) => {
	let i = 0;
	for (; x > 1024 && i < 8; i++) {
		x /= 1024;
	}
	return x.toFixed(2).toString().concat(' ', unit[i], 'B');
};

// bytePlotter renders a tooltip, which displays the payload as a byte value.
export const bytePlotter = <T>(text: string, mapper: (T => T) = multiplier(1)) => (payload: T) => {
	const p = mapper(payload);
	if (typeof p !== 'number') {
		return null;
	}
	return (
		<Typography type='caption' color='inherit'>
			<span style={styles.light}>{text}</span> {simplifyBytes(p)}
		</Typography>
	);
};

// bytePlotter renders a tooltip, which displays the payload as a byte value followed by '/s'.
export const bytePerSecPlotter = <T>(text: string, mapper: (T => T) = multiplier(1)) => (payload: T) => {
	const p = mapper(payload);
	if (typeof p !== 'number') {
		return null;
	}
	return (
		<Typography type='caption' color='inherit'>
			<span style={styles.light}>{text}</span> {simplifyBytes(p)}/s
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
