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
	resolve: {
		extensions: ['.js', '.jsx'],
	},
	entry:  './index',
	output: {
		path:     path.resolve(__dirname, ''),
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
		rules: [
			{
				test:    /\.jsx$/, // regexp for JSX files
				exclude: /node_modules/,
				use:     [ // order: from bottom to top
					{
						loader:  'babel-loader',
						options: {
							plugins: [ // order: from top to bottom
								// 'transform-decorators-legacy', // @withStyles, @withTheme
								'transform-class-properties', // static defaultProps
								'transform-flow-strip-types',
							],
							presets: [ // order: from bottom to top
								'env',
								'react',
								'stage-0',
							],
						},
					},
					// 'eslint-loader', // show errors not only in the editor, but also in the console
				],
			},
			{
				test: /font-awesome\.css$/,
				use:  [
					'style-loader',
					'css-loader',
					path.resolve(__dirname, './fa-only-woff-loader.js'),
				],
			},
			{
				test: /\.woff2?$/, // font-awesome icons
				use:  'url-loader',
			},
		],
	},
};
