import React from 'react';
import {hydrate} from 'react-dom';
import {createMuiTheme, MuiThemeProvider} from 'material-ui/styles';

import Dashboard from './components/Dashboard.jsx';

// Theme for the dashboard.
const theme = createMuiTheme({
    palette: {
        type: 'dark',
    },
});

// Renders the whole dashboard.
hydrate(
    <MuiThemeProvider theme={theme}>
        <Dashboard />
    </MuiThemeProvider>,
    document.getElementById('dashboard')
); // server-side rendering