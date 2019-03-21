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

import Table from '@material-ui/core/Table';
import TableHead from '@material-ui/core/TableHead';
import TableBody from '@material-ui/core/TableBody';
import TableRow from '@material-ui/core/TableRow';
import TableCell from '@material-ui/core/TableCell';
import Grid from '@material-ui/core/Grid/Grid';
import Typography from '@material-ui/core/Typography';
import {AreaChart, Area, Tooltip, YAxis} from 'recharts';
import {FontAwesomeIcon} from '@fortawesome/react-fontawesome';
import {faCircle as fasCircle} from '@fortawesome/free-solid-svg-icons';
import {faCircle as farCircle} from '@fortawesome/free-regular-svg-icons';
import convert from 'color-convert';

import CustomTooltip, {bytePlotter, multiplier} from 'CustomTooltip';
import type {Network as NetworkType, PeerEvent} from '../types/content';
import {styles as commonStyles, chartStrokeWidth, hues, hueScale} from '../common';

// Peer chart dimensions.
const trafficChartHeight = 15;
const trafficChartWidth  = 300;

// attemptSeparator separates the peer connection attempts
// such as the peers from the addresses with more attempts
// go to the beginning of the table, and the rest go to the end.
const attemptSeparator = 9;

// setMaxIngress adjusts the peer chart's gradient values based on the given value.
const setMaxIngress = (peer, value) => {
	peer.maxIngress = value;
	peer.ingressGradient = [];
	peer.ingressGradient.push({offset: hueScale[0], color: hues[0]});
	let i = 1;
	for (; i < hues.length && value > hueScale[i]; i++) {
		peer.ingressGradient.push({offset: Math.floor(hueScale[i] * 100 / value), color: hues[i]});
	}
	i--;
	if (i < hues.length - 1) {
		// Usually the maximum value gets between two points on the predefined
		// color scale (e.g. 123KB is somewhere between 100KB (#FFFF00) and
		// 1MB (#FF0000)), and the charts need to be comparable by the colors,
		// so we have to calculate the last hue using the maximum value and the
		// surrounding hues in order to avoid the uniformity of the top colors
		// on the charts. For this reason the two hues are translated into the
		// CIELAB color space, and the top color will be their weighted average
		// (CIELAB is perceptually uniform, meaning that any point on the line
		// between two pure color points is also a pure color, so the weighted
		// average will not lose from the saturation).
		//
		// In case the maximum value is greater than the biggest predefined
		// scale value, the top of the chart will have uniform color.
		const lastHue = convert.hex.lab(hues[i]);
		const proportion = (value - hueScale[i]) * 100 / (hueScale[i + 1] - hueScale[i]);
		convert.hex.lab(hues[i + 1]).forEach((val, j) => {
			lastHue[j] = (lastHue[j] * proportion + val * (100 - proportion)) / 100;
		});
		peer.ingressGradient.push({offset: 100, color: `#${convert.lab.hex(lastHue)}`});
	}
};

// setMaxEgress adjusts the peer chart's gradient values based on the given value.
// In case of the egress the chart is upside down, so the gradients need to be
// calculated inversely compared to the ingress.
const setMaxEgress = (peer, value) => {
	peer.maxEgress = value;
	peer.egressGradient = [];
	peer.egressGradient.push({offset: 100 - hueScale[0], color: hues[0]});
	let i = 1;
	for (; i < hues.length && value > hueScale[i]; i++) {
		peer.egressGradient.unshift({offset: 100 - Math.floor(hueScale[i] * 100 / value), color: hues[i]});
	}
	i--;
	if (i < hues.length - 1) {
		// Calculate the last hue.
		const lastHue = convert.hex.lab(hues[i]);
		const proportion = (value - hueScale[i]) * 100 / (hueScale[i + 1] - hueScale[i]);
		convert.hex.lab(hues[i + 1]).forEach((val, j) => {
			lastHue[j] = (lastHue[j] * proportion + val * (100 - proportion)) / 100;
		});
		peer.egressGradient.unshift({offset: 0, color: `#${convert.lab.hex(lastHue)}`});
	}
};


// setIngressChartAttributes searches for the maximum value of the ingress
// samples, and adjusts the peer chart's gradient values accordingly.
const setIngressChartAttributes = (peer) => {
	let max = 0;
	peer.ingress.forEach(({value}) => {
		if (value > max) {
			max = value;
		}
	});
	setMaxIngress(peer, max);
};

// setEgressChartAttributes searches for the maximum value of the egress
// samples, and adjusts the peer chart's gradient values accordingly.
const setEgressChartAttributes = (peer) => {
	let max = 0;
	peer.egress.forEach(({value}) => {
		if (value > max) {
			max = value;
		}
	});
	setMaxEgress(peer, max);
};

// shortName adds some heuristics to the node name in order to make it look meaningful.
const shortName = (name: string) => {
	const parts = name.split('/');
	if (parts[0].substring(0, 'parity'.length).toLowerCase() === 'parity') {
		// Merge Parity and Parity-Ethereum under the same name.
		parts[0] = 'Parity';
	}
	// Cutting anything from the version after the first - or +.
	parts[1] = parts[1].split('-')[0].split('+')[0];
	return `${parts[0]}/${parts[1]}`;
};

// shortLocation returns a shortened version of the given location object.
const shortLocation = (location: Object) => {
	if (!location) {
		return '';
	}
	return `${location.city ? `${location.city}/` : ''}${location.country ? location.country : ''}`;
};

// ethProtocol returns a shortened version of the eth protocol values.
const ethProtocol = (protocols: Object) => {
	const {eth} = protocols;
	if (typeof eth === 'string') {
		return eth;
	}
	if (!(eth instanceof Object)) {
		console.error('Wrong protocol type', eth, typeof eth);
		return '';
	}
	if (!eth.hasOwnProperty('version') || !eth.hasOwnProperty('difficulty') || !eth.hasOwnProperty('head')) {
		console.error('Missing protocol attributes', eth);
		return '';
	}
	return `h=${eth.head.substring(0, 10)} v=${eth.version} td=${eth.difficulty}`;
};

// inserter is a state updater function for the main component, which handles the peers.
export const inserter = (sampleLimit: number) => (update: NetworkType, prev: NetworkType) => {
	// The first message contains the metered peer history.
	if (update.peers && update.peers.bundles) {
		prev.peers = update.peers;
		Object.values(prev.peers.bundles).forEach((bundle) => {
			if (bundle.knownPeers) {
				Object.values(bundle.knownPeers).forEach((peer) => {
					if (!peer.maxIngress) {
						setIngressChartAttributes(peer);
					}
					if (!peer.maxEgress) {
						setEgressChartAttributes(peer);
					}
					if (!peer.name) {
						peer.name = '';
						peer.shortName = '';
					} else if (!peer.shortName) {
						peer.shortName = shortName(peer.name);
					}
					if (!peer.enode) {
						peer.enode = '';
					}
					if (!peer.protocols) {
						peer.protocols = {};
					}
					peer.eth = ethProtocol(peer.protocols);
				});
			}
			bundle.shortLocation = shortLocation(bundle.location);
		});
	}
	if (Array.isArray(update.diff)) {
		update.diff.forEach((event: PeerEvent) => {
			if (!event.addr) {
				console.error('Peer event without TCP address', event);
				return;
			}
			switch (event.remove) {
			case 'bundle': {
				delete prev.peers.bundles[event.addr];
				return;
			}
			case 'known': {
				if (!event.enode) {
					console.error('Remove known peer event without node URL', event.addr);
					return;
				}
				const bundle = prev.peers.bundles[event.addr];
				if (!bundle || !bundle.knownPeers || !bundle.knownPeers[event.enode]) {
					console.error('No known peer to remove', event.addr, event.enode);
					return;
				}
				delete bundle.knownPeers[event.enode];
				return;
			}
			}
			if (!prev.peers.bundles[event.addr]) {
				prev.peers.bundles[event.addr] = {
					location: {
						country:   '',
						city:      '',
						latitude:  0,
						longitude: 0,
					},
					shortLocation: '',
					knownPeers: {},
					attempts:   0,
				};
			}
			const bundle = prev.peers.bundles[event.addr];
			if (event.location) {
				bundle.location = event.location;
				bundle.shortLocation = shortLocation(bundle.location);
				return;
			}
			if (!event.enode) {
				bundle.attempts++;
				return;
			}
			if (!bundle.knownPeers) {
				bundle.knownPeers = {};
			}
			if (!bundle.knownPeers[event.enode]) {
				bundle.knownPeers[event.enode] = {
					connected:    [],
					disconnected: [],
					ingress:      [],
					egress:       [],
					active:       false,
					name:         '',
					shortName:    '',
					enode:        '',
					protocols:    {},
					eth:          '',
				};
			}
			const peer = bundle.knownPeers[event.enode];
			if (event.name) {
				peer.name = event.name;
				peer.shortName = shortName(event.name);
			}
			if (event.enode) {
				peer.enode = event.enode;
			}
			if (event.protocols) {
				peer.protocols = event.protocols;
				peer.eth = ethProtocol(peer.protocols);
			}
			if (!peer.maxIngress) {
				setIngressChartAttributes(peer);
			}
			if (!peer.maxEgress) {
				setEgressChartAttributes(peer);
			}
			if (event.connected) {
				if (!peer.connected) {
					console.warn('peer.connected should exist');
					peer.connected = [];
				}
				peer.connected.push(event.connected);
			}
			if (event.disconnected) {
				if (!peer.disconnected) {
					console.warn('peer.disconnected should exist');
					peer.disconnected = [];
				}
				peer.disconnected.push(event.disconnected);
			}
			switch (event.activity) {
			case 'active':
				peer.active = true;
				break;
			case 'inactive':
				peer.active = false;
				break;
			}
			if (Array.isArray(event.ingress) && Array.isArray(event.egress)) {
				if (event.ingress.length !== event.egress.length) {
					console.error('Different traffic sample length', event);
					return;
				}
				// Check if there is a new maximum value, and reset the colors in case.
				let maxIngress = peer.maxIngress;
				event.ingress.forEach(({value}) => {
					if (value > maxIngress) {
						maxIngress = value;
					}
				});
				if (maxIngress > peer.maxIngress) {
					setMaxIngress(peer, maxIngress);
				}
				// Push the new values.
				peer.ingress.splice(peer.ingress.length, 0, ...event.ingress);
				const ingressDiff = peer.ingress.length - sampleLimit;
				if (ingressDiff > 0) {
					// Check if the maximum value is in the beginning.
					let i = 0;
					while (i < ingressDiff && peer.ingress[i].value < peer.maxIngress) {
						i++;
					}
					// Remove the old values from the beginning.
					peer.ingress.splice(0, ingressDiff);
					if (i < ingressDiff) {
						// Reset the colors if the maximum value leaves the chart.
						setIngressChartAttributes(peer);
					}
				}
				// Check if there is a new maximum value, and reset the colors in case.
				let maxEgress = peer.maxEgress;
				event.egress.forEach(({value}) => {
					if (value > maxEgress) {
						maxEgress = value;
					}
				});
				if (maxEgress > peer.maxEgress) {
					setMaxEgress(peer, maxEgress);
				}
				// Push the new values.
				peer.egress.splice(peer.egress.length, 0, ...event.egress);
				const egressDiff = peer.egress.length - sampleLimit;
				if (egressDiff > 0) {
					// Check if the maximum value is in the beginning.
					let i = 0;
					while (i < egressDiff && peer.egress[i].value < peer.maxEgress) {
						i++;
					}
					// Remove the old values from the beginning.
					peer.egress.splice(0, egressDiff);
					if (i < egressDiff) {
						// Reset the colors if the maximum value leaves the chart.
						setEgressChartAttributes(peer);
					}
				}
			}
		});
	}
	return prev;
};

// styles contains the constant styles of the component.
const styles = {
	table: {
		background:     '#212121',
		borderCollapse: 'unset',
		padding:        5,
	},
	tableHead: {
		height: 'auto',
	},
	tableRow: {
		height: 'auto',
	},
	tableCell: {
		paddingTop:    0,
		paddingRight:  5,
		paddingBottom: 0,
		paddingLeft:   5,
		border:        'none',
		fontFamily:    'monospace',
		fontSize:      10,
	},
};

// limitedWidthStyle returns a style object which cuts the long text with three dots.
const limitedWidthStyle = (width) => {
	return {
		textOverflow: 'ellipsis',
		maxWidth:     width,
		overflow:     'hidden',
		whiteSpace:   'nowrap',
	};
};

export type Props = {
    container:    Object,
    content:      NetworkType,
    shouldUpdate: Object,
};

type State = {};

// Network renders the network page.
class Network extends Component<Props, State> {
	componentDidMount() {
		const {container} = this.props;
		if (typeof container === 'undefined') {
			return;
		}
		container.scrollTop = 0;
	}

	formatTime = (t: string) => {
		const time = new Date(t);
		if (isNaN(time)) {
			return '';
		}
		const month = `0${time.getMonth() + 1}`.slice(-2);
		const date = `0${time.getDate()}`.slice(-2);
		const hours = `0${time.getHours()}`.slice(-2);
		const minutes = `0${time.getMinutes()}`.slice(-2);
		const seconds = `0${time.getSeconds()}`.slice(-2);
		return `${month}/${date}/${hours}:${minutes}:${seconds}`;
	};

	copyToClipboard = (text: string) => (event) => {
		event.preventDefault();
		navigator.clipboard.writeText(text).then(() => {}, () => {
			console.error("Failed to copy", text);
		});
	};

	knownPeerTableRow = (addr, enode, bundle, peer) => {
		const ingressValues = peer.ingress.map(({value}) => ({ingress: value || 0.001}));
		const egressValues = peer.egress.map(({value}) => ({egress: -value || -0.001}));
		return (
			<TableRow key={`known_${addr}_${enode}`} style={styles.tableRow}>
				<TableCell style={styles.tableCell}>
					{peer.active
						? <FontAwesomeIcon icon={fasCircle} color='green' />
						: <FontAwesomeIcon icon={farCircle} style={commonStyles.light} />
					}
				</TableCell>
				<TableCell
					style={{
						cursor: 'copy',
						...styles.tableCell,
						...commonStyles.light,
					}}
					onClick={this.copyToClipboard(enode)}
				>
					{enode.substring(8, 16)}
				</TableCell>
				<TableCell
					style={{
						cursor: 'copy',
						...limitedWidthStyle(120),
						...styles.tableCell,
					}}
					onClick={this.copyToClipboard(peer.name)}
				>
					{peer.shortName}
				</TableCell>
				<TableCell
					style={{
						cursor: 'copy',
						...limitedWidthStyle(120),
						...styles.tableCell,
					}}
					onClick={this.copyToClipboard(JSON.stringify(bundle.location))}
				>
					{bundle.shortLocation}
				</TableCell>
				<TableCell style={styles.tableCell}>
					<AreaChart
						width={trafficChartWidth}
						height={trafficChartHeight}
						data={ingressValues}
						margin={{top: 5, right: 5, bottom: 0, left: 5}}
						syncId={`peerIngress_${addr}_${enode}`}
					>
						<defs>
							<linearGradient id={`ingressGradient_${addr}_${enode}`} x1='0' y1='1' x2='0' y2='0'>
								{peer.ingressGradient
								&& peer.ingressGradient.map(({offset, color}, i) => (
									<stop
										key={`ingressStop_${addr}_${enode}_${i}`}
										offset={`${offset}%`}
										stopColor={color}
									/>
								))}
							</linearGradient>
						</defs>
						<Tooltip cursor={false} content={<CustomTooltip tooltip={bytePlotter('Download')} />} />
						<YAxis hide scale='sqrt' domain={[0.001, dataMax => Math.max(dataMax, 0)]} />
						<Area
							dataKey='ingress'
							isAnimationActive={false}
							type='monotone'
							fill={`url(#ingressGradient_${addr}_${enode})`}
							stroke={peer.ingressGradient[peer.ingressGradient.length - 1].color}
							strokeWidth={chartStrokeWidth}
						/>
					</AreaChart>
					<AreaChart
						width={trafficChartWidth}
						height={trafficChartHeight}
						data={egressValues}
						margin={{top: 0, right: 5, bottom: 5, left: 5}}
						syncId={`peerIngress_${addr}_${enode}`}
					>
						<defs>
							<linearGradient id={`egressGradient_${addr}_${enode}`} x1='0' y1='1' x2='0' y2='0'>
								{peer.egressGradient
								&& peer.egressGradient.map(({offset, color}, i) => (
									<stop
										key={`egressStop_${addr}_${enode}_${i}`}
										offset={`${offset}%`}
										stopColor={color}
									/>
								))}
							</linearGradient>
						</defs>
						<Tooltip cursor={false} content={<CustomTooltip tooltip={bytePlotter('Upload', multiplier(-1))} />} />
						<YAxis hide scale='sqrt' domain={[dataMin => Math.min(dataMin, 0), -0.001]} />
						<Area
							dataKey='egress'
							isAnimationActive={false}
							type='monotone'
							fill={`url(#egressGradient_${addr}_${enode})`}
							stroke={peer.egressGradient[0].color}
							strokeWidth={chartStrokeWidth}
						/>
					</AreaChart>
				</TableCell>
				<TableCell
					style={{cursor: 'copy', ...styles.tableCell}}
					onClick={this.copyToClipboard(JSON.stringify(peer.protocols.eth))}
				>
					{peer.eth}
				</TableCell>
			</TableRow>
		);
	};

	connectionAttemptTableRow = (addr, bundle) => (
		<TableRow key={`attempt_${addr}`} style={styles.tableRow}>
			<TableCell
				style={{cursor: 'copy', ...styles.tableCell}}
				onClick={this.copyToClipboard(addr)}
			>
				{addr}
			</TableCell>
			<TableCell
				style={{cursor: 'copy', ...limitedWidthStyle(120), ...styles.tableCell}}
				onClick={this.copyToClipboard(JSON.stringify(bundle.location))}
			>
				{bundle.shortLocation}
			</TableCell>
			<TableCell style={styles.tableCell}>
				{bundle.attempts}
			</TableCell>
		</TableRow>
	);

	render() {
		return (
			<Grid container direction='row' justify='space-between'>
				<Grid item>
					<Table style={styles.table}>
						<TableHead style={styles.tableHead}>
							<TableRow style={styles.tableRow}>
								<TableCell style={styles.tableCell} />
								<TableCell style={styles.tableCell}>Node URL</TableCell>
								<TableCell style={styles.tableCell}>Name</TableCell>
								<TableCell style={styles.tableCell}>Location</TableCell>
								<TableCell style={styles.tableCell}>Traffic</TableCell>
								<TableCell style={styles.tableCell}>ETH protocol</TableCell>
							</TableRow>
						</TableHead>
						<TableBody>
							{Object.entries(this.props.content.peers.bundles).map(([addr, bundle]) => {
								if (!bundle.knownPeers || Object.keys(bundle.knownPeers).length < 1) {
									return null;
								}
								return Object.entries(bundle.knownPeers).map(([enode, peer]) => {
									if (peer.active === false) {
										return null;
									}
									return this.knownPeerTableRow(addr, enode, bundle, peer);
								});
							})}
						</TableBody>
						<TableBody>
							{Object.entries(this.props.content.peers.bundles).map(([addr, bundle]) => {
								if (!bundle.knownPeers || Object.keys(bundle.knownPeers).length < 1) {
									return null;
								}
								return Object.entries(bundle.knownPeers).map(([enode, peer]) => {
									if (peer.active === true) {
										return null;
									}
									return this.knownPeerTableRow(addr, enode, bundle, peer);
								});
							})}
						</TableBody>
					</Table>
				</Grid>
				<Grid item>
					<div style={styles.table}>
						<Typography variant='subtitle1' gutterBottom>
							Connection attempts
						</Typography>
						<Table>
							<TableHead style={styles.tableHead}>
								<TableRow style={styles.tableRow}>
									<TableCell style={styles.tableCell}>TCP address</TableCell>
									<TableCell style={styles.tableCell}>Location</TableCell>
									<TableCell style={styles.tableCell}>Nr</TableCell>
								</TableRow>
							</TableHead>
							<TableBody>
								{Object.entries(this.props.content.peers.bundles).map(([addr, bundle]) => {
									if (!bundle.attempts || bundle.attempts <= attemptSeparator) {
										return null;
									}
									return this.connectionAttemptTableRow(addr, bundle);
								})}
							</TableBody>
							<TableBody>
								{Object.entries(this.props.content.peers.bundles).map(([addr, bundle]) => {
									if (!bundle.attempts || bundle.attempts < 1 || bundle.attempts > attemptSeparator) {
										return null;
									}
									return this.connectionAttemptTableRow(addr, bundle);
								})}
							</TableBody>
						</Table>
					</div>
				</Grid>
			</Grid>
		);
	}
}

export default Network;
