// Copyright 2018 The go-ethereum Authors
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

const path = require('path');

module.exports = {
	target: 'web',
	entry:  {
		bundle: './index',
	},
	output: {
		filename:          '[name].js',
		path:              path.resolve(__dirname, ''),
		sourceMapFilename: '[file].map',
	},
	resolve: {
		modules: [
			'node_modules',
			path.resolve(__dirname, 'components'), // import './components/Component' -> import 'Component'
		],
		extensions: ['.js', '.jsx'],
	},
	module: {
		rules: [
			{
				test:    /\.jsx$/, // regexp for JSX files
				exclude: /node_modules/,
				use:     [ // order: from bottom to top
					{
						loader:  'babel-loader',
						options: {
							presets: [ // order: from bottom to top
								'@babel/env',
								'@babel/react',
							],
							plugins: [ // order: from top to bottom
								'@babel/proposal-function-bind', // instead of stage 0
								'@babel/proposal-class-properties', // static defaultProps
								'@babel/transform-flow-strip-types',
								'react-hot-loader/babel',
							],
						},
					},
					// 'eslint-loader', // show errors in the console
				],
			},
			{
				test:  /\.css$/,
				oneOf: [
					{
						test: /font-awesome/,
						use:  [
							'style-loader',
							'css-loader',
							path.resolve(__dirname, './fa-only-woff-loader.js'),
						],
					},
					{
						use: [
							'style-loader',
							'css-loader',
						],
					},
				],
			},
			{
				test: /\.woff2?$/, // font-awesome icons
				use:  'url-loader',
			},
		],
	},
};
