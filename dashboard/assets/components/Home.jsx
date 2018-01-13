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
import {LineChart, AreaChart, Area, YAxis, CartesianGrid, Line} from 'recharts';

import ChartGrid from './ChartGrid';
import type {ChartEntry} from '../types/message';

export type Props = {
    theme: Object,
    memory: Array<ChartEntry>,
    traffic: Array<ChartEntry>,
    shouldUpdate: Object,
};
// Home renders the home content.
class Home extends Component<Props> {
	constructor(props: Props) {
		super(props);
		const {theme} = props; // The theme property is injected by withTheme().
		this.memoryColor = theme.palette.primary[300];
		this.trafficColor = theme.palette.secondary[300];
	}

	shouldComponentUpdate(nextProps) {
		return nextProps.shouldUpdate.has('memory') || nextProps.shouldUpdate.has('traffic');
	}

	render() {
		const {memory, traffic} = this.props;

		return (
			<ChartGrid spacing={24}>
				<AreaChart xs={6} height={300} values={memory}>
					<YAxis />
					<Area type="monotone" dataKey="value" stroke={this.memoryColor} fill={this.memoryColor} />
				</AreaChart>
				<LineChart xs={6} height={300} values={traffic}>
					<Line type="monotone" dataKey="value" stroke={this.trafficColor} dot={false} />
				</LineChart>
				<LineChart xs={6} height={300} values={memory}>
					<YAxis />
					<CartesianGrid stroke="#eee" strokeDasharray="5 5" />
					<Line type="monotone" dataKey="value" stroke={this.memoryColor} dot={false} />
				</LineChart>
				<AreaChart xs={6} height={300} values={traffic}>
					<CartesianGrid stroke="#eee" strokeDasharray="5 5" vertical={false} />
					<Area type="monotone" dataKey="value" stroke={this.trafficColor} fill={this.trafficColor} />
				</AreaChart>
			</ChartGrid>
		);
	}
}

export default withTheme()(Home);
