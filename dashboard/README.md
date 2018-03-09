## Go Ethereum Dashboard

The dashboard is a data visualizer integrated into geth, intended to collect and visualize useful information of an Ethereum node. It consists of two parts:

* The client visualizes the collected data.
* The server collects the data, and updates the clients.

The client's UI uses [React][React] with JSX syntax, which is validated by the [ESLint][ESLint] linter mostly according to the [Airbnb React/JSX Style Guide][Airbnb]. The style is defined in the `.eslintrc` configuration file. The resources are bundled into a single `bundle.js` file using [Webpack][Webpack], which relies on the `webpack.config.js`. The bundled file is referenced from `dashboard.html` and takes part in the `assets.go` too. The necessary dependencies for the module bundler are gathered by [Node.js][Node.js].

### Development and bundling

As the dashboard depends on certain NPM packages (which are not included in the `go-ethereum` repo), these need to be installed first:

```
$ (cd dashboard/assets && yarn install && yarn flow)
```

Normally the dashboard assets are bundled into Geth via `go-bindata` to avoid external dependencies. Rebuilding Geth after each UI modification however is not feasible from a developer perspective. Instead, we can run `yarn dev` to watch for file system changes and refresh the browser automatically.

```
$ geth --dashboard --vmodule=dashboard=5
$ (cd dashboard/assets && yarn dev)
```

To bundle up the final UI into Geth, run `go generate`:

```
$ (cd dashboard && go generate)
```

### Static type checking

Since JavaScript doesn't provide type safety, [Flow][Flow] is used to check types. These are only useful during development, so at the end of the process Babel will strip them.

To take advantage of static type checking, your IDE needs to be prepared for it. In case of [Atom][Atom] a configuration guide can be found [here][Atom config]: Install the [Nuclide][Nuclide] package for Flow support, making sure it installs all of its support packages by enabling `Install Recommended Packages on Startup`, and set the path of the `flow-bin` which were installed previously by `yarn`.

For more IDE support install the `linter-eslint` package too, which finds the `.eslintrc` file, and provides real-time linting. Atom warns, that these two packages are incompatible, but they seem to work well together. For third-party library errors and auto-completion [flow-typed][flow-typed] is used.

### Have fun

[Webpack][Webpack] offers handy tools for visualizing the bundle's dependency tree and space usage.

* Generate the bundle's profile running `yarn stats`
* For the _dependency tree_ go to [Webpack Analyze][WA], and import `stats.json`
* For the _space usage_ go to [Webpack Visualizer][WV], and import `stats.json`

[React]: https://reactjs.org/
[ESLint]: https://eslint.org/
[Airbnb]: https://github.com/airbnb/javascript/tree/master/react
[Webpack]: https://webpack.github.io/
[WA]: http://webpack.github.io/analyse/
[WV]: http://chrisbateman.github.io/webpack-visualizer/
[Node.js]: https://nodejs.org/en/
[Flow]: https://flow.org/
[Atom]: https://atom.io/
[Atom config]: https://medium.com/@fastphrase/integrating-flow-into-a-react-project-fbbc2f130eed
[Nuclide]: https://nuclide.io/docs/quick-start/getting-started/
[flow-typed]: https://github.com/flowtype/flow-typed
