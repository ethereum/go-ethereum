import React, {Component} from 'react';
import PropTypes from 'prop-types';
import classNames from 'classnames';
import {withStyles} from 'material-ui/styles';
import Grid from 'material-ui/Grid';
import {LineChart, AreaChart, Area, YAxis, CartesianGrid, Line, ResponsiveContainer} from 'recharts';

import {TAGS, DRAWER_WIDTH} from "./Common.jsx";

// ChartGrid renders a grid container for responsive charts.
// The children are Recharts components extended with the Material-UI's xs property.
class ChartGrid extends Component {
    render() {
        return (
            <Grid container spacing={this.props.spacing}>
                {
                    React.Children.map(this.props.children, child => (
                        <Grid item xs={child.props.xs}>
                            <ResponsiveContainer width="100%" height={child.props.height}>
                                {child}
                            </ResponsiveContainer>
                        </Grid>
                    ))
                }
            </Grid>
        );
    }
}

ChartGrid.propTypes = {
    spacing: PropTypes.number.isRequired,
};

// ContentSwitch chooses and renders the proper page content.
class ContentSwitch extends Component {
    render() {
        switch(this.props.active) {
            case TAGS.home.id:
                return (
                    <ChartGrid spacing={24}>
                        <AreaChart xs={6} height={300} data={this.props.memory}>
                            <YAxis />
                            <Area type="monotone" dataKey="memory" stroke="#8884d8" fill="#8884d8" />
                        </AreaChart>
                        <LineChart xs={6} height={300} data={this.props.traffic}>
                            <Line type="monotone" dataKey="traffic" dot={false} />
                        </LineChart>
                        <LineChart xs={6} height={300} data={this.props.memory}>
                            <YAxis />
                            <CartesianGrid stroke="#eee" strokeDasharray="5 5" />
                            <Line type="monotone" dataKey="memory" stroke="#8884d8" dot={false} />
                        </LineChart>
                        <AreaChart xs={6} height={300} data={this.props.traffic}>
                            <CartesianGrid stroke="#eee" strokeDasharray="5 5" vertical={false} />
                            <Area type="monotone" dataKey="traffic" />
                        </AreaChart>
                    </ChartGrid>
                );
            case TAGS.logs.id:
                return <div>{this.props.logs.map((log, index) => <div key={index}>{log}</div>)}</div>;
            case TAGS.networking.id:
                // Only for testing.
                return (
                    <Grid container spacing={24}>
                        <Grid item xs={6}>
                            <ResponsiveContainer width="100%" height={300}>
                                <LineChart data={this.props.traffic}>
                                    <Line type="monotone" dataKey="traffic" dot={false} />
                                </LineChart>
                            </ResponsiveContainer>
                        </Grid>
                        <Grid item xs={6}>
                            <ResponsiveContainer width="100%" height={300}>
                                <AreaChart data={this.props.memory}>
                                    <YAxis />
                                    <Area type="monotone" dataKey="memory" stroke="#8884d8" fill="#8884d8" />
                                </AreaChart>
                            </ResponsiveContainer>
                        </Grid>
                        <Grid item xs={6}>
                            <ResponsiveContainer width="100%" height={300}>
                                <AreaChart data={this.props.traffic}>
                                    <CartesianGrid stroke="#eee" strokeDasharray="5 5" vertical={false} />
                                    <Area type="monotone" dataKey="traffic" />
                                </AreaChart>
                            </ResponsiveContainer>
                        </Grid>
                    </Grid>

                );
            case TAGS.txpool.id:
            case TAGS.blockchain.id:
            case TAGS.system.id:
        }
        return null;
    }
}

ContentSwitch.propTypes = {
    active: PropTypes.string.isRequired,
};

// Styles for the Main component.
const styles = theme => ({
    content: {
        width: '100%',
        marginLeft: -DRAWER_WIDTH,
        flexGrow: 1,
        backgroundColor: theme.palette.background.default,
        padding: theme.spacing.unit * 3,
        transition: theme.transitions.create('margin', {
            easing: theme.transitions.easing.sharp,
            duration: theme.transitions.duration.leavingScreen,
        }),
        marginTop: 56,
        overflow: 'auto',
        [theme.breakpoints.up('sm')]: {
            content: {
                height: 'calc(100% - 64px)',
                marginTop: 64,
            },
        },
    },
    contentShift: {
        marginLeft: 0,
        transition: theme.transitions.create('margin', {
            easing: theme.transitions.easing.easeOut,
            duration: theme.transitions.duration.enteringScreen,
        }),
    },
});

// Main renders a component for the page content.
class Main extends Component {
    render() {
        // The classes property is injected by withStyles().
        const {classes} = this.props;

        return (
            <main className={classNames(classes.content, this.props.opened && classes.contentShift)}>
                <ContentSwitch
                    active={this.props.active}
                    memory={this.props.memory}
                    traffic={this.props.traffic}
                    logs={this.props.logs}
                />
            </main>
        );
    }
}

Main.propTypes = {
    classes: PropTypes.object.isRequired,
    opened: PropTypes.bool.isRequired,
    active: PropTypes.string.isRequired,
};

export default withStyles(styles)(Main);