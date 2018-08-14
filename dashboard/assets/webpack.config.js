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
const UglifyJsPlugin = require('uglifyjs-webpack-plugin');
const path = require('path');

module.exports = {
	mode:   'development',
	target: 'web',
	entry:  {
		bundle: './index',
	},
	output: {
		filename: '[name].js',
		path:     path.resolve(__dirname, ''),
		// sourceMapFilename: '[file].map',
	},
	resolve: {
		modules: [
			'node_modules',
			path.resolve(__dirname, 'components'), // import './components/Component' -> import 'Component'
		],
		// alias: {
		// 	root: path.resolve(__dirname, ''),
		// },
		extensions: ['.js', '.jsx'],
	},
	// devtool:      'source-map',
	optimization: {
		minimize:     true,
		namedModules: true, // Module names instead of numbers - resolves the large diff problem.
		minimizer:    [
			new UglifyJsPlugin({
				uglifyOptions: {
					compress: true,
					output:   {
						comments: false,
						beautify: true,
					},
				},
				// sourceMap: true,
			}),
		],
	},
	plugins: [
		new webpack.DefinePlugin({
			PROD: process.env.NODE_ENV === 'production',
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
	devServer: {
		port:     8081,
		compress: true,
	},
};
