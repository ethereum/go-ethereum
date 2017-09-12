const path = require('path');

module.exports = {
    entry:  './src/index.js',
    output: {
        path:     path.resolve(__dirname, '../assets/js'),
        filename: 'bundle.js',
    },
    externals: { // External libraries, which will not be included in the bundled file(s).
        inferno: 'Inferno',
        component: 'Inferno.Component',
    },
    module: {
        loaders: [
            {
                test: /\.js$/, // regexp for JS files
                loader: 'babel-loader', // The babel configuration is in the package.json.
            },
        ],
    },
};