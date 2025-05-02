package rexamine

import (
	"bytes"
	"io"
	"os"
	"regexp"
	"testing"
)

var (
	regex    = `[\w\-+\.%]+@[\w-]+\.[a-zA-Z]{2,24}`
	fileName = "500mb.txt"
	pattern  = regexp.MustCompile(regex)
)

func BenchmarkIOCopy(b *testing.B) {
	for i := 0; i < b.N; i++ {
		targetFile, openErr := os.Open(fileName)
		if openErr != nil {
			b.Fatalf("Error opening file: %v", openErr)
		}

		buf := &bytes.Buffer{}
		_, copyErr := io.Copy(buf, targetFile)
		if copyErr != nil {
			b.Fatalf("Error reading file: %v", copyErr)
		}

		_ = pattern.FindAll(buf.Bytes(), -1)
	}
}

func BenchmarkIOReadAll(b *testing.B) {
	for i := 0; i < b.N; i++ {
		targetFile, openErr := os.Open(fileName)
		if openErr != nil {
			b.Fatalf("Error opening file: %v", openErr)
		}

		content, readErr := io.ReadAll(targetFile)
		if readErr != nil {
			b.Fatalf("Error reading file: %v", readErr)
		}

		_ = pattern.FindAll(content, -1)
	}
}

func BenchmarkRexamine(b *testing.B) {
	for i := 0; i < b.N; i++ {
		targetFile, openErr := os.Open(fileName)
		if openErr != nil {
			b.Fatalf("Error opening file: %v", openErr)
		}

		newReader := NewRegexReader(targetFile, pattern)

		_, err := newReader.FindAllMatches()
		if err != nil {
			b.Fatalf("Error searching for matches: %v", err)
		}
	}
}

func BenchmarkRexamineWriter(b *testing.B) {
	for i := 0; i < b.N; i++ {
		targetFile, openErr := os.Open(fileName)
		if openErr != nil {
			b.Fatalf("Error opening file: %v", openErr)
		}

		w := NewRegexWriter(pattern)
		go func() {
			_, err := io.Copy(w, targetFile)
			if err != nil {
				os.Exit(1)
			}
			w.Close()
		}()

		_, err := w.FindAllMatches()
		if err != nil {
			b.Fatalf("Error searching for matches file: %v", err)
		}
	}
}
