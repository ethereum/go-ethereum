// @flow

// Copyright 2017 The go-ethereum Authors
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

import withTheme from 'material-ui/styles/withTheme';
import {LineChart, AreaChart, Area, Line} from 'recharts';

import ChartGrid from './ChartGrid';
import type {ChartEntry} from '../types/content';

export type Props = {
    theme: Object,
    activeMemory: Array<ChartEntry>,
    ingress: Array<ChartEntry>,
    egress: Array<ChartEntry>,
    cpu: Array<ChartEntry>,
	shouldUpdate: Object,
};

// Home renders the home content.
class Home extends Component<Props> {
	shouldComponentUpdate(nextProps) {
		return typeof nextProps.shouldUpdate.home !== 'undefined';
	}

	render() {
		let {
			activeMemory, ingress, egress, cpu
		} = this.props;
		activeMemory = activeMemory.map(({value}) => (value || 0));
		cpu = cpu.map(({value}) => (value || 0));
		ingress = ingress.map(({value}) => (value || 0));
		egress = egress.map(({value}) => (value || 0));

		const color = '#8884d8';

		return (
			<ChartGrid spacing={24}>
				<AreaChart xs={6} height={300} values={activeMemory}>
					<Area type='monotone' dataKey='value' stroke={color} fill={color} />
				</AreaChart>
				<LineChart xs={6} height={300} values={ingress}>
					<Line type='monotone' dataKey='value' stroke={color} dot={false} />
				</LineChart>
				<LineChart xs={6} height={300} values={cpu}>
					<Line type='monotone' dataKey='value' stroke={color} dot={false} />
				</LineChart>
				<AreaChart xs={6} height={300} values={egress}>
					<Area type='monotone' dataKey='value' stroke={color} fill={color} />
				</AreaChart>
			</ChartGrid>
		);
	}
}

export default withTheme()(Home);
