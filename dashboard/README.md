## Go Ethereum Dashboard
### Description

The GED is a data visualizer integrated into geth, intended to collect and visualize useful information of an Ethereum node. 
The GED consists of two parts:
* The server listens to connections, collects data with a given refresh rate, and updates the dashboards through the opened connections.
* The client waits for update messages, updates the content and tries to reconnect on connection loss.

The client's UI is maintained by [Inferno][Inferno], a sympathetic React-like JavaScript library.
In order to create the Inferno's virtual DOM using JSX syntax, there is need to use the provided babel plugin.
[Webpack][Webpack] is used for bundling the resources in order to gain cost efficiency and maintainability.
[Node.js][Node.js] is used for installing the necessary dependencies.

### Installation steps

_Server_

1. `cd .../go-ethereum/`
1. `go generate ./dashboard && go install -v ./cmd/geth`
1. run the server with `geth --rinkeby --dashboard --vmodule=dashboard=5 --metrics console`
1. optionally use `--dashboard.assets=<path>` to set the assets' path => useful for debugging
1. enter `localhost:8080` (or change the configuration)

_Module bundler_

1. `cd .../go-ethereum/dashboard/bundler/`
1. `npm install`
1. `./node_modules/.bin/webpack` // check out `webpack.config.js`

[Inferno]: https://infernojs.org/
[Webpack]: https://webpack.github.io/
[Node.js]: https://nodejs.org/en/