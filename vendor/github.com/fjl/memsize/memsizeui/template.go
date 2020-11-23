package memsizeui

import (
	"html/template"
	"strconv"
	"sync"

	"github.com/fjl/memsize"
)

var (
	base         *template.Template // the "base" template
	baseInitOnce sync.Once
)

func baseInit() {
	base = template.Must(template.New("base").Parse(`<!DOCTYPE html>
<html>
	<head>
		<meta charset="UTF-8">
		<title>memsize</title>
		<style>
		body {
			 font-family: sans-serif;
		}
		button, .button {
			 display: inline-block;
			 font-weight: bold;
			 color: black;
			 text-decoration: none;
			 font-size: inherit;
			 padding: 3pt;
			 margin: 3pt;
			 background-color: #eee;
			 border: 1px solid #999;
			 border-radius: 2pt;
		}
		form.inline {
			display: inline-block;
		}
		</style>
	</head>
	<body>
		{{template "content" .}}
	</body>
</html>`))

	base.Funcs(template.FuncMap{
		"quote":     strconv.Quote,
		"humansize": memsize.HumanSize,
	})

	template.Must(base.New("rootbuttons").Parse(`
<a class="button" href="{{$.Link ""}}">Overview</a>
{{- range $root := .Roots -}}
<form class="inline" method="POST" action="{{$.Link "scan?root=" $root}}">
	<button type="submit">Scan {{quote $root}}</button>
</form>
{{- end -}}`))
}

func contentTemplate(source string) *template.Template {
	baseInitOnce.Do(baseInit)
	t := template.Must(base.Clone())
	template.Must(t.New("content").Parse(source))
	return t
}

var rootTemplate = contentTemplate(`
<h1>Memsize</h1>
{{template "rootbuttons" .}}
<hr/>
<h3>Reports</h3>
<ul>
	{{range .Reports}}
		<li><a href="{{printf "%d" | $.Link "report/"}}">{{quote .RootName}} @ {{.Date}}</a></li>
	{{else}}
		No reports yet, hit a scan button to create one.
	{{end}}
</ul>
`)

var notFoundTemplate = contentTemplate(`
<h1>{{.Data}}</h1>
{{template "rootbuttons" .}}
`)

var reportTemplate = contentTemplate(`
{{- $report := .Data -}}
<h1>Memsize Report {{$report.ID}}</h1>
<form method="POST" action="{{$.Link "scan?root=" $report.RootName}}">
	<a class="button" href="{{$.Link ""}}">Overview</a>
	<button type="submit">Scan Again</button>
</form>
<pre>
Root: {{quote $report.RootName}}
Date: {{$report.Date}}
Duration: {{$report.Duration}}
Bitmap Size: {{$report.Sizes.BitmapSize | humansize}}
Bitmap Utilization: {{$report.Sizes.BitmapUtilization}}
</pre>
<hr/>
<pre>
{{$report.Sizes.Report}}
</pre>
`)
