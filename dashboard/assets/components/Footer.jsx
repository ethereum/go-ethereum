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
import Transition from 'react-transition-group/Transition';
import IconButton from 'material-ui/IconButton';
import Typography from 'material-ui/Typography';
import ChevronLeftIcon from 'material-ui-icons/ChevronLeft';

import {DURATION} from './Common';

// arrowDefault is the default style of the arrow button.
const arrowDefault = {
	transition: `transform ${DURATION}ms`,
};
// arrowTransition is the additional style of the arrow button corresponding to the transition's state.
const arrowTransition = {
	entered: {transform: 'rotate(180deg)'},
};
// Styles for the Header component.
const styles = theme => ({
	header: {
		backgroundColor: theme.palette.background.appBar,
		color:           theme.palette.getContrastText(theme.palette.background.appBar),
		zIndex:          theme.zIndex.appBar,
	},
	toolbar: {
		paddingLeft:  theme.spacing.unit,
		paddingRight: theme.spacing.unit,
	},
	mainText: {
		paddingLeft: theme.spacing.unit,
	},
});
export type Props = {
	classes: Object,
};
// TODO (kurkomisi): If the structure is appropriate, make an abstraction of the common parts with the Header.
// Footer renders the header of the dashboard.
class Footer extends Component<Props> {
	shouldComponentUpdate(nextProps) {
		return typeof nextProps.shouldUpdate.logs !== 'undefined';
	}

	render() {
		const {classes} = this.props; // The classes property is injected by withStyles().

		return (
			<AppBar position="static" className={classes.header}>
				<Toolbar className={classes.toolbar}>
					<Typography type="title" color="inherit" className={classes.mainText}>
						{this.props.general.version}
					</Typography>
					<Typography type="title" color="inherit" className={classes.mainText}>
						{this.props.general.gitCommit}	
					</Typography>
				</Toolbar>
			</AppBar>
		);
	}
}

export default withStyles(styles)(Footer);
