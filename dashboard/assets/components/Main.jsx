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

import {MENU} from '../common';
import Logs from './Logs';
import Footer from './Footer';
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
	classes: Object,
	active: string,
	content: Content,
	shouldUpdate: Object,
    send: (string) => void,
	logs: () => Object,
};

// Main renders the chosen content.
class Main extends Component<Props> {
	handleScroll = () => {
		if (typeof this.container !== 'undefined') {
			// console.log(this.container.scrollTop, this.container.scrollHeight);
			if (this.container.scrollTop === 0) {
				// this.props.send(JSON.stringify({Logs: {Time: '2018-04-11T12:48:18.181274193+03:00'}}));
				console.log("Top");
			}
			if (this.container.scrollHeight - this.container.scrollTop === this.container.clientHeight) {
				console.log("Bottom");
				// this.container.scrollTop = 0;
			}
		}
	};

	componentDidUpdate() {
		// if (typeof this.container !== 'undefined') {
		// 	this.container.scrollTop = this.container.scrollHeight - this.container.clientHeight;
		// }
	}

	render() {
		const {
			classes, active, content, shouldUpdate,
		} = this.props;

		let children = null;
		switch (active) {
		case MENU.get('home').id:
		case MENU.get('chain').id:
		case MENU.get('txpool').id:
		case MENU.get('network').id:
		case MENU.get('system').id:
			children = <div>Work in progress.</div>;
			break;
		case MENU.get('logs').id:
			children = <Logs logs={this.props.logs} />;
		}

		return (
			<div style={styles.wrapper}>
				<div
					className={classes.content}
					style={styles.content}
					ref={(ref) => { this.container = ref; }}
					onScroll={this.handleScroll}
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
