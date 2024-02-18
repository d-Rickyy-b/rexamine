package main

import (
	"flag"
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

	content, readErr := io.ReadAll(targetFile)
	if readErr != nil {
		os.Exit(1)
	}

	_ = pattern.FindAll(content, -1)
}
