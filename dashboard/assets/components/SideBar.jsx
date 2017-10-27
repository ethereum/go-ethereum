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
        height: '100%',
        width: DRAWER_WIDTH,
    },
    drawerHeader: {
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'flex-end',
        padding: '0 8px',
        ...theme.mixins.toolbar,
        transitionDuration: {
            enter: theme.transitions.duration.enteringScreen,
            exit: theme.transitions.duration.leavingScreen,
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
            this.clickOn[id] = e => {
                e.preventDefault();
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
                                )
                            })
                        }
                    </List>
                </div>
            </Drawer>
        );
    }
}

SideBar.propTypes = {
    classes: PropTypes.object.isRequired,
    opened: PropTypes.bool.isRequired,
    close: PropTypes.func.isRequired,
    changeContent: PropTypes.func.isRequired,
};

export default withStyles(styles)(SideBar);