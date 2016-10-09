package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

const port = 8080

var serve = flag.String("serve", "", "Working directory.")

func main() {
	flag.Parse()
	file, err := os.Open(*serve)
	if err != nil {
		log.Fatalf("Failed to open directory %q: %s", *serve, err)
	}
	fileInfo, err := file.Stat()
	if err != nil {
		log.Fatalf("Unable to get file stats for %q: %s", *serve, err)
	}
	if !fileInfo.IsDir() {
		log.Fatalf("Provided path %q is not a directory", *serve)
	}

	absolute, err := filepath.Abs(*serve)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Serving content from %q. Visit http://localhost:%d\n", absolute, port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), http.FileServer(http.Dir(*serve))))
}
