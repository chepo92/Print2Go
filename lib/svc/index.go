package svc

import (
	"html/template"
	"net/http"
	"os"

	_ "embed"
)

var (
	//go:embed index.tmpl
	ibuf      string
	indexTmpl = template.Must(template.New("index").Delims("[[", "]]").Parse(ibuf))
)

func indexPage(w http.ResponseWriter, r *http.Request) {
	hn, err := os.Hostname()
	if err != nil {
		hn = "<unknown>"
	}

	indexTmpl.Execute(w, struct{ Hostname string }{Hostname: hn})
}
