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
import type {ChildrenArray} from 'react';

import Grid from 'material-ui/Grid';

// styles contains the constant styles of the component.
const styles = {
	container: {
		flexWrap: 'nowrap',
		height:   '100%',
		maxWidth: '100%',
		margin:   0,
	},
	item: {
		flex:    1,
		padding: 0,
	},
}

export type Props = {
	children: ChildrenArray<React$Element<any>>,
};

// ChartRow renders a row of equally sized responsive charts.
class ChartRow extends Component<Props> {
	render() {
		return (
			<Grid container direction='row' style={styles.container} justify='space-between'>
				{React.Children.map(this.props.children, child => (
					<Grid item xs style={styles.item}>
						{child}
					</Grid>
				))}
			</Grid>
		);
	}
}

export default ChartRow;
