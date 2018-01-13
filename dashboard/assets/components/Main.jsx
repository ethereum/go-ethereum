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

import Home from './Home';
import {MENU} from './Common';
import type {Content} from '../types/content';

// Styles for the Content component.
const styles = theme => ({
	content: {
		flexGrow:        1,
		backgroundColor: theme.palette.background.default,
		padding:         theme.spacing.unit * 3,
		overflow:        'auto',
	},
});
export type Props = {
	classes: Object,
	active: string,
	content: Content,
	shouldUpdate: Object,
};
// Main renders the chosen content.
class Main extends Component<Props> {
	render() {
		const {
			classes, active, content, shouldUpdate,
		} = this.props;

		let children = null;
		switch (active) {
		case MENU.get('home').id:
			children = <Home memory={content.home.memory} traffic={content.home.traffic} shouldUpdate={shouldUpdate} />;
			break;
		case MENU.get('chain').id:
		case MENU.get('txpool').id:
		case MENU.get('network').id:
		case MENU.get('system').id:
			children = <div>Work in progress.</div>;
			break;
		case MENU.get('logs').id:
			children = <div>{content.logs.log.map((log, index) => <div key={index}>{log}</div>)}</div>;
		}

		return <div className={classes.content}>{children}</div>;
	}
}

export default withStyles(styles)(Main);
