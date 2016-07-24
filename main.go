package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/russross/blackfriday"
)

const port = 8080

var dir = flag.String("dir", "", "Working directory.")

// parseFlags parses flags and makes sure they are legit.
func parseFlags() error {
	flag.Parse()
	file, err := os.Open(*dir)
	if err != nil {
		return fmt.Errorf("failed to open path %q", *dir)
	}
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("unable to get file stats for %q", *dir)
	}
	if !fileInfo.IsDir() {
		return fmt.Errorf("the provided path %q is not a directory", *dir)
	}
	return nil
}

// printWorkingDirectoryList lists markdown files in working directory.
func printWorkingDirectoryList(w http.ResponseWriter) error {
	// List files in working directory.
	entries, err := ioutil.ReadDir(*dir)
	if err != nil {
		return fmt.Errorf("failed to list files in working directory %q, make sure it has not been deleted: %s", *dir, err)
	}
	// Filter markdown files.
	var markdowns []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".md" {
			markdowns = append(markdowns, entry.Name())
		}
	}
	// Build catalog in markdown.
	var buffer bytes.Buffer
	if len(markdowns) == 0 {
		buffer.WriteString("No markdown file (.md) has been found in working directory.")
	} else {
		buffer.WriteString("## Catalog:\n")
		for _, markdown := range markdowns {
			fmt.Fprintf(&buffer, "* [%s](./%s)\n", markdown, markdown)
		}
	}
	return serveMarkdown(w, buffer.Bytes())
}

// serveMarkdownFromRequestPath renders the markdown file requested in URL as HTML.
func serveMarkdownFromRequestPath(w http.ResponseWriter, r *http.Request) error {
	localPath := filepath.Join(*dir, r.URL.Path)
	rawFile, err := ioutil.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read local file, make sure %q exists", localPath)
	}
	return serveMarkdown(w, rawFile)
}

// serveMarkdown renders passed markdown bytes as HTML.
func serveMarkdown(w http.ResponseWriter, markdown []byte) error {
	html := blackfriday.MarkdownCommon(markdown)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write(html); err != nil {
		return fmt.Errorf("failed to send markdown: %s", err)
	}
	return nil
}

func main() {
	if err := parseFlags(); err != nil {
		log.Fatalf("Failed to parse flags: %s", err)
	}
	http.Handle("/images/", http.FileServer(http.Dir(*dir)))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			if err := printWorkingDirectoryList(w); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, err)
				log.Print(err)
			}
		} else if filepath.Ext(r.URL.Path) == ".md" {
			if err := serveMarkdownFromRequestPath(w, r); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, err)
				log.Print(err)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, "404 not found")
		}
	})

	fmt.Printf("Visit http://localhost:%d\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
