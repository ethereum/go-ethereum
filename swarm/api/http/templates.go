// Copyright 2016 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package http

import (
	"html/template"
	"path"

	"github.com/ethereum/go-ethereum/swarm/api"
)

type htmlListData struct {
	URI  *api.URI
	List *api.ManifestList
}

var htmlListTemplate = template.Must(template.New("html-list").Funcs(template.FuncMap{"basename": path.Base}).Parse(`
<!DOCTYPE html>
<html>
<head>
  <meta http-equiv="Content-Type" content="text/html; charset=utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Swarm index of {{ .URI }}</title>
</head>

<body>
  <h1>Swarm index of {{ .URI }}</h1>
  <hr>
  <table>
    <thead>
      <tr>
	<th>Path</th>
	<th>Type</th>
	<th>Size</th>
      </tr>
    </thead>

    <tbody>
      {{ range .List.CommonPrefixes }}
	<tr>
	  <td><a href="{{ basename . }}/?list=true">{{ basename . }}/</a></td>
	  <td>DIR</td>
	  <td>-</td>
	</tr>
      {{ end }}

      {{ range .List.Entries }}
	<tr>
	  <td><a href="{{ basename .Path }}">{{ basename .Path }}</a></td>
	  <td>{{ .ContentType }}</td>
	  <td>{{ .Size }}</td>
	</tr>
      {{ end }}
  </table>
  <hr>
</body>
`[1:]))
