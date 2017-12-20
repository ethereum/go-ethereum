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
import type {Node} from 'react';

import Grid from 'material-ui/Grid';
import {ResponsiveContainer} from 'recharts';

export type Props = {
	spacing: number,
	children: Node,
};
// ChartGrid renders a grid container for responsive charts.
// The children are Recharts components extended with the Material-UI's xs property.
class ChartGrid extends Component<Props> {
	render() {
		return (
			<Grid container spacing={this.props.spacing}>
				{
					React.Children.map(this.props.children, child => (
						<Grid item xs={child.props.xs}>
							<ResponsiveContainer width="100%" height={child.props.height}>
								{React.cloneElement(child, {data: child.props.values.map(value => ({value}))})}
							</ResponsiveContainer>
						</Grid>
					))
				}
			</Grid>
		);
	}
}

export default ChartGrid;
