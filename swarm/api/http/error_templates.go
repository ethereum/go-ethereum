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

/*
We use html templates to handle simple but as informative as possible error pages.

To eliminate circular dependency in case of an error, we don't store error pages on swarm.
We can't save the error pages as html files on disk, or when deploying compiled binaries
they won't be found.

For this reason we resort to save the HTML error pages as strings, which then can be
parsed by Go's html/template package
*/
package http

//This returns the HTML for generic errors
func GetGenericErrorPage() string {
	page := `
<html>

  <head>
    <meta http-equiv="Content-Type" content="text/html; charset=UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0">
    <meta http-equiv="X-UA-Compatible" ww="chrome=1">
    <meta name="description" content="Ethereum/Swarm error page">
    <meta property="og:url" content="https://swarm-gateways.net/bzz:/theswarm.eth">
		<link rel="shortcut icon" type="image/x-icon" href="data:image/png;base64,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"/>
    <style>

      body, div, header, footer {
        margin: 0;
        padding: 0;
      }

      body {
        overflow: hidden;
      }

      .container {
        min-width: 100%;
        min-height: 100%;
        max-height: 100%;
      }

      header {
        display: flex;
        align-items: center;
        background-color: #ffa500;
        /* height: 20vh; */
        padding: 5px;
      }

      .header-left, .header-right {
        width: 20%;
      }

      .header-left {
        padding-left: 40px;
        float: left;
      }

      .header-right {
        padding-right: 40px;
        float: right;
      }

      .page-title {
        /* margin-top: 4.5vh; */
        text-align: center;
        float:      left;
        width:      60%;
        color:      white;
      }

      content-body {
        display: block;
        margin: 0 auto;
        /* width: 50%; */
        min-height: 60vh;
        max-height: 60vh;
        padding: 50px 20px;
        opacity: 0.6;
        background-color: #A9F5BF;
      }

      table {
        font-size: 1.2em;
        margin: 0 auto;
      }

      tr {
        height: 60px;
      }

      td {
        text-align: center;
      }

      .key {
        color: #111;
        font-weight: bold;
        width: 200px;
      }

      .value {
        color: red;
        font-weight: bold
      }

      footer {
        height: 20vh;
        background-color: #ffa500;
        font-size: 1em;
        text-align: center;
        padding: 20px;
      }

    </style>

    <title>Swarm::HTTP Error Page</title>
  </head>


  <body>
    <div class="container">

      <header>
        <div class="header-left">
          <img style="height:18vh;margin-left:40px" src="data:image/png;base64,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"/>
        </div>
        <div class="page-title">
          <h1>There was a problem serving the requested page</h1>
        </div>
        <div class="header-right">
          <div id="timestamp">{{.Timestamp}}</div>
        </div>
      </header>

      <content-body>
        <section>
          <table>
            <thead>
              <td style="height: 150px; font-size: 1.3em; color: black; font-weight: bold">
                Hmmmmm....Swarm was not able to serve your request!
              </td>
            </thead>
            <tbody>
              <tr>
                <td class="key">
                  Error message:
                </td>
              </tr>
              <tr>
                <td class="value">
                  {{.Msg}}
                </td>
              </tr>

              <tr>
                <td class="key">
                  Error code:
                </td>
              </tr>
              <tr>
                <td class="value">
                  {{.Code}}
                </td>
              </tr>

            </tbody>
          </table>
        </section>
      </content-body>

      <footer>
        <p>
          Swarm: Serverless Hosting Incentivised Peer-To-Peer Storage And Content Distribution<br/>
          <a href="http://swarm-gateways.net/bzz:/theswarm.eth">Swarm</a>
        </p>
      </footer>


    </div>
  </body>

</html>
`
	return page
}

//This returns the HTML for a 404 Not Found error
func GetNotFoundErrorPage() string {
	page := `
<html>

  <head>
    <meta http-equiv="Content-Type" content="text/html; charset=UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0">
    <meta http-equiv="X-UA-Compatible" ww="chrome=1">
    <meta name="description" content="Ethereum/Swarm error page">
    <meta property="og:url" content="https://swarm-gateways.net/bzz:/theswarm.eth">
		<link rel="shortcut icon" type="image/x-icon" href="data:image/png;base64,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"/>
    <style>

      body, div, header, footer {
        margin: 0;
        padding: 0;
      }

      body {
        overflow: hidden;
      }

      .container {
        min-width: 100%;
        min-height: 100%;
        max-height: 100%;
      }

      header {
        display: flex;
        align-items: center;
        background-color: #ffa500;
        /* height: 20vh; */
        padding: 5px;
      }

      .header-left, .header-right {
        width: 20%;
      }

      .header-left {
        padding-left: 40px;
        float: left;
      }

      .header-right {
        padding-right: 40px;
        float: right;
      }

      .page-title {
        /* margin-top: 4.5vh; */
        text-align: center;
        float:      left;
        width:      60%;
        color:      white;
      }

      content-body {
        display: block;
        margin: 0 auto;
        /* width: 50%; */
        min-height: 60vh;
        max-height: 60vh;
        padding: 50px 20px;
        opacity: 0.6;
        background-color: #A9F5BF;
      }

      table {
        font-size: 1.2em;
        margin: 0 auto;
      }

      tr {
        height: 60px;
      }

      td {
        text-align: center;
      }

      .key {
        color: #111;
        font-weight: bold;
        width: 200px;
      }

      .value {
        color: red;
        font-weight: bold
      }

      footer {
        height: 20vh;
        background-color: #ffa500;
        font-size: 1em;
        text-align: center;
        padding: 20px;
      }

    </style>

    <title>Swarm::404 HTTP Not Found</title>
  </head>


  <body>
    <div class="container">

      <header>
        <div class="header-left">
          <img style="height:18vh;margin-left:40px" src="data:image/png;base64,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"/>
        </div>
        <div class="page-title">
          <h1>Resource Not Found</h1>
        </div>
        <div class="header-right">
          <div id="timestamp">{{.Timestamp}}</div>
        </div>
      </header>

      <content-body>
        <section>
          <table>
            <thead>
              <td style="height: 150px; font-size: 1.3em; color: black; font-weight: bold">
                Unfortunately, the resource you were trying to access could not be found on swarm.
              </td>
            </thead>
            <tbody>
              <tr>
                <td class="key">
                </td>
              </tr>
              <tr>
                <td class="value">
                  {{.Msg}}
                </td>
              </tr>

              <tr>
                <td class="key">
                  Error code:
                </td>
              </tr>
              <tr>
                <td class="value">
                  {{.Code}}
                </td>
              </tr>

            </tbody>
          </table>
        </section>
      </content-body>

      <footer>
        <p>
          Swarm: Serverless Hosting Incentivised Peer-To-Peer Storage And Content Distribution<br/>
          <a href="http://swarm-gateways.net/bzz:/theswarm.eth">Swarm</a>
        </p>
      </footer>


    </div>
  </body>

</html>
`
	return page
}

//This returns the HTML for a page listing disambiguation options
//i.e. if user requested bzz:/<hash>/read and the manifest contains "readme.md" and "readinglist.txt",
//this page is returned with a clickable list the existing disambiguation links in the manifest
func GetMultipleChoicesErrorPage() string {
	page := `
<html>

  <head>
    <meta http-equiv="Content-Type" content="text/html; charset=UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0">
    <meta http-equiv="X-UA-Compatible" ww="chrome=1">
    <meta name="description" content="Ethereum/Swarm multiple options page">
    <meta property="og:url" content="https://swarm-gateways.net/bzz:/theswarm.eth">
		<link rel="shortcut icon" type="image/x-icon" href="data:image/png;base64,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"/>
    <style>

      body, div, header, footer {
        margin: 0;
        padding: 0;
      }

      body {
        overflow: hidden;
      }

      .container {
        min-width: 100%;
        min-height: 100%;
        max-height: 100%;
      }

      header {
        display: flex;
        align-items: center;
        background-color: #ffa500;
        /* height: 20vh; */
        padding: 5px;
      }

      .header-left, .header-right {
        width: 20%;
      }

      .header-left {
        padding-left: 40px;
        float: left;
      }

      .header-right {
        padding-right: 40px;
        float: right;
      }

      .page-title {
        /* margin-top: 4.5vh; */
        text-align: center;
        float:      left;
        width:      60%;
        color:      white;
      }

      content-body {
        display: block;
        margin: 0 auto;
        /* width: 50%; */
        min-height: 60vh;
        max-height: 60vh;
        padding: 50px 20px;
        opacity: 0.6;
        background-color: #A9F5BF;
      }

      table {
        font-size: 1.2em;
        margin: 0 auto;
      }

      tr {
        height: 60px;
      }

      td {
        text-align: center;
      }

      .key {
        color: #111;
        font-weight: bold;
        width: 200px;
      }

      .value {
        color: red;
        font-weight: bold
      }

      footer {
        height: 20vh;
        background-color: #ffa500;
        font-size: 1em;
        text-align: center;
        padding: 20px;
      }

    </style>

    <title>Swarm::HTTP Disambiguation Page</title>
  </head>


  <body>
    <div class="container">

      <header>
        <div class="header-left">
          <img style="height:18vh;margin-left:40px" src="data:image/png;base64,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"/>
        </div>
        <div class="page-title">
          <h1>Swarm: disambiguation</h1>
        </div>
        <div class="header-right">
          <div id="timestamp">{{.Timestamp}}</div>
        </div>
      </header>

      <content-body>
        <section>
          <table>
            <thead>
              <td style="height: 150px; font-size: 1.3em; color: black; font-weight: bold">
                Your request yields ambiguous results!
              </td>
            </thead>
            <tbody>
              <tr>
                <td class="key">
                  Your request may refer to:
                </td>
              </tr>
              <tr>
                <td class="value">
                    {{ .Details}}
                </td>
              </tr>

              <tr>
                <td class="key">
                  Error code:
                </td>
              </tr>
              <tr>
                <td class="value">
                  {{.Code}}
                </td>
              </tr>

            </tbody>
          </table>
        </section>
      </content-body>

      <footer>
        <p>
          Swarm: Serverless Hosting Incentivised Peer-To-Peer Storage And Content Distribution<br/>
          <a href="http://swarm-gateways.net/bzz:/theswarm.eth">Swarm</a>
        </p>
      </footer>


    </div>
  </body>

</html>
`
	return page
}
