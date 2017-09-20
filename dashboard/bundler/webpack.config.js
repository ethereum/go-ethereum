const path = require('path');
const webpack = require('webpack');

const plugins = [
    new webpack.optimize.CommonsChunkPlugin({
        name:      'main', // Move dependencies to our main file.
        children:  true, // Look for common dependencies in all children,
        minChunks: 2, // How many times a dependency must come up before being extracted
    }),

    // This plugins optimizes chunks and modules by
    // how much they are used in your app.
    new webpack.optimize.OccurrenceOrderPlugin(),

    // This plugin prevents Webpack from creating chunks
    // that would be too small to be worth loading separately.
    new webpack.optimize.MinChunkSizePlugin({
        minChunkSize: 51200, // ~50kb
    }),

    // This plugin minifies all the Javascript code of the final bundle.
    new webpack.optimize.UglifyJsPlugin({
        mangle:   true,
        compress: {
            warnings: false, // Suppress uglification warnings
        },
    }),

    // This plugin defines various variables that we can set to false
    // to avoid code related to them from being compiled in the final bundle.
    new webpack.DefinePlugin({
        __SERVER__:      false,
        __DEVELOPMENT__: false,
        __DEVTOOLS__:    false,
        'process.env':   {
            BABEL_ENV: JSON.stringify(process.env.NODE_ENV),
        },
    }),
];

module.exports = {
    entry:  './src/index.js',
    output: {
        path:     path.resolve(__dirname, '../assets/js'),
        filename: 'bundle.js',
    },
    plugins: plugins,
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