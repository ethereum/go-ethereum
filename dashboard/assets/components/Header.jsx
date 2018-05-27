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
import IconButton from 'material-ui/IconButton';
import Icon from 'material-ui/Icon';
import MenuIcon from 'material-ui-icons/Menu';
import Typography from 'material-ui/Typography';

// themeStyles returns the styles generated from the theme for the component.
const themeStyles = (theme: Object) => ({
	header: {
		backgroundColor: theme.palette.grey[900],
		color:           theme.palette.getContrastText(theme.palette.grey[900]),
		zIndex:          theme.zIndex.appBar,
	},
	toolbar: {
		paddingLeft:  theme.spacing.unit,
		paddingRight: theme.spacing.unit,
	},
	title: {
		paddingLeft: theme.spacing.unit,
		fontSize:    3 * theme.spacing.unit,
	},
});

export type Props = {
	classes: Object, // injected by withStyles()
	switchSideBar: () => void,
};

// Header renders the header of the dashboard.
class Header extends Component<Props> {
	render() {
		const {classes} = this.props;

		return (
			<AppBar position='static' className={classes.header}>
				<Toolbar className={classes.toolbar}>
					<IconButton onClick={this.props.switchSideBar}>
						<Icon>
							<MenuIcon />
						</Icon>
					</IconButton>
					<Typography type='title' color='inherit' noWrap className={classes.title}>
						Go Ethereum Dashboard
					</Typography>
				</Toolbar>
			</AppBar>
		);
	}
}

export default withStyles(themeStyles)(Header);
