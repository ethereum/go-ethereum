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
import Grid from 'material-ui/Grid';
import {LineChart, AreaChart, Area, YAxis, CartesianGrid, Line, ResponsiveContainer} from 'recharts';
import {withTheme} from 'material-ui/styles';

import {isNullOrUndefined, DATA_KEYS} from "./Common.jsx";

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
                                {React.cloneElement(child, {data: child.props.values.map(value => ({value: value}))})}
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

// Home renders the home component.
class Home extends Component {
    shouldComponentUpdate(nextProps) {
        return !isNullOrUndefined(nextProps.shouldUpdate[DATA_KEYS.memory]) ||
            !isNullOrUndefined(nextProps.shouldUpdate[DATA_KEYS.traffic]);
    }

    render() {
        const {theme} = this.props;
        const memoryColor = theme.palette.primary[300];
        const trafficColor = theme.palette.secondary[300];

        return (
            <ChartGrid spacing={24}>
                <AreaChart xs={6} height={300} values={this.props.memory}>
                    <YAxis />
                    <Area type="monotone" dataKey="value" stroke={memoryColor} fill={memoryColor} />
                </AreaChart>
                <LineChart xs={6} height={300} values={this.props.traffic}>
                    <Line type="monotone" dataKey="value" stroke={trafficColor} dot={false} />
                </LineChart>
                <LineChart xs={6} height={300} values={this.props.memory}>
                    <YAxis />
                    <CartesianGrid stroke="#eee" strokeDasharray="5 5" />
                    <Line type="monotone" dataKey="value" stroke={memoryColor} dot={false} />
                </LineChart>
                <AreaChart xs={6} height={300} values={this.props.traffic}>
                    <CartesianGrid stroke="#eee" strokeDasharray="5 5" vertical={false} />
                    <Area type="monotone" dataKey="value" stroke={trafficColor} fill={trafficColor} />
                </AreaChart>
            </ChartGrid>
        );
    }
}

Home.propTypes = {
    theme:        PropTypes.object.isRequired,
    shouldUpdate: PropTypes.object.isRequired,
};

export default withTheme()(Home);
