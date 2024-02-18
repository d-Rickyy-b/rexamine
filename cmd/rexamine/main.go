package main

import (
	"flag"
	"os"
	"regexp"

	"github.com/d-Rickyy-b/rexamine/pkg/streamregex"
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

	// fmt.Println("Using custom")
	newReader := streamregex.NewRegexReader(targetFile, pattern)

	_, err := newReader.FindAllMatches()
	if err != nil {
		return
	}

	// fmt.Println(strings.Join(matches, "\n"))
}
