var webpack = require('webpack'); // eslint-disable-line

var env = process.env.NODE_ENV;   // eslint-disable-line
var filename = 'number-to-bn';      // eslint-disable-line
var library = 'numberToBN';          // eslint-disable-line
var config = {
  devtool: 'cheap-module-source-map',
  entry: [
    './src/index.js',
  ],
  output: {
    path: 'dist',
    filename: filename + '.js',       // eslint-disable-line
    library: library,                 // eslint-disable-line
    libraryTarget: 'umd',
    umdNamedDefine: true,
  },
  plugins: [
    new webpack.BannerPlugin({ banner: ' /* eslint-disable */ ', raw: true, entryOnly: true }),
    new webpack.BannerPlugin({ banner: ' /* eslint-disable */ ', raw: true }),
    new webpack.optimize.OccurrenceOrderPlugin(),
    new webpack.DefinePlugin({
      'process.env.NODE_ENV': JSON.stringify(env),
    }),
  ],
};

if (env === 'production') {
  config.output.filename = filename + '.min.js'; // eslint-disable-line
  config.plugins
  .push(new webpack.optimize.UglifyJsPlugin({
    compressor: {
      pure_getters: true,
      unsafe: true,
      unsafe_comps: true,
      warnings: false,
      screw_ie8: false,
    },
    mangle: {
      screw_ie8: false,
    },
    output: {
      screw_ie8: false,
    },
  }));
  config.plugins.push(new webpack.optimize.DedupePlugin());
}

module.exports = config;
