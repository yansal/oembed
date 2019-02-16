package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/yansal/oembed"
)

func main() {
	http.Handle("/", &handler{tmpl: template.Must(template.New("").Parse(`<html>
	<form><input name="url" placeholder="url"></form>
	{{if .HTML}}{{.HTML}}{{end}}
	{{if .Err}}<pre>{{printf "%+v" .Err}}</pre>{{end}}`))})
	http.Handle("/favicon.ico", http.NotFoundHandler())

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

type handler struct{ tmpl *template.Template }

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	url := r.FormValue("url")
	if url == "" {
		if err := h.tmpl.Execute(w, nil); err != nil {
			log.Print(err)
		}
		return
	}

	// TODO: validate url

	oembedData, err := oembed.Get(r.Context(), url)
	var tmplData struct {
		HTML template.HTML
		Err  error
	}
	if err != nil {
		// TODO: log error with stacktrace, remove stacktrace from template data
		tmplData.Err = err
	} else {
		tmplData.HTML, tmplData.Err = extractHTML(oembedData)
	}
	if err := h.tmpl.Execute(w, tmplData); err != nil {
		log.Print(err)
	}
}

func extractHTML(data oembed.Data) (template.HTML, error) {
	if data.HTML != "" {
		return template.HTML(data.HTML), nil
	}
	return "", fmt.Errorf("don't know what to do with data %+v", data)
}
