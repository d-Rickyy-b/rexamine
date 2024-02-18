package streamregex

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"unicode/utf8"
)

const (
	minReadBufferSize = 16
	defaultBufferSize = 4096
)

var errNegativeRead = errors.New("reader returned negative count from Read")

// RegexReader implements a buffered reader that supports regex matching
// It works great for regular regexes, but it causes problems for regexes that exceed 2x the buffer's size
// This can easily happen by using regexes including non-greedy quantifiers like .*
type RegexReader struct {
	rd              io.Reader
	buf             []byte
	prevBuf         []byte
	pattern         *regexp.Regexp
	sourceReadBytes int // Total number of bytes read from the underlying reader rd
	readBytes       int // Total number of bytes read from this reader
	prevReadBytes   int // Total number of bytes read from this reader after the previous regex match
	err             error
	r, w            int // buf read and write positions
}

// NewRegexReader returns a new Reader whose buffer has the default size.
func NewRegexReader(r io.Reader, pattern *regexp.Regexp) *RegexReader {
	return NewRegexReaderSize(r, pattern, defaultBufferSize)
}

// NewRegexReaderSize returns a new Reader whose buffer has at least the specified size.
func NewRegexReaderSize(r io.Reader, pattern *regexp.Regexp, size int) *RegexReader {
	if size < minReadBufferSize {
		size = minReadBufferSize
	}

	return &RegexReader{
		rd:      r,
		buf:     make([]byte, size),
		prevBuf: make([]byte, size),
		pattern: pattern,
	}
}

// FindAllMatches finds all matches found in the reader and returns them as a slice of strings.
// Calling this method will block until the reader has been fully read.
func (rr *RegexReader) FindAllMatches() ([]string, error) {
	var results []string

	err := rr.FindAllMatchesFunc(func(match string) {
		results = append(results, match)
	})
	if err != nil {
		fmt.Println("Error:", err)
	}

	return results, err
}

// FindAllMatchesFunc finds the next match and calls the deliver function with the match.
// This allows for accessing the match while the reader is still reading.
func (rr *RegexReader) FindAllMatchesFunc(deliver func(string)) error {
	for {
		loc := rr.pattern.FindReaderIndex(rr)
		if loc == nil {
			break
		}

		length := loc[1] - loc[0]
		offset := loc[0]

		lastBytes, err := rr.getLastBytes(offset, length)
		if err != nil {
			return err
		}
		deliver(string(lastBytes))
	}

	return nil
}

// ReadRune implements the io.RuneReader interface.
// It reads and returns the next UTF-8-encoded Unicode code point from the buffer.
// It automatically fills the buffer as necessary from the underlying reader.
func (rr *RegexReader) ReadRune() (r rune, size int, err error) {
	for rr.r+utf8.UTFMax > rr.w && !utf8.FullRune(rr.buf[rr.r:rr.w]) && rr.err == nil && rr.w-rr.r < len(rr.buf) {
		rr.fill() // m.w-m.r < len(buf) => buffer is not full
	}

	if rr.r == rr.w {
		return 0, 0, rr.readErr()
	}

	r, size = rune(rr.buf[rr.r]), 1
	if r >= utf8.RuneSelf {
		r, size = utf8.DecodeRune(rr.buf[rr.r:rr.w])
	}

	rr.r += size
	rr.readBytes += size

	return r, size, nil
}

// Read implements the io.Reader interface.
// It reads as many bytes as fit in p.
// It automatically fills the buffer as necessary from the underlying reader.
func (rr *RegexReader) Read(p []byte) (n int, err error) {
	if rr.r == rr.w {
		if rr.err != nil {
			return 0, rr.readErr()
		}
		rr.fill()
		if rr.r == rr.w {
			return 0, io.EOF
		}
	}
	n = copy(p, rr.buf[rr.r:rr.w])

	rr.r += n

	return n, nil
}

// getLastBytes returns the last n bytes from the buffer.
// It resets the reader to right after the match (n+l).
func (rr *RegexReader) getLastBytes(n, l int) ([]byte, error) {
	n += rr.prevReadBytes
	defer func() {
		rr.prevReadBytes = rr.readBytes
	}()

	// The regex reader reads more bytes than the regex match, so we need to reset the reader to the correct position
	err := rr.resetReaderTo(n + l)
	if err != nil {
		return nil, err
	}

	result := make([]byte, l)

	// If the underlying reader supports io.ReaderAt, use that to get the last bytes
	rAt, ok := rr.rd.(io.ReaderAt)
	if ok {
		_, err := rAt.ReadAt(result, int64(n))
		if err != nil {
			return result, err
		}
		return result, nil
	}

	bl := rr.bufLower()
	baseOffset := n - bl

	// Check if we need to copy bytes from prevBuf, buf or both
	if baseOffset >= len(rr.buf) {
		// Match is fully in buf
		baseOffset -= len(rr.buf)
		copy(result, rr.buf[baseOffset:baseOffset+l])
	} else if baseOffset < len(rr.buf) {
		if baseOffset+l > len(rr.buf) {
			// Match is split between buf and prevBuf
			copy(result, rr.prevBuf[baseOffset:])
			copy(result[len(rr.buf)-baseOffset:], rr.buf[:l-(len(rr.buf)-baseOffset)])
		} else {
			// Match is fully is in prevBuf
			copy(result, rr.prevBuf[baseOffset:baseOffset+l])
		}
	} else {
		fmt.Println("Out of bounds match:", rr.readBytes-(n+l))
		panic("Out of bounds match")
	}

	return result, nil
}

// bufLower returns the lowest address from the beginning of the source reader m.rd that is still in the buffer
// Since the RegexReader internally has two buffers of the same size, the max we can buffer is 2x the buffer size
//
// For example:
//   - If the buffer is 100 bytes and the reader has read 150 bytes, the lowest address is 0
//   - If the buffer is 100 bytes and the reader has read 250 bytes, the lowest address is 50
func (rr *RegexReader) bufLower() (n int) {
	bufferedBytes := len(rr.buf) + rr.w
	return max(rr.sourceReadBytes-bufferedBytes, 0)
}

// bufUpper returns the highest address from the beginning of the source reader m.rd that is in the buffer
func (rr *RegexReader) bufUpper() (n int) {
	return rr.sourceReadBytes
}

// fill reads data from the source reader into the buffer
func (rr *RegexReader) fill() {
	if rr.r > 0 {
		// Copy previous buffer end to the beginning of the buffer
		if rr.r < len(rr.buf) {
			// We only need to copy within prevBuf if the read bytes size is smaller than the buffer size
			copy(rr.prevBuf, rr.prevBuf[rr.r:])
		}
		// Copy current buffer to the end of the previous buffer
		copy(rr.prevBuf[len(rr.buf)-rr.r:], rr.buf[:rr.r])
		// Copy unread part of current buffer to the beginning of the buffer
		copy(rr.buf, rr.buf[rr.r:rr.w])

		rr.w -= rr.r
		rr.r = 0
	}

	if rr.w >= len(rr.buf) {
		panic("tried to fill full buffer")
	}

	maxConsecutiveEmptyReads := 100
	// Read new data: try a limited number of times.
	for i := maxConsecutiveEmptyReads; i > 0; i-- {
		n, err := rr.rd.Read(rr.buf[rr.w:])
		if n < 0 {
			panic(errNegativeRead)
		}
		rr.w += n
		rr.sourceReadBytes += n
		if err != nil {
			rr.err = err
			return
		}
		if n > 0 {
			return
		}
	}
	rr.err = io.ErrNoProgress
}

func (rr *RegexReader) readErr() error {
	err := rr.err
	rr.err = nil
	return err
}

// resetReaderTo resets the reader to a specific position
// We need to reset this reader sometimes, because the regex reader consumes more bytes than the regex match
func (rr *RegexReader) resetReaderTo(newR int) error {
	diff := rr.readBytes - newR
	rr.readBytes -= diff
	rr.r -= diff

	if rr.r < 0 {
		return fmt.Errorf("resetReaderTo: negative read position: %d", rr.r)
	}

	return nil
}
