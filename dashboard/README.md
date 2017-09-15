## Go Ethereum Dashboard
### Description

The dashboard is a data visualizer integrated into geth, intended to collect and visualize useful information of an Ethereum node. 
The dashboard consists of two parts:
* The server listens to connections, collects data with a given refresh rate, and updates the dashboards through the opened connections.
* The client waits for update messages, updates the content and tries to reconnect on connection loss.

### Users
#### Installation steps

1. `cd .../go-ethereum/`
1. `go install -v ./cmd/geth`
1. Run the server with `geth --rinkeby --dashboard --vmodule=dashboard=5 --metrics`.
1. Enter `localhost:8080` (or change the configuration).

### Developers

The client's UI is maintained by [Inferno][Inferno], a sympathetic React-like JavaScript library.
In order to create the Inferno's virtual DOM using JSX syntax, babel plugin is required.

[Webpack module bundler][Webpack] is used for bundling the resources in order to gain cost efficiency and maintainability.
The resources will be bundled into a single JS file (`bundle.js`), which can be then referenced from the main html file. 
Finally this JS file will also take part in the `assets.go`.

[Node.js][Node.js] is used for installing the necessary dependencies for the module bundler.

#### Installation steps

_Module bundler_

1. `cd .../go-ethereum/dashboard/bundler/`
1. `npm install`
1. `./node_modules/.bin/webpack` // check out `webpack.config.js`

_Server_

1. Bundle the resources.
1. `cd .../go-ethereum/`.
1. `go generate ./dashboard && go install -v ./cmd/geth`.
1. Run the server with `geth --rinkeby --dashboard --vmodule=dashboard=5 --metrics console`.
    * Optionally use `--dashboard.assets=<path>` to set the assets' path (e.g. `--dashboard.assets=".../go-ethereum/dashboard/assets"`). 
Using this flag it is enough to only bundle the resources with webpack and refresh the page.
There is no need for stopping the server and regenerating the `assets.go` on every change of the UI.
1. Enter `localhost:8080` (or change the configuration).

#### Tools
[Webpack][Webpack] offers great tools for visualizing the bundle's dependency tree and space usage.

* Generate the bundle's profile by running `webpack --profile --json > stats.json`
* For the _dependency tree_ go to [Webpack Analyze][WA], and import `stats.json`
* For the _space usage_ go to [Webpack Visualizer][WV], and import `stats.json`

[Inferno]: https://infernojs.org/
[Webpack]: https://webpack.github.io/
[WA]: http://webpack.github.io/analyse/
[WV]: http://chrisbateman.github.io/webpack-visualizer/
[Node.js]: https://nodejs.org/en/