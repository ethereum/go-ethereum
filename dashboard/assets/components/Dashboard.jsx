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

import Header from './Header';
import Body from './Body';
import {MENU} from '../common';
import type {Content} from '../types/content';
import {inserter as logInserter} from './Logs';

// deepUpdate updates an object corresponding to the given update data, which has
// the shape of the same structure as the original object. updater also has the same
// structure, except that it contains functions where the original data needs to be
// updated. These functions are used to handle the update.
//
// Since the messages have the same shape as the state content, this approach allows
// the generalization of the message handling. The only necessary thing is to set a
// handler function for every path of the state in order to maximize the flexibility
// of the update.
const deepUpdate = (updater: Object, update: Object, prev: Object): $Shape<Content> => {
	if (typeof update === 'undefined') {
		// TODO (kurkomisi): originally this was deep copy, investigate it.
		return prev;
	}
	if (typeof updater === 'function') {
		return updater(update, prev);
	}
	const updated = {};
	Object.keys(prev).forEach((key) => {
		updated[key] = deepUpdate(updater[key], update[key], prev[key]);
	});

	return updated;
};

// shouldUpdate returns the structure of a message. It is used to prevent unnecessary render
// method triggerings. In the affected component's shouldComponentUpdate method it can be checked
// whether the involved data was changed or not by checking the message structure.
//
// We could return the message itself too, but it's safer not to give access to it.
const shouldUpdate = (updater: Object, msg: Object) => {
	const su = {};
	Object.keys(msg).forEach((key) => {
		su[key] = typeof updater[key] !== 'function' ? shouldUpdate(updater[key], msg[key]) : true;
	});

	return su;
};

// replacer is a state updater function, which replaces the original data.
const replacer = <T>(update: T) => update;

// appender is a state updater function, which appends the update data to the
// existing data. limit defines the maximum allowed size of the created array,
// mapper maps the update data.
const appender = <T>(limit: number, mapper = replacer) => (update: Array<T>, prev: Array<T>) => [
	...prev,
	...update.map(sample => mapper(sample)),
].slice(-limit);

// defaultContent returns the initial value of the state content. Needs to be a function in order to
// instantiate the object again, because it is used by the state, and isn't automatically cleaned
// when a new connection is established. The state is mutated during the update in order to avoid
// the execution of unnecessary operations (e.g. copy of the log array).
const defaultContent: () => Content = () => ({
	general: {
		version: null,
		commit:  null,
	},
	home:    {},
	chain:   {},
	txpool:  {},
	network: {},
	system:  {
		activeMemory:   [],
		virtualMemory:  [],
		networkIngress: [],
		networkEgress:  [],
		processCPU:     [],
		systemCPU:      [],
		diskRead:       [],
		diskWrite:      [],
	},
	logs: {
		chunks:        [],
		endTop:        false,
		endBottom:     true,
		topChanged:    0,
		bottomChanged: 0,
	},
});

// updaters contains the state updater functions for each path of the state.
//
// TODO (kurkomisi): Define a tricky type which embraces the content and the updaters.
const updaters = {
	general: {
		version: replacer,
		commit:  replacer,
	},
	home:    null,
	chain:   null,
	txpool:  null,
	network: null,
	system:  {
		activeMemory:   appender(200),
		virtualMemory:  appender(200),
		networkIngress: appender(200),
		networkEgress:  appender(200),
		processCPU:     appender(200),
		systemCPU:      appender(200),
		diskRead:       appender(200),
		diskWrite:      appender(200),
	},
	logs: logInserter(5),
};

// styles contains the constant styles of the component.
const styles = {
	dashboard: {
		display:  'flex',
		flexFlow: 'column',
		width:    '100%',
		height:   '100%',
		zIndex:   1,
		overflow: 'hidden',
	},
};

// themeStyles returns the styles generated from the theme for the component.
const themeStyles: Object = (theme: Object) => ({
	dashboard: {
		background: theme.palette.background.default,
	},
});

export type Props = {
	classes: Object, // injected by withStyles()
};

type State = {
	active:       string,  // active menu
	sideBar:      boolean, // true if the sidebar is opened
	content:      Content, // the visualized data
	shouldUpdate: Object,  // labels for the components, which need to re-render based on the incoming message
	server:       ?WebSocket,
};

// Dashboard is the main component, which renders the whole page, makes connection with the server and
// listens for messages. When there is an incoming message, updates the page's content correspondingly.
class Dashboard extends Component<Props, State> {
	constructor(props: Props) {
		super(props);
		this.state = {
			active:       MENU.get('home').id,
			sideBar:      true,
			content:      defaultContent(),
			shouldUpdate: {},
			server:       null,
		};
	}

	// componentDidMount initiates the establishment of the first websocket connection after the component is rendered.
	componentDidMount() {
		this.reconnect();
	}

	// reconnect establishes a websocket connection with the server, listens for incoming messages
	// and tries to reconnect on connection loss.
	reconnect = () => {
		// PROD is defined by webpack.
		const server = new WebSocket(`${((window.location.protocol === 'https:') ? 'wss://' : 'ws://')}${PROD ? window.location.host : 'localhost:8080'}/api`);
		server.onopen = () => {
			this.setState({content: defaultContent(), shouldUpdate: {}, server});
		};
		server.onmessage = (event) => {
			const msg: $Shape<Content> = JSON.parse(event.data);
			if (!msg) {
				console.error(`Incoming message is ${msg}`);
				return;
			}
			this.update(msg);
		};
		server.onclose = () => {
			this.setState({server: null});
			setTimeout(this.reconnect, 3000);
		};
	};

	// send sends a message to the server, which can be accessed only through this function for safety reasons.
	send = (msg: string) => {
		if (this.state.server != null) {
			this.state.server.send(msg);
		}
	};

	// update updates the content corresponding to the incoming message.
	update = (msg: $Shape<Content>) => {
		this.setState(prevState => ({
			content:      deepUpdate(updaters, msg, prevState.content),
			shouldUpdate: shouldUpdate(updaters, msg),
		}));
	};

	// changeContent sets the active label, which is used at the content rendering.
	changeContent = (newActive: string) => {
		this.setState(prevState => (prevState.active !== newActive ? {active: newActive} : {}));
	};

	// switchSideBar opens or closes the sidebar's state.
	switchSideBar = () => {
		this.setState(prevState => ({sideBar: !prevState.sideBar}));
	};

	render() {
		return (
			<div className={this.props.classes.dashboard} style={styles.dashboard}>
				<Header
					switchSideBar={this.switchSideBar}
				/>
				<Body
					opened={this.state.sideBar}
					changeContent={this.changeContent}
					active={this.state.active}
					content={this.state.content}
					shouldUpdate={this.state.shouldUpdate}
					send={this.send}
				/>
			</div>
		);
	}
}

export default withStyles(themeStyles)(Dashboard);
