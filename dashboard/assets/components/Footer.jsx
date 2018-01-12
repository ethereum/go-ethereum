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
import AppBar from 'material-ui/AppBar';
import Toolbar from 'material-ui/Toolbar';
import Typography from 'material-ui/Typography';

import type {General} from '../types/content';

// styles contains styles for the Header component.
const styles = theme => ({
	footer: {
		backgroundColor: theme.palette.background.appBar,
		color:           theme.palette.getContrastText(theme.palette.background.appBar),
		zIndex:          theme.zIndex.appBar,
	},
	toolbar: {
		paddingLeft:    theme.spacing.unit,
		paddingRight:   theme.spacing.unit,
		display:        'flex',
		justifyContent: 'flex-end',
	},
	light: {
		color: 'rgba(255, 255, 255, 0.54)',
	},
});
export type Props = {
	general: General,
	classes: Object,
};
// TODO (kurkomisi): If the structure is appropriate, make an abstraction of the common parts with the Header.
// Footer renders the header of the dashboard.
class Footer extends Component<Props> {
	shouldComponentUpdate(nextProps) {
		return typeof nextProps.shouldUpdate.logs !== 'undefined';
	}

	info = (about: string, data: string) => (
		<Typography type="caption" color="inherit">
			<span className={this.props.classes.light}>{about}</span> {data}
		</Typography>
	);

	render() {
		const {classes, general} = this.props; // The classes property is injected by withStyles().
		const geth = general.version ? this.info('Geth', general.version) : null;
		const commit = general.commit ? this.info('Commit', general.commit.substring(0, 7)) : null;

		return (
			<AppBar position="static" className={classes.footer}>
				<Toolbar className={classes.toolbar}>
					<div>
						{geth}
						{commit}
					</div>
				</Toolbar>
			</AppBar>
		);
	}
}

export default withStyles(styles)(Footer);
