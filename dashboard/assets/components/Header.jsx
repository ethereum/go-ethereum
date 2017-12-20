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
	opened: boolean,
	openSideBar: () => {},
	closeSideBar: () => {},
};
// Header renders the header of the dashboard.
class Header extends Component<Props> {
	shouldComponentUpdate(nextProps) {
		return nextProps.opened !== this.props.opened;
	}

	// changeSideBar opens or closes the sidebar corresponding to the previous state.
	changeSideBar = () => {
		if (this.props.opened) {
			this.props.closeSideBar();
		} else {
			this.props.openSideBar();
		}
	};

	// arrowButton is connected to the sidebar; changes its state.
	arrowButton = (transitionState: string) => (
		<IconButton onClick={this.changeSideBar}>
			<ChevronLeftIcon
				style={{
					...arrowDefault,
					...arrowTransition[transitionState],
				}}
			/>
		</IconButton>
	);

	render() {
		const {classes, opened} = this.props; // The classes property is injected by withStyles().

		return (
			<AppBar position="static" className={classes.header}>
				<Toolbar className={classes.toolbar}>
					<Transition mountOnEnter in={opened} timeout={{enter: DURATION}}>
						{this.arrowButton}
					</Transition>
					<Typography type="title" color="inherit" noWrap className={classes.mainText}>
						Go Ethereum Dashboard
					</Typography>
				</Toolbar>
			</AppBar>
		);
	}
}

export default withStyles(styles)(Header);
