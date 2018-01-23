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

import withStyles from 'material-ui/styles/withStyles';
import Typography from 'material-ui/Typography';
import Grid from 'material-ui/Grid';
import {ResponsiveContainer, AreaChart, Area, Tooltip} from 'recharts';

import ChartRow from './ChartRow';
import CustomTooltip, {bytePlotter, bytePerSecPlotter, percentPlotter, multiplier} from './CustomTooltip';
import {styles as commonStyles} from '../common';
import type {Content} from '../types/content';

// styles contains the constant styles of the component.
const styles = {
	footer: {
		maxWidth: '100%',
		flexWrap: 'nowrap',
		margin:   0,
	},
	chartRowWrapper: {
		height:  '100%',
		padding: 0,
	},
	doubleChartWrapper: {
		height:     '100%',
		width:      '99%',
		paddingTop: 5,
	},
};

// themeStyles returns the styles generated from the theme for the component.
const themeStyles: Object = (theme: Object) => ({
	footer: {
		backgroundColor: theme.palette.background.appBar,
		color:           theme.palette.getContrastText(theme.palette.background.appBar),
		zIndex:          theme.zIndex.appBar,
		height:          theme.spacing.unit * 10,
	},
});

export type Props = {
	classes: Object, // injected by withStyles()
	theme: Object,
	content: Content,
	shouldUpdate: Object,
};

// Footer renders the footer of the dashboard.
class Footer extends Component<Props> {
	shouldComponentUpdate(nextProps) {
		return typeof nextProps.shouldUpdate.home !== 'undefined';
	}

	// info renders a label with the given values.
	info = (about: string, value: ?string) => (value ? (
		<Typography type='caption' color='inherit'>
			<span style={commonStyles.light}>{about}</span> {value}
		</Typography>
	) : null);

	// doubleChart renders a pair of charts separated by the baseline.
	doubleChart = (syncId, topChart, bottomChart) => {
		const topKey = 'topKey';
		const bottomKey = 'bottomKey';
		const topDefault = topChart.default ? topChart.default : 0;
		const bottomDefault = bottomChart.default ? bottomChart.default : 0;
		const topTooltip = topChart.tooltip ? (
			<Tooltip cursor={false} content={<CustomTooltip tooltip={topChart.tooltip} />} />
		) : null;
		const bottomTooltip = bottomChart.tooltip ? (
			<Tooltip cursor={false} content={<CustomTooltip tooltip={bottomChart.tooltip} />} />
		) : null;
		const topColor = '#8884d8';
		const bottomColor = '#82ca9d';

		// Put the samples of the two charts into the same array in order to avoid problems
		// at the synchronized area charts. If one of the two arrays doesn't have value at
		// a given position, give it a 0 default value.
		let data = [...topChart.data.map(({value}) => {
			const d = {};
			d[topKey] = value || topDefault;
			return d;
		})];
		for (let i = 0; i < data.length && i < bottomChart.data.length; i++) {
			// The value needs to be negative in order to plot it upside down.
			const d = bottomChart.data[i];
			data[i][bottomKey] = d && d.value ? -d.value : bottomDefault;
		}
		data = [...data, ...bottomChart.data.slice(data.length).map(({value}) => {
			const d = {};
			d[topKey] = topDefault;
			d[bottomKey] = -value || bottomDefault;
			return d;
		})];

		return (
			<div style={styles.doubleChartWrapper}>
				<ResponsiveContainer width='100%' height='50%'>
					<AreaChart data={data} syncId={syncId} >
						{topTooltip}
						<Area type='monotone' dataKey={topKey} stroke={topColor} fill={topColor} />
					</AreaChart>
				</ResponsiveContainer>
				<div style={{marginTop: -10, width: '100%', height: '50%'}}>
					<ResponsiveContainer width='100%' height='100%'>
						<AreaChart data={data} syncId={syncId} >
							{bottomTooltip}
							<Area type='monotone' dataKey={bottomKey} stroke={bottomColor} fill={bottomColor} />
						</AreaChart>
					</ResponsiveContainer>
				</div>
			</div>
		);
	}

	render() {
		const {content} = this.props;
		const {general, home} = content;

		return (
			<Grid container className={this.props.classes.footer} direction='row' alignItems='center' style={styles.footer}>
				<Grid item xs style={styles.chartRowWrapper}>
					<ChartRow>
						{this.doubleChart(
							'all',
							{data: home.processCPU, tooltip: percentPlotter('Process')},
							{data: home.systemCPU, tooltip: percentPlotter('System', multiplier(-1))},
						)}
						{this.doubleChart(
							'all',
							{data: home.activeMemory, tooltip: bytePlotter('Active')},
							{data: home.virtualMemory, tooltip: bytePlotter('Virtual', multiplier(-1))},
						)}
						{this.doubleChart(
							'all',
							{data: home.diskRead, tooltip: bytePerSecPlotter('Disk Read')},
							{data: home.diskWrite, tooltip: bytePerSecPlotter('Disk Write', multiplier(-1))},
						)}
						{this.doubleChart(
							'all',
							{data: home.networkIngress, tooltip: bytePerSecPlotter('Download')},
							{data: home.networkEgress, tooltip: bytePerSecPlotter('Upload', multiplier(-1))},
						)}
					</ChartRow>
				</Grid>
				<Grid item >
					{this.info('Geth', general.version)}
					{this.info('Commit', general.commit ? general.commit.substring(0, 7) : null)}
				</Grid>
			</Grid>
		);
	}
}

export default withStyles(themeStyles)(Footer);
