package streamregex

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
	targetFile, openErr := os.Open(fileName)
	if openErr != nil {
		os.Exit(1)
	}

	buf := &bytes.Buffer{}
	_, copyErr := io.Copy(buf, targetFile)
	if copyErr != nil {
		return
	}

	_ = pattern.FindAll(buf.Bytes(), -1)
}

func BenchmarkIOReadAll(b *testing.B) {
	targetFile, openErr := os.Open(fileName)
	if openErr != nil {
		os.Exit(1)
	}

	content, readErr := io.ReadAll(targetFile)
	if readErr != nil {
		os.Exit(1)
	}

	_ = pattern.FindAll(content, -1)
}

func BenchmarkRexamine(b *testing.B) {
	targetFile, openErr := os.Open(fileName)
	if openErr != nil {
		os.Exit(1)
	}

	newReader := NewRegexReaderSize(targetFile, pattern, 16)

	_, err := newReader.FindAllMatches()
	if err != nil {
		return
	}
}

func BenchmarkRexamineWriter(b *testing.B) {
	for i := 0; i < b.N; i++ {
		targetFile, openErr := os.Open(fileName)
		if openErr != nil {
			os.Exit(1)
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
			return
		}
	}
}
