package main

import (
	"flag"
	"github.com/d-Rickyy-b/rexamine"
	"io"
	"os"
	"regexp"
)

func main() {
	regex := flag.String("regex", `[\w\-+\.%]+@[\w-]+\.[a-zA-Z]{2,24}`, "Regex to use for matching")
	fileFlag := flag.String("file", "", "File to read from")

	flag.Parse()

	pattern := regexp.MustCompile(*regex)

	targetFile, openErr := os.Open(*fileFlag)
	if openErr != nil {
		os.Exit(1)
	}

	w := rexamine.NewRegexWriter(pattern)
	go func() {
		_, err := io.Copy(w, targetFile)
		if err != nil {
			os.Exit(1)
		}
		w.Close()
	}()

	_, err := w.FindAllMatches()
	if err != nil {
		return
	}
}
