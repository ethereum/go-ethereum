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
