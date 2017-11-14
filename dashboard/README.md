## Go Ethereum Dashboard

The dashboard is a data visualizer integrated into geth, intended to collect and visualize useful information of an Ethereum node. It consists of two parts:

* The client visualizes the collected data.
* The server collects the data, and updates the clients.

The client's UI uses [React][React] with JSX syntax, which is validated by the [ESLint][ESLint] linter mostly according to the [Airbnb React/JSX Style Guide][Airbnb]. The style is defined in the `.eslintrc` configuration file. The resources are bundled into a single `bundle.js` file using [Webpack][Webpack], which relies on the `webpack.config.js`. The bundled file is referenced from `dashboard.html` and takes part in the `assets.go` too. The necessary dependencies for the module bundler are gathered by [Node.js][Node.js].

### Development and bundling

As the dashboard depends on certain NPM packages (which are not included in the go-ethereum repo), these need to be installed first:

```
$ (cd dashboard/assets && npm install)
```

Normally the dashboard assets are bundled into Geth via `go-bindata` to avoid external dependencies. Rebuilding Geth after each UI modification however is not feasible from a developer perspective. Instead, we can run `webpack` in watch mode to automatically rebundle the UI, and ask `geth` to use external assets to not rely on compiled resources:

```
$ (cd dashboard/assets && ./node_modules/.bin/webpack --watch)
$ geth --dashboard --dashboard.assets=dashboard/assets/public --vmodule=dashboard=5
```

To bundle up the final UI into Geth, run `webpack` and `go generate`:

```
$ (cd dashboard/assets && ./node_modules/.bin/webpack)
$ go generate ./dashboard
```

### Have fun

[Webpack][Webpack] offers handy tools for visualizing the bundle's dependency tree and space usage.

* Generate the bundle's profile running `webpack --profile --json > stats.json`
* For the _dependency tree_ go to [Webpack Analyze][WA], and import `stats.json`
* For the _space usage_ go to [Webpack Visualizer][WV], and import `stats.json`

[React]: https://reactjs.org/
[ESLint]: https://eslint.org/
[Airbnb]: https://github.com/airbnb/javascript/tree/master/react
[Webpack]: https://webpack.github.io/
[WA]: http://webpack.github.io/analyse/
[WV]: http://chrisbateman.github.io/webpack-visualizer/
[Node.js]: https://nodejs.org/en/
