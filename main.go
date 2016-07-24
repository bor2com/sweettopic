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

var dir = flag.String("dir", "", "Working directory.")

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

func printWorkingDirectoryList(w http.ResponseWriter) error {
	entries, err := ioutil.ReadDir(*dir)
	if err != nil {
		return fmt.Errorf("failed to list files in working directory %q, make sure it has not been deleted: %s", *dir, err)
	}
	var markdowns []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".md" {
			markdowns = append(markdowns, entry.Name())
		}
	}
	var buffer bytes.Buffer
	if len(markdowns) == 0 {
		buffer.WriteString("Nothing found in working directory. Make sure it contains subdirectories.")
	} else {
		buffer.WriteString("Catalog:\n")
		for _, markdown := range markdowns {
			fmt.Fprintf(&buffer, "* %s\n", markdown)
		}
	}
	return serveMarkdown(w, buffer.Bytes())
}

func serveMarkdownFromRequestPath(w http.ResponseWriter, r *http.Request) error {
	localPath := filepath.Join(*dir, r.URL.Path)
	rawFile, err := ioutil.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read local file, make sure %q exists", localPath)
	}
	return serveMarkdown(w, rawFile)
}

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
		} else {
			switch filepath.Ext(r.URL.Path) {
			case ".md":
				if err := serveMarkdownFromRequestPath(w, r); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Fprint(w, err)
					log.Print(err)
				}
			case ".png", ".jpg", ".jpeg":
				fallthrough
			default:
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprint(w, "404 not found")
			}
		}
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
