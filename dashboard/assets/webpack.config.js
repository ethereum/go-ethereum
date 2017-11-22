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

const webpack = require('webpack');
const path = require('path');

module.exports = {
    entry:  './index.jsx',
    output: {
        path:     path.resolve(__dirname, 'public'),
        filename: 'bundle.js',
    },
    plugins: [
        new webpack.optimize.UglifyJsPlugin({
            comments: false,
            mangle:   false,
            beautify: true,
        }),
    ],
    module: {
        loaders: [
            {
                test:   /\.jsx$/, // regexp for JSX files
                loader: 'babel-loader',
                query:  {
                    plugins: ['transform-decorators-legacy'], // @withStyles, @withTheme
                    presets: ['env', 'react', 'stage-0'],
                },
            },
            {
                test: /font-awesome\.css$/,
                use:  [
                    'style-loader',
                    'css-loader',
                    path.resolve(__dirname, './faOnlyWoffLoader.js'),
                ],
            },
            {
                test:   /\.woff2?$/,
                loader: 'url-loader',
            },
        ],
    },
};
