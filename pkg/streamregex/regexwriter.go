package streamregex

import (
	"io"
	"regexp"
)

// RegexWriter
type RegexWriter struct {
	rd RegexReader
	wr io.WriteCloser
}

// NewRegexWriter returns a new Writer whose buffer has the default size.
func NewRegexWriter(pattern *regexp.Regexp) *RegexWriter {
	return NewRegexWriterSize(pattern, defaultBufferSize)
}

// NewRegexWriterSize returns a new Reader whose buffer has at least the specified size.
func NewRegexWriterSize(pattern *regexp.Regexp, size int) *RegexWriter {
	if size < minReadBufferSize {
		size = minReadBufferSize
	}

	r, w := io.Pipe()
	regexReader := NewRegexReaderSize(r, pattern, size)

	return &RegexWriter{rd: *regexReader, wr: w}
}

// Write writes data to the pipe, blocking until one or more readers
// have consumed all the data or the read end is closed.
func (w *RegexWriter) Write(data []byte) (n int, err error) {
	return w.wr.Write(data)
}

// FindAllMatches finds all matches found in the reader and returns them as a slice of strings.
// Calling this method will block until the reader has been fully read.
func (w *RegexWriter) FindAllMatches() ([]string, error) {
	return w.rd.FindAllMatches()
}

// FindAllMatchesFunc finds the next match and calls the deliver function with the match.
// This allows for accessing the match while the reader is still reading.
func (w *RegexWriter) FindAllMatchesFunc(deliver func(string)) error {
	return w.rd.FindAllMatchesFunc(deliver)
}

// Close closes the write end of the pipe.
func (w *RegexWriter) Close() {
	w.wr.Close()
}
