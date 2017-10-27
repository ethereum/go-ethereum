import React, {Component} from 'react';
import PropTypes from 'prop-types';
import classNames from 'classnames';
import {withStyles} from 'material-ui/styles';
import AppBar from 'material-ui/AppBar';
import Toolbar from 'material-ui/Toolbar';
import Typography from 'material-ui/Typography';
import IconButton from 'material-ui/IconButton';
import MenuIcon from 'material-ui-icons/Menu';

import {DRAWER_WIDTH} from './Common.jsx';

// Styles for the Header component.
const styles = theme => ({
    appBar: {
        position: 'absolute',
        transition: theme.transitions.create(['margin', 'width'], {
            easing: theme.transitions.easing.sharp,
            duration: theme.transitions.duration.leavingScreen,
        }),
    },
    appBarShift: {
        marginLeft: DRAWER_WIDTH,
        width: `calc(100% - ${DRAWER_WIDTH}px)`,
        transition: theme.transitions.create(['margin', 'width'], {
            easing: theme.transitions.easing.easeOut,
            duration: theme.transitions.duration.enteringScreen,
        }),
    },
    menuButton: {
        marginLeft: 12,
        marginRight: 20,
    },
    hide: {
        display: 'none',
    },
});

// Header renders a header, which contains a sidebar opener icon when that is closed.
class Header extends Component {
    render() {
        // The classes property is injected by withStyles().
        const {classes} = this.props;

        return (
            <AppBar className={classNames(classes.appBar, this.props.opened && classes.appBarShift)}>
                <Toolbar disableGutters={!this.props.opened}>
                    <IconButton
                        color="contrast"
                        aria-label="open drawer"
                        onClick={this.props.open}
                        className={classNames(classes.menuButton, this.props.opened && classes.hide)}
                    >
                        <MenuIcon />
                    </IconButton>
                    <Typography type="title" color="inherit" noWrap>
                        Go Ethereum Dashboard
                    </Typography>
                </Toolbar>
            </AppBar>
        );
    }
}

Header.propTypes = {
    classes: PropTypes.object.isRequired,
    opened: PropTypes.bool.isRequired,
    open: PropTypes.func.isRequired,
};

export default withStyles(styles)(Header);