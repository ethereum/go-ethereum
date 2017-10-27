const path = require('path');

module.exports = {
    entry:  './index.jsx',
    output: {
        path:     path.resolve(__dirname, 'public'),
        filename: 'bundle.js',
    },
    module: {
        loaders: [
            {
                test: /\.jsx$/, // regexp for JSX files
                loader: 'babel-loader', // The babel configuration is in the package.json.
                query: {
                    presets: ['env', 'react', 'stage-0']
                }
            },
        ],
    },
};