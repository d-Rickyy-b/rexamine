package streamregex

import (
	"bytes"
	"io"
	"os"
	"regexp"
	"testing"
)

func TestWriter(t *testing.T) {
	pattern := regexp.MustCompile(`[\w\-+\.%]+@[\w-]+\.[a-zA-Z]{2,3}`)
	testData := []byte("test1@a.com" + "********" + // first buffer (16 chars)
		"********" + "test2@a.com" + // second buffer
		"********" + "test3@a.com") // third buffer
	reader := bytes.NewReader(testData)

	w := NewRegexWriterSize(pattern, 16)
	// Use a go routine to write data to the writer
	go func() {
		_, err := io.Copy(w, reader)
		if err != nil {
			os.Exit(1)
		}
		w.Close()
	}()

	m, err := w.FindAllMatches()
	if err != nil {
		t.Fatalf("Error searching for matches: %v", err)
	}

	if len(m) != 3 {
		t.Fatalf("Expected 3 matches, got %d", len(m))
	}

	expected := []string{"test1@a.com", "test2@a.com", "test3@a.com"}
	for i, exp := range expected {
		if m[i] != exp {
			t.Fatalf("Expected match %d to be '%s', got '%s'", i, exp, m[i])
		}
	}
}
