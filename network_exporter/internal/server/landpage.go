package server

import (
	"fmt"
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
	const (
		landingPageCSS = `
/* 基础设置 */
body {
    font-family: 'Arial', sans-serif;
    font-size: 18px;
    line-height: 1.8;
    color: #ffffff;
    background: linear-gradient(45deg, #6e7dff, #00b5e2);
    margin: 0;
    padding: 0;
    text-align: center;
    transition: all 0.3s ease-in-out;
    min-height: 100vh;
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
}

/* 标题样式 */
h1 {
    font-size: 4.5em;
    font-weight: bold;
    margin: 20px 0;
    color: #fff;
    text-shadow: 3px 3px 15px rgba(0, 0, 0, 0.4);
    animation: fadeInDown 1s ease-in-out;
}

h2 {
    font-size: 2.8em;
    font-weight: 500;
    margin: 15px 0;
    color: #f0f0f0;
    text-shadow: 2px 2px 10px rgba(0, 0, 0, 0.3);
    animation: fadeInUp 1s ease-in-out;
}

/* 列表样式 */
ul {
    list-style: none;
    padding: 0;
    margin: 50px 0;
    display: flex;
    flex-direction: column;
    align-items: center;
}

ul li {
    width: 80%;
    max-width: 600px;
    background: rgba(255, 255, 255, 0.15);
    border-radius: 20px;
    margin: 15px 0;
    padding: 25px;
    font-size: 1.6em;
    backdrop-filter: blur(10px);
    transition: transform 0.4s ease, box-shadow 0.4s ease;
    cursor: pointer;
}

ul li:hover {
    transform: translateY(-5px) scale(1.05);
    box-shadow: 0 10px 30px rgba(0, 0, 0, 0.4);
}

/* 链接样式 */
a {
    color: #ffffff;
    font-weight: bold;
    font-size: 1.5em;
    padding: 12px 25px;
    border-radius: 12px;
    background: rgba(255, 255, 255, 0.15);
    display: inline-block;
    transition: background 0.3s ease, transform 0.3s ease;
}

a:hover {
    background: rgba(255, 255, 255, 0.25);
    transform: scale(1.1);
}

/* 段落样式 */
p {
    font-size: 1.5em;
    color: #e0e0e0;
    max-width: 800px;
    margin-top: 20px;
    text-align: center;
}

p.version {
    font-size: 1.3em;
    color: #f0f0f0;
    margin-top: 50px;
}

/* 按钮样式 */
button {
    font-size: 1.4em;
    padding: 12px 30px;
    border: none;
    border-radius: 12px;
    background: rgba(255, 255, 255, 0.2);
    color: white;
    cursor: pointer;
    transition: all 0.3s ease;
}

button:hover {
    background: rgba(255, 255, 255, 0.3);
    transform: scale(1.1);
}

/* 响应式优化 */
@media (max-width: 768px) {
    h1 {
        font-size: 3.5em;
    }
    h2 {
        font-size: 2.2em;
    }
    ul li {
        width: 90%;
        font-size: 1.4em;
    }
}

/* 动画效果 */
@keyframes fadeInDown {
    from {
        opacity: 0;
        transform: translateY(-20px);
    }
    to {
        opacity: 1;
        transform: translateY(0);
    }
}

@keyframes fadeInUp {
    from {
        opacity: 0;
        transform: translateY(20px);
    }
    to {
        opacity: 1;
        transform: translateY(0);
    }
}
/* 基础设置 */
body {
    font-family: 'Arial', sans-serif;
    font-size: 18px;
    line-height: 1.8;
    color: #ffffff;
    background: linear-gradient(45deg, #6e7dff, #00b5e2);
    margin: 0;
    padding: 0;
    text-align: center;
    transition: all 0.3s ease-in-out;
    min-height: 100vh;
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
    animation: gradientShift 8s infinite alternate ease-in-out;
}

/* 标题样式 */
h1 {
    font-size: 4.5em;
    font-weight: bold;
    margin: 20px 0;
    color: #fff;
    text-shadow: 3px 3px 15px rgba(0, 0, 0, 0.4);
    animation: bounce 3s infinite ease-in-out;
}

h2 {
    font-size: 2.8em;
    font-weight: 500;
    margin: 15px 0;
    color: #f0f0f0;
    text-shadow: 2px 2px 10px rgba(0, 0, 0, 0.3);
    animation: fadeInUp 2s infinite alternate ease-in-out;
}

/* 列表样式 */
ul {
    list-style: none;
    padding: 0;
    margin: 50px 0;
    display: flex;
    flex-direction: column;
    align-items: center;
}

ul li {
    width: 80%;
    max-width: 600px;
    background: rgba(255, 255, 255, 0.15);
    border-radius: 20px;
    margin: 15px 0;
    padding: 25px;
    font-size: 1.6em;
    backdrop-filter: blur(10px);
    transition: transform 0.4s ease, box-shadow 0.4s ease;
    cursor: pointer;
    animation: float 5s infinite alternate ease-in-out;
}

ul li:hover {
    transform: translateY(-5px) scale(1.05);
    box-shadow: 0 10px 30px rgba(0, 0, 0, 0.4);
}

/* 链接样式 */
a {
    color: #ffffff;
    font-weight: bold;
    font-size: 1.5em;
    padding: 12px 25px;
    border-radius: 12px;
    background: rgba(255, 255, 255, 0.15);
    display: inline-block;
    transition: background 0.3s ease, transform 0.3s ease;
    animation: pulse 3s infinite ease-in-out;
}

a:hover {
    background: rgba(255, 255, 255, 0.25);
    transform: scale(1.1);
}

/* 段落样式 */
p {
    font-size: 1.5em;
    color: #e0e0e0;
    max-width: 800px;
    margin-top: 20px;
    text-align: center;
}

p.version {
    font-size: 1.3em;
    color: #f0f0f0;
    margin-top: 50px;
}

/* 按钮样式 */
button {
    font-size: 1.4em;
    padding: 12px 30px;
    border: none;
    border-radius: 12px;
    background: rgba(255, 255, 255, 0.2);
    color: white;
    cursor: pointer;
    transition: all 0.3s ease;
    animation: pulse 4s infinite ease-in-out;
}

button:hover {
    background: rgba(255, 255, 255, 0.3);
    transform: scale(1.1);
}

/* 响应式优化 */
@media (max-width: 768px) {
    h1 {
        font-size: 3.5em;
    }
    h2 {
        font-size: 2.2em;
    }
    ul li {
        width: 90%;
        font-size: 1.4em;
    }
}

/* 动画效果 */
@keyframes gradientShift {
    from {
        background: linear-gradient(45deg, #6e7dff, #00b5e2);
    }
    to {
        background: linear-gradient(45deg, #00b5e2, #6e7dff);
    }
}

@keyframes bounce {
    0%, 100% {
        transform: translateY(0);
    }
    50% {
        transform: translateY(-10px);
    }
}

@keyframes fadeInUp {
    0%, 100% {
        opacity: 0.7;
        transform: translateY(10px);
    }
    50% {
        opacity: 1;
        transform: translateY(0);
    }
}

@keyframes float {
    from {
        transform: translateY(0);
    }
    to {
        transform: translateY(-10px);
    }
}

@keyframes pulse {
    0% {
        transform: scale(1);
    }
    50% {
        transform: scale(1.05);
    }
    100% {
        transform: scale(1);
    }
}

`
		landingPageHTML = `
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link rel="icon" type="image/x-icon" href="/favicon.ico">
	<title>{{.Name}}</title>
	<style>{{.CSS}}</style>
</head>
<body>
	<h1>
		{{.Name}}
	</h1>
	<ul>
		{{range .Links}}
			<li>
				<a href="{{.Address}}">
					{{.Text}}
				</a>
			</li>
		{{end}}
	</ul>
	<p class="version">
		Version: {{.Version}}
	</p>
</body>
</html>
`
	)

	if c.CSS == "" {
		c.CSS = landingPageCSS
	}

	tmpl, err := template.New("landingPage").Parse(landingPageHTML)
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
	if _, err := w.Write(h.landingPage); err != nil {
		fmt.Printf("The response to writing to the login page failed: %v\n", err)
	}
}
// Part 2 commit for network_exporter/internal/server/landpage.go
