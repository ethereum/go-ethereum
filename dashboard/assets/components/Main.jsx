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

import withStyles from '@material-ui/core/styles/withStyles';

import Network from 'Network';
import Logs from 'Logs';
import Footer from 'Footer';
import {MENU} from '../common';
import type {Content} from '../types/content';

// styles contains the constant styles of the component.
const styles = {
	wrapper: {
		display:       'flex',
		flexDirection: 'column',
		width:         '100%',
	},
	content: {
		flex:     1,
		overflow: 'auto',
	},
};

// themeStyles returns the styles generated from the theme for the component.
const themeStyles = theme => ({
	content: {
		backgroundColor: theme.palette.background.default,
		padding:         theme.spacing.unit * 3,
	},
});

export type Props = {
	classes:      Object,
	active:       string,
	content:      Content,
	shouldUpdate: Object,
	send:         string => void,
};

type State = {};

// Main renders the chosen content.
class Main extends Component<Props, State> {
	constructor(props) {
		super(props);
		this.container = React.createRef();
		this.content = React.createRef();
	}

	componentDidUpdate(prevProps, prevState, snapshot) {
		if (this.content && typeof this.content.didUpdate === 'function') {
			this.content.didUpdate(prevProps, prevState, snapshot);
		}
	}

	onScroll = () => {
		if (this.content && typeof this.content.onScroll === 'function') {
			this.content.onScroll();
		}
	};

	getSnapshotBeforeUpdate(prevProps: Readonly<P>, prevState: Readonly<S>) {
		if (this.content && typeof this.content.beforeUpdate === 'function') {
			return this.content.beforeUpdate();
		}
		return null;
	}

	render() {
		const {
			classes, active, content, shouldUpdate,
		} = this.props;

		let children = null;
		switch (active) {
		case MENU.get('home').id:
			children = <div>Work in progress.</div>;
			break;
		case MENU.get('chain').id:
			children = <div>Work in progress.</div>;
			break;
		case MENU.get('txpool').id:
			children = <div>Work in progress.</div>;
			break;
		case MENU.get('network').id:
			children = <Network
				content={this.props.content.network}
				container={this.container}
			/>;
			break;
		case MENU.get('system').id:
			children = <div>Work in progress.</div>;
			break;
		case MENU.get('logs').id:
			children = (
				<Logs
					ref={(ref) => { this.content = ref; }}
					container={this.container}
					send={this.props.send}
					content={this.props.content}
					shouldUpdate={shouldUpdate}
				/>
			);
		}

		return (
			<div style={styles.wrapper}>
				<div
					className={classes.content}
					style={styles.content}
					ref={(ref) => { this.container = ref; }}
					onScroll={this.onScroll}
				>
					{children}
				</div>
				<Footer
					general={content.general}
					system={content.system}
					shouldUpdate={shouldUpdate}
				/>
			</div>
		);
	}
}

export default withStyles(themeStyles)(Main);
