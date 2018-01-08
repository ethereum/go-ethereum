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
import {lensPath, view, set} from 'ramda';

import Header from './Header';
import Body from './Body';
import {MENU, SAMPLE} from './Common';
import type {Message, HomeMessage, LogsMessage, Chart} from '../types/message';
import type {Content} from '../types/content';

// appender appends an array (A) to the end of another array (B) in the state.
// lens is the path of B in the state, samples is A, and limit is the maximum size of the changed array.
//
// appender retrieves a function, which overrides the state's value at lens, and returns with the overridden state.
const appender = (lens, samples, limit) => (state) => {
	const newSamples = [
		...view(lens, state), // retrieves a specific value of the state at the given path (lens).
		...samples,
	];
	// set is a function of ramda.js, which needs the path, the new value, the original state, and retrieves
	// the altered state.
	return set(
		lens,
		newSamples.slice(newSamples.length > limit ? newSamples.length - limit : 0),
		state
	);
};
// Lenses for specific data fields in the state, used for a clearer deep update.
// NOTE: This solution will be changed very likely.
const memoryLens = lensPath(['content', 'home', 'memory']);
const trafficLens = lensPath(['content', 'home', 'traffic']);
const logLens = lensPath(['content', 'logs', 'log']);
// styles retrieves the styles for the Dashboard component.
const styles = theme => ({
	dashboard: {
		display:    'flex',
		flexFlow:   'column',
		width:      '100%',
		height:     '100%',
		background: theme.palette.background.default,
		zIndex:     1,
		overflow:   'hidden',
	},
});
export type Props = {
	classes: Object,
};
type State = {
	active: string, // active menu
	sideBar: boolean, // true if the sidebar is opened
	content: $Shape<Content>, // the visualized data
	shouldUpdate: Set<string> // labels for the components, which need to rerender based on the incoming message
};
// Dashboard is the main component, which renders the whole page, makes connection with the server and
// listens for messages. When there is an incoming message, updates the page's content correspondingly.
class Dashboard extends Component<Props, State> {
	constructor(props: Props) {
		super(props);
		this.state = {
			active:       MENU.get('home').id,
			sideBar:      true,
			content:      {home: {memory: [], traffic: []}, logs: {log: []}},
			shouldUpdate: new Set(),
		};
	}

	// componentDidMount initiates the establishment of the first websocket connection after the component is rendered.
	componentDidMount() {
		this.reconnect();
	}

	// reconnect establishes a websocket connection with the server, listens for incoming messages
	// and tries to reconnect on connection loss.
	reconnect = () => {
		this.setState({
			content: {home: {memory: [], traffic: []}, logs: {log: []}},
		});
		const server = new WebSocket(`${((window.location.protocol === 'https:') ? 'wss://' : 'ws://') + window.location.host}/api`);
		server.onmessage = (event) => {
			const msg: Message = JSON.parse(event.data);
			if (!msg) {
				return;
			}
			this.update(msg);
		};
		server.onclose = () => {
			setTimeout(this.reconnect, 3000);
		};
	};

	// samples retrieves the raw data of a chart field from the incoming message.
	samples = (chart: Chart) => {
		let s = [];
		if (chart.history) {
			s = chart.history.map(({value}) => (value || 0)); // traffic comes without value at the beginning
		}
		if (chart.new) {
			s = [...s, chart.new.value || 0];
		}
		return s;
	};

	// handleHome changes the home-menu related part of the state.
	handleHome = (home: HomeMessage) => {
		this.setState((prevState) => {
			let newState = prevState;
			newState.shouldUpdate = new Set();
			if (home.memory) {
				newState = appender(memoryLens, this.samples(home.memory), SAMPLE.get('memory').limit)(newState);
				newState.shouldUpdate.add('memory');
			}
			if (home.traffic) {
				newState = appender(trafficLens, this.samples(home.traffic), SAMPLE.get('traffic').limit)(newState);
				newState.shouldUpdate.add('traffic');
			}
			return newState;
		});
	};

	// handleLogs changes the logs-menu related part of the state.
	handleLogs = (logs: LogsMessage) => {
		this.setState((prevState) => {
			let newState = prevState;
			newState.shouldUpdate = new Set();
			if (logs.log) {
				newState = appender(logLens, [logs.log], SAMPLE.get('logs').limit)(newState);
				newState.shouldUpdate.add('logs');
			}
			return newState;
		});
	};

	// update analyzes the incoming message, and updates the charts' content correspondingly.
	update = (msg: Message) => {
		if (msg.home) {
			this.handleHome(msg.home);
		}
		if (msg.logs) {
			this.handleLogs(msg.logs);
		}
	};

	// changeContent sets the active label, which is used at the content rendering.
	changeContent = (newActive: string) => {
		this.setState(prevState => (prevState.active !== newActive ? {active: newActive} : {}));
	};

	// openSideBar opens the sidebar.
	openSideBar = () => {
		this.setState({sideBar: true});
	};

	// closeSideBar closes the sidebar.
	closeSideBar = () => {
		this.setState({sideBar: false});
	};

	render() {
		const {classes} = this.props; // The classes property is injected by withStyles().

		return (
			<div className={classes.dashboard}>
				<Header
					opened={this.state.sideBar}
					openSideBar={this.openSideBar}
					closeSideBar={this.closeSideBar}
				/>
				<Body
					opened={this.state.sideBar}
					changeContent={this.changeContent}
					active={this.state.active}
					content={this.state.content}
					shouldUpdate={this.state.shouldUpdate}
				/>
			</div>
		);
	}
}

export default withStyles(styles)(Dashboard);
