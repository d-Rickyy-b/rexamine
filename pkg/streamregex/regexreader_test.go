package streamregex

import (
	"bytes"
	"io"
	"regexp"
	"testing"
)

// TestEmptyInput tests the case where the input data is empty.
func TestEmptyInput(t *testing.T) {
	pattern := regexp.MustCompile(`[\w\-+\.%]+@[\w-]+\.[a-zA-Z]{2,3}`)
	testData := []byte("")
	reader := bytes.NewReader(testData)
	regexReader := NewRegexReaderSize(reader, pattern, 16)

	m, err := regexReader.FindAllMatches()
	if err != nil {
		t.Fatalf("Error searching for matches: %v", err)
	}

	if len(m) != 0 {
		t.Fatalf("Expected 0 matches, got %d", len(m))
	}
}

// TestNoMatches tests the case where the input data does not contain any matches for the regex pattern.
func TestNoMatches(t *testing.T) {
	pattern := regexp.MustCompile(`[\w\-+\.%]+@[\w-]+\.[a-zA-Z]{2,3}`)
	testData := []byte("this is some text without any email addresses in it")
	reader := bytes.NewReader(testData)
	regexReader := NewRegexReaderSize(reader, pattern, 16)

	m, err := regexReader.FindAllMatches()
	if err != nil {
		t.Fatalf("Error searching for matches: %v", err)
	}

	if len(m) != 0 {
		t.Fatalf("Expected 0 matches, got %d", len(m))
	}
}

// TestMultipleMatchesInDifferentBuffers tests the case where the regex pattern matches multiple times across different buffers.
func TestMultipleMatchesInDifferentBuffers(t *testing.T) {
	pattern := regexp.MustCompile(`[\w\-+\.%]+@[\w-]+\.[a-zA-Z]{2,3}`)
	testData := []byte("test1@a.com" + "********" + // first buffer (16 chars)
		"********" + "test2@a.com" + // second buffer
		"********" + "test3@a.com") // third buffer
	reader := bytes.NewReader(testData)
	regexReader := NewRegexReaderSize(reader, pattern, 16)

	m, err := regexReader.FindAllMatches()
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

// TestMatchPrevBuffer tests the case where the regex pattern match fully resides in the prevBuf.
func TestMatchPrevBuffer(t *testing.T) {
	pattern := regexp.MustCompile(`[\w\-+\.%]+@[\w-]+\.[a-zA-Z]{2,3}`)
	testData := []byte("****test@a.com**bbbbccccddddeeeeffff")
	reader := bytes.NewReader(testData)
	regexReader := NewRegexReaderSize(reader, pattern, 16)

	m, err := regexReader.FindAllMatches()
	if err != nil {
		t.Fatalf("Error searching for matches: %v", err)
	}

	if len(m) != 1 {
		t.Fatalf("Expected 1 match, got %d", len(m))
	}

	if m[0] != "test@a.com" {
		t.Fatalf("Expected match to be 'test@a.com', got %s", m[0])
	}
}

// TestMatchInBeginning tests the case where the regex pattern match starts right at the beginning of the buffer.
func TestMatchInBeginning(t *testing.T) {
	pattern := regexp.MustCompile(`[\w\-+\.%]+@[\w-]+\.[a-zA-Z]{2,3}`)
	testData := []byte("test@a.com*****bbbbccccddddeeeeffff")
	reader := bytes.NewReader(testData)
	regexReader := NewRegexReaderSize(reader, pattern, 16)

	m, err := regexReader.FindAllMatches()
	if err != nil {
		t.Fatalf("Error searching for matches: %v", err)
	}

	if len(m) != 1 {
		t.Fatalf("Expected 1 match, got %d", len(m))
	}

	if m[0] != "test@a.com" {
		t.Fatalf("Expected match to be 'test@a.com', got %s", m[0])
	}
}

// TestMatchInSecondBuffer tests the case where the regex pattern match starts in the current buffer.
func TestMatchInSecondBuffer(t *testing.T) {
	pattern := regexp.MustCompile(`[\w\-+\.%]+@[\w-]+\.[a-zA-Z]{2,3}`)
	testData := []byte("****************test@a.com*bbbbccccddddeeeeffff")
	reader := bytes.NewReader(testData)
	regexReader := NewRegexReaderSize(reader, pattern, 16)

	m, err := regexReader.FindAllMatches()
	if err != nil {
		t.Fatalf("Error searching for matches: %v", err)
	}

	if len(m) != 1 {
		t.Fatalf("Expected 1 match, got %d", len(m))
	}

	if m[0] != "test@a.com" {
		t.Fatalf("Expected match to be 'test@a.com', got %s", m[0])
	}
}

// // TestTwoConsequtiveMatches tests the case where the regex pattern matches two times right next to each other in the same buffer
func TestTwoConsequtiveMatches(t *testing.T) {
	pattern := regexp.MustCompile(`[\w\-+\.%]+@[\w-]+\.[a-zA-Z]{2,3}`)
	testData := []byte("****test@a.comtest2@a.com****")
	reader := bytes.NewReader(testData)
	regexReader := NewRegexReaderSize(reader, pattern, 16)

	m, err := regexReader.FindAllMatches()
	if err != nil {
		t.Fatalf("Error searching for matches: %v", err)
	}

	if len(m) != 2 {
		t.Fatalf("Expected 2 matches, got %d", len(m))
	}

	if m[0] != "test@a.com" {
		t.Fatalf("Expected first match to be 'test@a.com', got %s", m[0])
	}

	if m[1] != "test2@a.com" {
		t.Fatalf("Expected second match to be 'test2@a.com', got %s", m[1])
	}
}

// TestMatchAcrossBuffers tests multiple cases where the regex pattern match starts in the previous buffer and ends in the current buffer.
func TestMatchAcrossBuffers(t *testing.T) {
	pattern := regexp.MustCompile(`[\w\-+\.%]+@[\w-]+\.[a-zA-Z]{2,3}`)
	tests := []struct {
		name     string
		testData []byte
		expected string
	}{
		{"OneCharInPrevBuf", []byte("***************test@a.com"), "test@a.com"},
		{"MultipleCharInPrevBuf", []byte("*************test@a.com****"), "test@a.com"},
		{"FullPrevBuf", []byte("thisisatest@a.de****bbbbccccddddeeeeffff****"), "thisisatest@a.de"},
		{"TestMatchFirstCharInPrevBuf", []byte("***************test@a.com**bbbbccccddddeeeeffff****"), "test@a.com"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reader := bytes.NewReader(test.testData)
			regexReader := NewRegexReaderSize(reader, pattern, 16)

			m, err := regexReader.FindAllMatches()
			if err != nil {
				t.Fatalf("Error searching for matches: %v", err)
			}

			if len(m) != 1 {
				t.Fatalf("Expected 1 match, got %d", len(m))
			}

			if m[0] != test.expected {
				t.Fatalf("Expected match to be '%s', got '%s'", test.expected, m[0])
			}
		})
	}
}

// ReaderOnly is a struct that wraps a bytes.Reader to only implement the io.Reader interface.
// This is used to test the regex reader with a reader that doesn't support seeking. Otherwise certain code wouldn't be reachable.
type ReaderOnly struct {
	r *bytes.Reader
}

func (ro *ReaderOnly) Read(p []byte) (n int, err error) {
	return ro.r.Read(p)
}

// TestOutOfBoundsRead tests the case where the buffer size is too small to read the entire data.
// Seeking in the buffer must be disabled for this test, otherwise there is no out of bounds error.
func TestOutOfBoundsRead(t *testing.T) {
	pattern := regexp.MustCompile(`[\w\-+\.%]+@[\w-]+\.[a-zA-Z]{2,3}`)
	reader := bytes.NewReader([]byte("abcdefghijklmnopqrstuvwxyzyxwvutsrpqonmlkjihgfedcbatest@a.com"))
	readerOnly := &ReaderOnly{reader}

	// A buffer size of 16 is too small to read the entire data
	regexReader := NewRegexReaderSize(readerOnly, pattern, 16)

	_, err := regexReader.FindAllMatches()
	if err == nil {
		t.Fatalf("Expected OOB error, found nil")
	}
}

// TestMatchInBeginningNoSeek tests the case where the regex pattern match starts right at the beginning of the buffer
// if the result can't be fetched via seeking in the buffer.
func TestMatchInBeginningNoSeek(t *testing.T) {
	pattern := regexp.MustCompile(`[\w\-+\.%]+@[\w-]+\.[a-zA-Z]{2,3}`)
	testData := []byte("test@a.com*****bbbbccccddddeeeeffff")
	reader := bytes.NewReader(testData)
	readerOnly := &ReaderOnly{reader}

	regexReader := NewRegexReaderSize(readerOnly, pattern, 16)

	m, err := regexReader.FindAllMatches()
	if err != nil {
		t.Fatalf("Error searching for matches: %v", err)
	}

	if len(m) != 1 {
		t.Fatalf("Expected 1 match, got %d", len(m))
	}

	if m[0] != "test@a.com" {
		t.Fatalf("Expected match to be 'test@a.com', got %s", m[0])
	}
}

// TestMatchPrevBufferNoSeek tests the case where the regex pattern match fully resides in the prevBuf
// if the result can't be fetched via seeking in the buffer.
func TestMatchPrevBufferNoSeek(t *testing.T) {
	pattern := regexp.MustCompile(`[\w\-+\.%]+@[\w-]+\.[a-zA-Z]{2,3}`)
	testData := []byte("****test@a.com**bbbbccccddddeeeeffff")
	reader := bytes.NewReader(testData)
	readerOnly := &ReaderOnly{reader}

	regexReader := NewRegexReaderSize(readerOnly, pattern, 16)

	m, err := regexReader.FindAllMatches()
	if err != nil {
		t.Fatalf("Error searching for matches: %v", err)
	}

	if len(m) != 1 {
		t.Fatalf("Expected 1 match, got %d", len(m))
	}

	if m[0] != "test@a.com" {
		t.Fatalf("Expected match to be 'test@a.com', got %s", m[0])
	}
}

// TestMultipleMatchesInDifferentBuffersNoSeek tests the case where the regex pattern matches multiple times across different buffers
// if the result can't be fetched via seeking in the buffer.
func TestMultipleMatchesInDifferentBuffersNoSeek(t *testing.T) {
	pattern := regexp.MustCompile(`[\w\-+\.%]+@[\w-]+\.[a-zA-Z]{2,3}`)
	testData := []byte("test1@a.com" + "********" + // first buffer (16 chars)
		"********" + "test2@a.com" + // second buffer
		"********" + "test3@a.com") // third buffer
	reader := bytes.NewReader(testData)
	readerOnly := &ReaderOnly{reader}

	regexReader := NewRegexReaderSize(readerOnly, pattern, 16)

	m, err := regexReader.FindAllMatches()
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

// TestTwoConsequtiveMatchesNoSeek tests the case where the regex pattern matches two times right next to each other in the same buffer
// if the result can't be fetched via seeking in the buffer.
func TestTwoConsequtiveMatchesNoSeek(t *testing.T) {
	pattern := regexp.MustCompile(`[\w\-+\.%]+@[\w-]+\.[a-zA-Z]{2,3}`)
	testData := []byte("****test@a.comtest2@a.com****")
	reader := bytes.NewReader(testData)
	readerOnly := &ReaderOnly{reader}

	regexReader := NewRegexReaderSize(readerOnly, pattern, 16)

	m, err := regexReader.FindAllMatches()
	if err != nil {
		t.Fatalf("Error searching for matches: %v", err)
	}

	if len(m) != 2 {
		t.Fatalf("Expected 2 matches, got %d", len(m))
	}

	if m[0] != "test@a.com" {
		t.Fatalf("Expected first match to be 'test@a.com', got %s", m[0])
	}

	if m[1] != "test2@a.com" {
		t.Fatalf("Expected second match to be 'test2@a.com', got %s", m[1])
	}
}

// TestMatchAcrossBuffersNoSeek tests multiple cases where the regex pattern match starts in the previous buffer and ends in the current buffer.
// if the result can't be fetched via seeking in the buffer.
func TestMatchAcrossBuffersNoSeek(t *testing.T) {
	pattern := regexp.MustCompile(`[\w\-+\.%]+@[\w-]+\.[a-zA-Z]{2,3}`)
	tests := []struct {
		name     string
		testData []byte
		expected string
	}{
		{"OneCharInPrevBuf", []byte("***************test@a.com"), "test@a.com"},
		{"MultipleCharInPrevBuf", []byte("*************test@a.com****"), "test@a.com"},
		{"FullPrevBuf", []byte("thisisatest@a.de****bbbbccccddddeeeeffff****"), "thisisatest@a.de"},
		{"TestMatchFirstCharInPrevBuf", []byte("***************test@a.com**bbbbccccddddeeeeffff****"), "test@a.com"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reader := bytes.NewReader(test.testData)
			readerOnly := &ReaderOnly{reader}
			regexReader := NewRegexReaderSize(readerOnly, pattern, 16)

			m, err := regexReader.FindAllMatches()
			if err != nil {
				t.Fatalf("Error searching for matches: %v", err)
			}

			if len(m) != 1 {
				t.Fatalf("Expected 1 match, got %d", len(m))
			}

			if m[0] != test.expected {
				t.Fatalf("Expected match to be '%s', got '%s'", test.expected, m[0])
			}
		})
	}
}

// TestMinAndDefaultBufferSizes tests if the buffer size is set to the minimum/default values correctly.
func TestMinAndDefaultBufferSizes(t *testing.T) {
	pattern := regexp.MustCompile(`[\w\-+\.%]+@[\w-]+\.[a-zA-Z]{2,3}`)
	testData := []byte("***************")
	reader := bytes.NewReader(testData)
	regexReader := NewRegexReaderSize(reader, pattern, 1)

	if len(regexReader.buf) != minReadBufferSize {
		t.Fatalf("Expected buffer size to be %d, got %d", minReadBufferSize, len(regexReader.buf))
	}

	regexReader2 := NewRegexReader(reader, pattern)

	if len(regexReader2.buf) != defaultBufferSize {
		t.Fatalf("Expected buffer size to be %d, got %d", defaultBufferSize, len(regexReader2.buf))
	}
}

func TestReader(t *testing.T) {
	pattern := regexp.MustCompile(`[\w\-+\.%]+@[\w-]+\.[a-zA-Z]{2,3}`)
	testData := []byte("test1@a.com" + "********" + // first buffer (16 chars)
		"********" + "test2@a.de" + // second buffer
		"********" + "test3@a.com") // third buffer
	reader := bytes.NewReader(testData)
	readerOnly := &ReaderOnly{reader}
	regexReader := NewRegexReaderSize(readerOnly, pattern, 16)

	m, err := regexReader.FindAllMatches()
	if err != nil {
		t.Fatalf("Error searching for matches: %v", err)
	}

	if len(m) != 3 {
		t.Fatalf("Expected 3 matches, got %d", len(m))
	}

	expected := []string{"test1@a.com", "test2@a.de", "test3@a.com"}
	for i, exp := range expected {
		if m[i] != exp {
			t.Fatalf("Expected match %d to be '%s', got '%s'", i, exp, m[i])
		}
	}
}

// TestReaderInterface tests the io.Reader interface of the regex reader.
func TestReaderInterface(t *testing.T) {
	pattern := regexp.MustCompile(`[\w\-+\.%]+@[\w-]+\.[a-zA-Z]{2,3}`)
	testData := []byte("test1@a.com" + "********" + // first buffer (16 chars)
		"********" + "test2@a.de" + // second buffer
		"********" + "test3@a.com") // third buffer
	reader := bytes.NewReader(testData)
	readerOnly := &ReaderOnly{reader}
	regexReader := NewRegexReaderSize(readerOnly, pattern, 16)

	byteBuf := make([]byte, 0, 16)
	tmpBuf := bytes.NewBuffer(byteBuf)
	i, err := io.CopyN(tmpBuf, regexReader, 16)
	if err != nil {
		return
	}

	if tmpBuf.String() != "test1@a.com*****" {
		t.Fatalf("Expected to read 'test1@a.com*****', got '%s'", tmpBuf)
	}

	if i != 16 {
		t.Fatalf("Expected to read 16 bytes, got %d", i)
	}

	if regexReader.readBytes != 16 {
		t.Fatalf("Expected readBytes to be 16 bytes, got %d", regexReader.readBytes)
	}

	tmpBuf.Reset()
	i, err = io.CopyN(tmpBuf, regexReader, 16)
	if err != nil {
		return
	}

	if tmpBuf.String() != "***********test2" {
		t.Fatalf("Expected to read '***********test2', got '%s'", tmpBuf)
	}

	if i != 16 {
		t.Fatalf("Expected to read 16 bytes, got %d", i)
	}

	if regexReader.readBytes != 32 {
		t.Fatalf("Expected readBytes to be 32 bytes, got %d", regexReader.readBytes)
	}
}
