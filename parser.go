package main

import (
	"bytes"
	"encoding/base64"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/russross/blackfriday"
	"golang.org/x/crypto/sha3"
)

var (
	from = flag.String("from", "", "Source directory with markdown and images.")
	to   = flag.String("to", "", "Destination for static content.")
)

func processImages(images []string) (map[string]string, error) {
	mapping := make(map[string]string)
	for _, image := range images {
		file, err := os.Open(image)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		// Compute hash and destination filename.
		shaker := sha3.New224()
		if _, err := io.Copy(shaker, file); err != nil {
			return nil, err
		}
		digest := make([]byte, 0, 28)
		digest = shaker.Sum(digest)
		hash := base64.RawURLEncoding.EncodeToString(digest)
		destPath := filepath.Join(*to, hash+".jpg")

		// Figure out mapping.
		mKey, err := filepath.Rel(*from, image)
		if err != nil {
			return nil, err
		}
		mValue, err := filepath.Rel(*to, destPath)
		if err != nil {
			return nil, err
		}
		mapping[mKey] = mValue
		log.Printf("Map image %q to %q in markdown.", mKey, mValue)

		// Copy image.
		if _, err := file.Seek(0, os.SEEK_SET); err != nil {
			return nil, err
		}
		copy, err := os.Create(destPath)
		if err != nil {
			return nil, err
		}
		if _, err := io.Copy(copy, file); err != nil {
			return nil, err
		}
		if err := copy.Close(); err != nil {
			return nil, err
		}
		log.Printf("Image %q copied to %q.", image, destPath)
	}
	return mapping, nil
}

func processMarkdowns(markdowns []string, imapping map[string]string, output io.WriteCloser) error {
	for _, markdown := range markdowns {
		stream, err := ioutil.ReadFile(markdown)
		if err != nil {
			return err
		}
		for mKey, mValue := range imapping {
			stream = bytes.Replace(stream, []byte(mKey), []byte(mValue), -1)
		}
		html := blackfriday.MarkdownCommon(stream)
		if _, err := output.Write(html); err != nil {
			return err
		}
	}
	return output.Close()
}

func main() {
	flag.Parse()

	if _, err := os.Open(*to); err == nil {
		log.Fatalf("Destination folder %q already exists.", *to)
	}
	if err := os.MkdirAll(*to, 0777); err != nil {
		log.Fatalf("Failed to create destination directory %q: %s", *to, err)
	}

	var markdowns, images []string
	walker := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".md":
			markdowns = append(markdowns, path)
		case ".jpg", ".jpeg":
			images = append(images, path)
		default:
			log.Printf("File or directory %q skipped.", path)
		}
		return nil
	}

	if err := filepath.Walk(*from, walker); err != nil {
		log.Fatalf("Error while processing source files: %s", err)
	}

	imapping, err := processImages(images)
	if err != nil {
		log.Fatal(err)
	}

	outputPath := filepath.Join(*to, "index.html")
	output, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("Failed to create destination HTML %q: %s", outputPath, err)
	}
	if err := processMarkdowns(markdowns, imapping, output); err != nil {
		log.Fatal(err)
	}
}
