package server

import (
	"bytes"
	"net/http"
	"text/template"
)

type LandingPageConfig struct {
	CSS     string
	Name    string
	Links   []LandingPageLinks
	Version string
}

type LandingPageLinks struct {
	Address string
	Text    string
}

type LandingPageHandler struct {
	landingPage []byte
}

func NewLandingPage(c LandingPageConfig) (*LandingPageHandler, error) {
	// 使用简单的HTML模板，确保能正确显示所有链接
	const simplePageHTML = `
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>{{.Name}}</title>
	<style>
		body {
			font-family: Arial, sans-serif;
			margin: 40px;
			line-height: 1.6;
			color: #333;
			text-align: center;
		}
		h1 {
			color: #0078d4;
		}
		ul {
			list-style: none;
			padding: 0;
			margin: 30px 0;
		}
		li {
			margin: 15px 0;
		}
		a {
			display: inline-block;
			color: #0078d4;
			text-decoration: none;
			font-weight: bold;
			padding: 10px 20px;
			border: 1px solid #0078d4;
			border-radius: 4px;
			transition: all 0.3s;
		}
		a:hover {
			background-color: #0078d4;
			color: white;
		}
		.version {
			margin-top: 40px;
			font-size: 0.9em;
			color: #666;
		}
	</style>
</head>
<body>
	<h1>{{.Name}}</h1>
	<ul>
		{{range .Links}}
		<li><a href="{{.Address}}">{{.Text}}</a></li>
		{{end}}
	</ul>
	<p class="version">Version: {{.Version}}</p>
</body>
</html>
`
	tmpl, err := template.New("landingPage").Parse(simplePageHTML)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, c); err != nil {
		return nil, err
	}

	return &LandingPageHandler{
		landingPage: buf.Bytes(),
	}, nil
}

func (h *LandingPageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	_, _ = w.Write(h.landingPage)
}
