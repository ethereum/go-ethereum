/******/ (function(modules) { // webpackBootstrap
/******/ 	// The module cache
/******/ 	var installedModules = {};
/******/
/******/ 	// The require function
/******/ 	function __webpack_require__(moduleId) {
/******/
/******/ 		// Check if module is in cache
/******/ 		if(installedModules[moduleId]) {
/******/ 			return installedModules[moduleId].exports;
/******/ 		}
/******/ 		// Create a new module (and put it into the cache)
/******/ 		var module = installedModules[moduleId] = {
/******/ 			i: moduleId,
/******/ 			l: false,
/******/ 			exports: {}
/******/ 		};
/******/
/******/ 		// Execute the module function
/******/ 		modules[moduleId].call(module.exports, module, module.exports, __webpack_require__);
/******/
/******/ 		// Flag the module as loaded
/******/ 		module.l = true;
/******/
/******/ 		// Return the exports of the module
/******/ 		return module.exports;
/******/ 	}
/******/
/******/
/******/ 	// expose the modules object (__webpack_modules__)
/******/ 	__webpack_require__.m = modules;
/******/
/******/ 	// expose the module cache
/******/ 	__webpack_require__.c = installedModules;
/******/
/******/ 	// define getter function for harmony exports
/******/ 	__webpack_require__.d = function(exports, name, getter) {
/******/ 		if(!__webpack_require__.o(exports, name)) {
/******/ 			Object.defineProperty(exports, name, {
/******/ 				configurable: false,
/******/ 				enumerable: true,
/******/ 				get: getter
/******/ 			});
/******/ 		}
/******/ 	};
/******/
/******/ 	// getDefaultExport function for compatibility with non-harmony modules
/******/ 	__webpack_require__.n = function(module) {
/******/ 		var getter = module && module.__esModule ?
/******/ 			function getDefault() { return module['default']; } :
/******/ 			function getModuleExports() { return module; };
/******/ 		__webpack_require__.d(getter, 'a', getter);
/******/ 		return getter;
/******/ 	};
/******/
/******/ 	// Object.prototype.hasOwnProperty.call
/******/ 	__webpack_require__.o = function(object, property) { return Object.prototype.hasOwnProperty.call(object, property); };
/******/
/******/ 	// __webpack_public_path__
/******/ 	__webpack_require__.p = "";
/******/
/******/ 	// Load entry module and return exports
/******/ 	return __webpack_require__(__webpack_require__.s = 3);
/******/ })
/************************************************************************/
/******/ ([
/* 0 */
/***/ (function(module, exports) {

module.exports = Inferno;

/***/ }),
/* 1 */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
/* harmony export (binding) */ __webpack_require__.d(__webpack_exports__, "c", function() { return isNullOrUndefined; });
/* harmony export (binding) */ __webpack_require__.d(__webpack_exports__, "d", function() { return mapChildren; });
/* harmony export (binding) */ __webpack_require__.d(__webpack_exports__, "a", function() { return Clearfix; });
/* harmony export (binding) */ __webpack_require__.d(__webpack_exports__, "b", function() { return MEMORY_SAMPLE_LIMIT; });
/* unused harmony export TRAFFIC_SAMPLE_LIMIT */
/* harmony import */ var __WEBPACK_IMPORTED_MODULE_0_inferno__ = __webpack_require__(0);
/* harmony import */ var __WEBPACK_IMPORTED_MODULE_0_inferno___default = __webpack_require__.n(__WEBPACK_IMPORTED_MODULE_0_inferno__);
// isNullOrUndefined returns true if a variable is null or undefined.
var isNullOrUndefined = function isNullOrUndefined(variable) {
  return variable === null || typeof variable === 'undefined';
};

var mapChildren = function mapChildren(children, mapFunc) {
  return !Array.isArray(children) || children.length < 1 || children.length === 1 ? mapFunc(children) : children.map(mapFunc);
};


var Clearfix = function Clearfix() {
  return Object(__WEBPACK_IMPORTED_MODULE_0_inferno__["createVNode"])(2, "div", "clearfix");
};

var MEMORY_SAMPLE_LIMIT = 200; // Maximum number of memory data samples.
var TRAFFIC_SAMPLE_LIMIT = 200; // Maximum number of traffic data samples.

/***/ }),
/* 2 */
/***/ (function(module, exports) {

module.exports = Inferno.Component;

/***/ }),
/* 3 */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
Object.defineProperty(__webpack_exports__, "__esModule", { value: true });
/* harmony import */ var __WEBPACK_IMPORTED_MODULE_0_inferno__ = __webpack_require__(0);
/* harmony import */ var __WEBPACK_IMPORTED_MODULE_0_inferno___default = __webpack_require__.n(__WEBPACK_IMPORTED_MODULE_0_inferno__);
/* harmony import */ var __WEBPACK_IMPORTED_MODULE_1__components_Dashboard__ = __webpack_require__(4);



// Renders the whole dashboard.

__WEBPACK_IMPORTED_MODULE_0_inferno___default.a.render(Object(__WEBPACK_IMPORTED_MODULE_0_inferno__["createVNode"])(16, __WEBPACK_IMPORTED_MODULE_1__components_Dashboard__["a" /* default */]), document.getElementById('dashboard'));

/***/ }),
/* 4 */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
/* harmony import */ var __WEBPACK_IMPORTED_MODULE_0_component__ = __webpack_require__(2);
/* harmony import */ var __WEBPACK_IMPORTED_MODULE_0_component___default = __webpack_require__.n(__WEBPACK_IMPORTED_MODULE_0_component__);
/* harmony import */ var __WEBPACK_IMPORTED_MODULE_1__Common__ = __webpack_require__(1);
/* harmony import */ var __WEBPACK_IMPORTED_MODULE_2__SideBar__ = __webpack_require__(5);
/* harmony import */ var __WEBPACK_IMPORTED_MODULE_3__TopNavigation__ = __webpack_require__(6);
/* harmony import */ var __WEBPACK_IMPORTED_MODULE_4__PageContent__ = __webpack_require__(7);
/* harmony import */ var __WEBPACK_IMPORTED_MODULE_5__Footer__ = __webpack_require__(8);
/* harmony import */ var __WEBPACK_IMPORTED_MODULE_6_inferno__ = __webpack_require__(0);
/* harmony import */ var __WEBPACK_IMPORTED_MODULE_6_inferno___default = __webpack_require__.n(__WEBPACK_IMPORTED_MODULE_6_inferno__);
function _classCallCheck(instance, Constructor) { if (!(instance instanceof Constructor)) { throw new TypeError("Cannot call a class as a function"); } }

function _possibleConstructorReturn(self, call) { if (!self) { throw new ReferenceError("this hasn't been initialised - super() hasn't been called"); } return call && (typeof call === "object" || typeof call === "function") ? call : self; }

function _inherits(subClass, superClass) { if (typeof superClass !== "function" && superClass !== null) { throw new TypeError("Super expression must either be null or a function, not " + typeof superClass); } subClass.prototype = Object.create(superClass && superClass.prototype, { constructor: { value: subClass, enumerable: false, writable: true, configurable: true } }); if (superClass) Object.setPrototypeOf ? Object.setPrototypeOf(subClass, superClass) : subClass.__proto__ = superClass; }








// Dashboard is the main component, which renders the whole page,
// makes connection with the server and listens for messages.
// When there is an incoming message, updates the page's content correspondingly.


var Dashboard = function (_Component) {
    _inherits(Dashboard, _Component);

    function Dashboard(props) {
        _classCallCheck(this, Dashboard);

        var _this = _possibleConstructorReturn(this, _Component.call(this, props));

        _this.updateCharts = function (msg) {
            var memory = _this.state.charts.memory;
            var traffic = _this.state.charts.traffic;

            // Fill the dashboard with the past data. metrics is set only in the first msg,
            // after the connection is established.
            if (msg.metrics !== undefined) {
                // Clear the arrays to prevent data confusion with the previous connection.
                memory.labels = [];
                traffic.labels = [];
                memory.datasets[0].data = [];
                traffic.datasets[0].data = [];

                var mem = msg.metrics.memory;
                var traff = msg.metrics.processor; // TODO (kurkomisi): !!!

                // Put the past data to the beginning of the arrays. This prevents confusion with the next msg data,
                // which goes to the end.
                for (var i = mem.length - 1; i >= 0 && __WEBPACK_IMPORTED_MODULE_1__Common__["b" /* MEMORY_SAMPLE_LIMIT */] > memory.labels.length; --i) {
                    memory.labels.unshift(mem[i].time.substring(mem[i].time.length - 5));
                    traffic.labels.unshift(mem[i].time.substring(mem[i].time.length - 5));
                    memory.datasets[0].data.unshift(mem[i].value);
                    traffic.datasets[0].data.unshift(traff[i].value);
                }

                _this.setState({ charts: { memory: memory, traffic: traffic } }); // Update the components.
                return;
            }

            // Put the new data to the end of the arrays.
            if (msg.memory !== undefined) {
                // Remove the first elements in case the samples' amount exceeds the limit.
                if (memory.labels.length === __WEBPACK_IMPORTED_MODULE_1__Common__["b" /* MEMORY_SAMPLE_LIMIT */]) {
                    memory.labels.shift();
                    traffic.labels.shift();
                    memory.datasets[0].data.shift();
                    traffic.datasets[0].data.shift();
                }
                memory.labels.push(msg.memory.time.substring(msg.memory.time.length - 5));
                traffic.labels.push(msg.memory.time.substring(msg.memory.time.length - 5));
                memory.datasets[0].data.push(msg.memory.value);
                traffic.datasets[0].data.push(msg.processor.value);

                _this.setState({ charts: { memory: memory, traffic: traffic } }); // Update the components.
            }
        };

        _this.reconnect = function () {
            var server = new WebSocket("ws://" + location.host + "/api");

            server.onmessage = function (event) {
                var msg = JSON.parse(event.data);
                if (Object(__WEBPACK_IMPORTED_MODULE_1__Common__["c" /* isNullOrUndefined */])(msg)) {
                    return;
                }
                _this.updateCharts(msg);
            };

            server.onclose = function () {
                return setTimeout(_this.reconnect, 3000);
            };
        };

        _this.componentDidMount = function () {
            return _this.reconnect();
        };

        _this.render = function () {
            return Object(__WEBPACK_IMPORTED_MODULE_6_inferno__["createVNode"])(2, "div", "container body", Object(__WEBPACK_IMPORTED_MODULE_6_inferno__["createVNode"])(2, "div", "main_container", [Object(__WEBPACK_IMPORTED_MODULE_6_inferno__["createVNode"])(16, __WEBPACK_IMPORTED_MODULE_2__SideBar__["a" /* SideBar */]), Object(__WEBPACK_IMPORTED_MODULE_6_inferno__["createVNode"])(16, __WEBPACK_IMPORTED_MODULE_3__TopNavigation__["a" /* TopNavigation */]), Object(__WEBPACK_IMPORTED_MODULE_6_inferno__["createVNode"])(16, __WEBPACK_IMPORTED_MODULE_4__PageContent__["a" /* default */], null, null, {
                "charts": _this.state.charts
            }), Object(__WEBPACK_IMPORTED_MODULE_6_inferno__["createVNode"])(16, __WEBPACK_IMPORTED_MODULE_5__Footer__["a" /* Footer */])]));
        };

        _this.state = {
            charts: { // Stores the state of the charts.
                memory: {
                    labels: [],
                    datasets: [{
                        label: "system/memory/inuse",
                        backgroundColor: "rgba(38, 185, 154, 0.31)",
                        borderColor: "rgba(38, 185, 154, 0.7)",
                        pointBorderColor: "rgba(38, 185, 154, 0.7)",
                        pointBackgroundColor: "rgba(38, 185, 154, 0.7)",
                        pointHoverBackgroundColor: "#fff",
                        pointHoverBorderColor: "rgba(220,220,220,1)",
                        pointBorderWidth: 1,
                        data: []
                    }]
                },
                traffic: {
                    labels: [],
                    datasets: [{
                        label: "p2p/InboundTraffic",
                        backgroundColor: "rgba(3, 88, 106, 0.3)",
                        borderColor: "rgba(3, 88, 106, 0.70)",
                        pointBorderColor: "rgba(3, 88, 106, 0.70)",
                        pointBackgroundColor: "rgba(3, 88, 106, 0.70)",
                        pointHoverBackgroundColor: "#fff",
                        pointHoverBorderColor: "rgba(151,187,205,1)",
                        pointBorderWidth: 1,
                        data: []
                    }]
                }
            }
        };
        return _this;
    }

    // updateCharts Analyzes the incoming message, and updates the charts' content correspondingly.


    // reconnect establishes a websocket connection with the server, listens for incoming messages
    // and tries to reconnect on connection loss.


    // componentDidMount initiates the establishment of the first websocket connection after the component is rendered.


    // render renders the components of the dashboard.


    return Dashboard;
}(__WEBPACK_IMPORTED_MODULE_0_component___default.a);

/* harmony default export */ __webpack_exports__["a"] = (Dashboard);

/***/ }),
/* 5 */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
/* harmony export (binding) */ __webpack_require__.d(__webpack_exports__, "a", function() { return SideBar; });
/* harmony import */ var __WEBPACK_IMPORTED_MODULE_0_component__ = __webpack_require__(2);
/* harmony import */ var __WEBPACK_IMPORTED_MODULE_0_component___default = __webpack_require__.n(__WEBPACK_IMPORTED_MODULE_0_component__);
/* harmony import */ var __WEBPACK_IMPORTED_MODULE_1__Common__ = __webpack_require__(1);
/* harmony import */ var __WEBPACK_IMPORTED_MODULE_2_inferno__ = __webpack_require__(0);
/* harmony import */ var __WEBPACK_IMPORTED_MODULE_2_inferno___default = __webpack_require__.n(__WEBPACK_IMPORTED_MODULE_2_inferno__);
function _classCallCheck(instance, Constructor) { if (!(instance instanceof Constructor)) { throw new TypeError("Cannot call a class as a function"); } }

function _possibleConstructorReturn(self, call) { if (!self) { throw new ReferenceError("this hasn't been initialised - super() hasn't been called"); } return call && (typeof call === "object" || typeof call === "function") ? call : self; }

function _inherits(subClass, superClass) { if (typeof superClass !== "function" && superClass !== null) { throw new TypeError("Super expression must either be null or a function, not " + typeof superClass); } subClass.prototype = Object.create(superClass && superClass.prototype, { constructor: { value: subClass, enumerable: false, writable: true, configurable: true } }); if (superClass) Object.setPrototypeOf ? Object.setPrototypeOf(subClass, superClass) : subClass.__proto__ = superClass; }




// MenuItem renders an item for a Menu component and the belonging submenu items, if there is any.


var MenuItem = function (_Component) {
    _inherits(MenuItem, _Component);

    function MenuItem() {
        var _temp, _this, _ret;

        _classCallCheck(this, MenuItem);

        for (var _len = arguments.length, args = Array(_len), _key = 0; _key < _len; _key++) {
            args[_key] = arguments[_key];
        }

        return _ret = (_temp = (_this = _possibleConstructorReturn(this, _Component.call.apply(_Component, [this].concat(args))), _this), _this.render = function () {
            return Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "li", null, [Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "a", null, [Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "i", "fa " + _this.props.className), _this.props.text, Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "div", "fa fa-chevron-down")]),
            // Render dropdown menu only if there are children.
            Object(__WEBPACK_IMPORTED_MODULE_1__Common__["c" /* isNullOrUndefined */])(_this.props.children) || Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "ul", "nav child_menu", Object(__WEBPACK_IMPORTED_MODULE_1__Common__["d" /* mapChildren */])(_this.props.children, function (child) {
                return Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "li", null, child);
            }))]);
        }, _temp), _possibleConstructorReturn(_this, _ret);
    }

    return MenuItem;
}(__WEBPACK_IMPORTED_MODULE_0_component___default.a);

// Menu renders a menu component.


var Menu = function Menu() {
    return Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "ul", "nav side-menu", [Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(16, MenuItem, null, null, {
        "className": "fa-home",
        "text": "Home",
        children: [Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "a", null, "Dashboard1", {
            "href": "dashboard1.html"
        }), Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "a", null, "Dashboard2", {
            "href": "dashboard2.html"
        })]
    }), Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(16, MenuItem, null, null, {
        "className": "fa-edit",
        "text": "Networking",
        children: Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "a", null, "Networking", {
            "href": "networking.html"
        })
    }), Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(16, MenuItem, null, null, {
        "className": "fa-desktop",
        "text": "Txpool",
        children: Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "a", null, "Txpool", {
            "href": "txpool.html"
        })
    }), Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(16, MenuItem, null, null, {
        "className": "fa-table",
        "text": "Logs",
        children: Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "a", null, "Logs", {
            "href": "logs.html"
        })
    }), Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(16, MenuItem, null, null, {
        "className": "fa-clone",
        "text": "Blockchain",
        children: [Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "a", null, "Blockchain1", {
            "href": "blockchain1.html"
        }), Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "a", null, "Blockchain2", {
            "href": "blockchain2.html"
        }), Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "a", null, "Blockchain3", {
            "href": "blockchain3.html"
        }), Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "a", null, "Blockchain4", {
            "href": "blockchain4.html"
        }), Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "a", null, "Blockchain5", {
            "href": "blockchain5.html"
        })]
    }), Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(16, MenuItem, null, null, {
        "className": "fa-bar-chart-o",
        "text": "System"
    })]);
};

// SideBar renders a sidebar component.
var SideBar = function SideBar() {
    return Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "div", "col-md-3 left_col", Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "div", "left_col scroll-view", [Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "div", "navbar nav_title", Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "a", "site_title", [Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "i", "fa fa-paw"), Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "span", null, "Go Ethereum Dashboard")], {
        "href": "dashboard.html"
    }), {
        "style": { border: 0 }
    }), Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(16, __WEBPACK_IMPORTED_MODULE_1__Common__["a" /* Clearfix */]), Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "div", "main_menu_side hidden-print main_menu", Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "div", "menu_section", Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(16, Menu)), {
        "id": "sidebar-menu"
    })]));
};

/***/ }),
/* 6 */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
/* harmony export (binding) */ __webpack_require__.d(__webpack_exports__, "a", function() { return TopNavigation; });
/* harmony import */ var __WEBPACK_IMPORTED_MODULE_0_inferno__ = __webpack_require__(0);
/* harmony import */ var __WEBPACK_IMPORTED_MODULE_0_inferno___default = __webpack_require__.n(__WEBPACK_IMPORTED_MODULE_0_inferno__);

// TopNavigation renders a top navigation component.
var TopNavigation = function TopNavigation() {
    return Object(__WEBPACK_IMPORTED_MODULE_0_inferno__["createVNode"])(2, "div", "top_nav", Object(__WEBPACK_IMPORTED_MODULE_0_inferno__["createVNode"])(2, "div", "nav_menu", Object(__WEBPACK_IMPORTED_MODULE_0_inferno__["createVNode"])(2, "nav", "", Object(__WEBPACK_IMPORTED_MODULE_0_inferno__["createVNode"])(2, "div", "nav toggle", Object(__WEBPACK_IMPORTED_MODULE_0_inferno__["createVNode"])(2, "a", null, [" ", Object(__WEBPACK_IMPORTED_MODULE_0_inferno__["createVNode"])(2, "i", "fa fa-bars")], {
        "id": "_______menu_toggle"
    })), {
        "role": "navigation"
    })));
};

/***/ }),
/* 7 */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
/* harmony import */ var __WEBPACK_IMPORTED_MODULE_0_component__ = __webpack_require__(2);
/* harmony import */ var __WEBPACK_IMPORTED_MODULE_0_component___default = __webpack_require__.n(__WEBPACK_IMPORTED_MODULE_0_component__);
/* harmony import */ var __WEBPACK_IMPORTED_MODULE_1__Common__ = __webpack_require__(1);
/* harmony import */ var __WEBPACK_IMPORTED_MODULE_2_inferno__ = __webpack_require__(0);
/* harmony import */ var __WEBPACK_IMPORTED_MODULE_2_inferno___default = __webpack_require__.n(__WEBPACK_IMPORTED_MODULE_2_inferno__);
function _classCallCheck(instance, Constructor) { if (!(instance instanceof Constructor)) { throw new TypeError("Cannot call a class as a function"); } }

function _possibleConstructorReturn(self, call) { if (!self) { throw new ReferenceError("this hasn't been initialised - super() hasn't been called"); } return call && (typeof call === "object" || typeof call === "function") ? call : self; }

function _inherits(subClass, superClass) { if (typeof superClass !== "function" && superClass !== null) { throw new TypeError("Super expression must either be null or a function, not " + typeof superClass); } subClass.prototype = Object.create(superClass && superClass.prototype, { constructor: { value: subClass, enumerable: false, writable: true, configurable: true } }); if (superClass) Object.setPrototypeOf ? Object.setPrototypeOf(subClass, superClass) : subClass.__proto__ = superClass; }




// Chart name is already in use.
// ChartComponent renders a chart component and updates it, when the related data changes.


var ChartComponent = function (_Component) {
    _inherits(ChartComponent, _Component);

    function ChartComponent(props) {
        _classCallCheck(this, ChartComponent);

        var _this = _possibleConstructorReturn(this, _Component.call(this, props));

        _this.componentDidMount = function () {
            return _this.state.chart = new Chart(_this.data, {
                type: _this.props.type,
                data: _this.props.data
            });
        };

        _this.render = function () {
            if (!Object(__WEBPACK_IMPORTED_MODULE_1__Common__["c" /* isNullOrUndefined */])(_this.state.chart)) {
                _this.state.chart.update();
            }

            return Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "div", _this.props.className, Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "div", "x_panel", [Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "div", "x_title", [Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "h2", null, _this.props.text), Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "ul", "nav navbar-right panel_toolbox", [Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "li", null, Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "a", "collapse-link", Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "i", "fa fa-chevron-up"))),
            // Render dropdown menu only if there are children.
            Object(__WEBPACK_IMPORTED_MODULE_1__Common__["c" /* isNullOrUndefined */])(_this.props.children) || Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "li", "dropdown", [Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "a", "dropdown-toggle", Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "i", "fa fa-wrench"), {
                "href": "#",
                "data-toggle": "dropdown",
                "role": "button",
                "aria-expanded": "false"
            }), Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "ul", "dropdown-menu", Object(__WEBPACK_IMPORTED_MODULE_1__Common__["d" /* mapChildren */])(_this.props.children, function (child) {
                return Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "li", null, child);
            }), {
                "role": "menu"
            })]), Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "li", null, Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "a", "close-link", Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "i", "fa fa-close")))]), Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(16, __WEBPACK_IMPORTED_MODULE_1__Common__["a" /* Clearfix */])]), Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "div", "x_content", Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "canvas", null, null, null, null, function (data) {
                _this.data = data;
            }))]));
        };

        _this.state = {};
        return _this;
    }

    return ChartComponent;
}(__WEBPACK_IMPORTED_MODULE_0_component___default.a);

// Row renders a row component of charts only if there is any chart.


var Row = function (_Component2) {
    _inherits(Row, _Component2);

    function Row() {
        var _temp, _this2, _ret;

        _classCallCheck(this, Row);

        for (var _len = arguments.length, args = Array(_len), _key = 0; _key < _len; _key++) {
            args[_key] = arguments[_key];
        }

        return _ret = (_temp = (_this2 = _possibleConstructorReturn(this, _Component2.call.apply(_Component2, [this].concat(args))), _this2), _this2.render = function () {
            return Object(__WEBPACK_IMPORTED_MODULE_1__Common__["c" /* isNullOrUndefined */])(_this2.props.children) || Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "div", "row", [" ", _this2.props.children, " "]);
        }, _temp), _possibleConstructorReturn(_this2, _ret);
    }

    return Row;
}(__WEBPACK_IMPORTED_MODULE_0_component___default.a);

// PageContent renders a component for the page content.


var PageContent = function (_Component3) {
    _inherits(PageContent, _Component3);

    function PageContent() {
        var _temp2, _this3, _ret2;

        _classCallCheck(this, PageContent);

        for (var _len2 = arguments.length, args = Array(_len2), _key2 = 0; _key2 < _len2; _key2++) {
            args[_key2] = arguments[_key2];
        }

        return _ret2 = (_temp2 = (_this3 = _possibleConstructorReturn(this, _Component3.call.apply(_Component3, [this].concat(args))), _this3), _this3.render = function () {
            return Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "div", "right_col", Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "div", "", [Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "div", "page-title", Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "div", "title_left", Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "h3", null, ["Go Ethereum Dashboard", Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "small", null, "Statistics")]))), Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(16, __WEBPACK_IMPORTED_MODULE_1__Common__["a" /* Clearfix */]), Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(16, Row, null, null, {
                children: [Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(16, ChartComponent, null, null, {
                    "className": "col-md-6 col-sm-6 col-xs-12",
                    "text": "Memory usage system/memory/inuse",
                    "type": "line",
                    "data": _this3.props.charts.memory,
                    children: [Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "a", null, "Settings 1", {
                        "href": "#"
                    }), Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(2, "a", null, "Settings 2", {
                        "href": "#"
                    })]
                }), Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(16, ChartComponent, null, null, {
                    "className": "col-md-6 col-sm-6 col-xs-12",
                    "text": "Inbound traffic p2p/InboundTraffic",
                    "type": "line",
                    "data": _this3.props.charts.traffic
                })]
            }), Object(__WEBPACK_IMPORTED_MODULE_2_inferno__["createVNode"])(16, __WEBPACK_IMPORTED_MODULE_1__Common__["a" /* Clearfix */])]), {
                "role": "main"
            });
        }, _temp2), _possibleConstructorReturn(_this3, _ret2);
    }

    return PageContent;
}(__WEBPACK_IMPORTED_MODULE_0_component___default.a);

/* harmony default export */ __webpack_exports__["a"] = (PageContent);

/***/ }),
/* 8 */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
/* harmony export (binding) */ __webpack_require__.d(__webpack_exports__, "a", function() { return Footer; });
/* harmony import */ var __WEBPACK_IMPORTED_MODULE_0__Common__ = __webpack_require__(1);
/* harmony import */ var __WEBPACK_IMPORTED_MODULE_1_inferno__ = __webpack_require__(0);
/* harmony import */ var __WEBPACK_IMPORTED_MODULE_1_inferno___default = __webpack_require__.n(__WEBPACK_IMPORTED_MODULE_1_inferno__);


// Footer renders a footer component.

var Footer = function Footer() {
    return Object(__WEBPACK_IMPORTED_MODULE_1_inferno__["createVNode"])(2, "footer", null, [Object(__WEBPACK_IMPORTED_MODULE_1_inferno__["createVNode"])(2, "div", "pull-right", "Copyright 2017 The go-ethereum Authors"), Object(__WEBPACK_IMPORTED_MODULE_1_inferno__["createVNode"])(16, __WEBPACK_IMPORTED_MODULE_0__Common__["a" /* Clearfix */])]);
};

/***/ })
/******/ ]);