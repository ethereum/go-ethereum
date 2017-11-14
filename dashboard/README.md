## Go Ethereum Dashboard

The dashboard is a data visualizer integrated into geth, intended to collect and visualize useful information of an Ethereum node. 
Consists of two parts:
* The client visualizes the collected data.
* The server collects the data, and updates the clients.

The client's UI uses [React][React] with JSX syntax, which is validated by the [ESLint][ESLint] linter 
mostly according to the [Airbnb React/JSX Style Guide][Airbnb]. The style is defined in the `.eslintrc` configuration file.
The resources are bundled into a single `bundle.js` file using [Webpack][Webpack], which relies on the `webpack.config.js`.
The bundled file is referenced from `dashboard.html` and takes part in the `assets.go` too.
The necessary dependencies for the module bundler are gathered by [Node.js][Node.js].

### Install and run the server

1. `go generate ./dashboard && go install -v ./cmd/geth`.
1. `geth --dashboard --vmodule=dashboard=5`.

During the development use the `--dashboard.assets=<absolute path>` flag to set the assets' path 
(e.g. `geth --rinkeby --dashboard --dashboard.assets="<path>/dashboard/assets/public" --vmodule=dashboard=5 console`).
This way there is no need to stop and regenerate the server to modify the client.

### Install the module bundler

1. `cd dashboard/assets`
1. `npm install`

### Bundle the resources

1. `cd dashboard/assets`
1. `./node_modules/.bin/webpack`
1. Enter `localhost:8080` to check the result

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
