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
import PropTypes from 'prop-types';
import {withStyles} from 'material-ui/styles';
import Drawer from 'material-ui/Drawer';
import {IconButton} from "material-ui";
import List, {ListItem, ListItemText} from 'material-ui/List';
import ChevronLeftIcon from 'material-ui-icons/ChevronLeft';

import {TAGS, DRAWER_WIDTH} from './Common.jsx';

// Styles for the SideBar component.
const styles = theme => ({
    drawerPaper: {
        position: 'relative',
        height:   '100%',
        width:    DRAWER_WIDTH,
    },
    drawerHeader: {
        display:            'flex',
        alignItems:         'center',
        justifyContent:     'flex-end',
        padding:            '0 8px',
        ...theme.mixins.toolbar,
        transitionDuration: {
            enter: theme.transitions.duration.enteringScreen,
            exit:  theme.transitions.duration.leavingScreen,
        }
    },
});

// SideBar renders a sidebar component.
class SideBar extends Component {
    constructor(props) {
        super(props);

        // clickOn contains onClick event functions for the menu items.
        // Instantiate only once, and reuse the existing functions to prevent the creation of
        // new function instances every time the render method is triggered.
        this.clickOn = {};
        for(let key in TAGS) {
            const id = TAGS[key].id;
            this.clickOn[id] = event => {
                event.preventDefault();
                console.log(event.target.key);
                this.props.changeContent(id);
            };
        }
    }

    render() {
        // The classes property is injected by withStyles().
        const {classes} = this.props;

        return (
            <Drawer
                type="persistent"
                classes={{paper: classes.drawerPaper,}}
                open={this.props.opened}
            >
                <div>
                    <div className={classes.drawerHeader}>
                        <IconButton onClick={this.props.close}>
                            <ChevronLeftIcon />
                        </IconButton>
                    </div>
                    <List>
                        {
                            Object.values(TAGS).map(tag => {
                                return (
                                    <ListItem button key={tag.id} onClick={this.clickOn[tag.id]}>
                                        <ListItemText primary={tag.title} />
                                    </ListItem>
                                );
                            })
                        }
                    </List>
                </div>
            </Drawer>
        );
    }
}

SideBar.propTypes = {
    classes:       PropTypes.object.isRequired,
    opened:        PropTypes.bool.isRequired,
    close:         PropTypes.func.isRequired,
    changeContent: PropTypes.func.isRequired,
};

export default withStyles(styles)(SideBar);
