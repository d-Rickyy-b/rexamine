<p align="center">
  <img src="./docs/img/rexamine.png" height="220" />
</p>

# rexamine

[![build](https://github.com/d-Rickyy-b/rexamine/actions/workflows/test.yml/badge.svg)](https://github.com/d-Rickyy-b/rexamine/actions/workflows/test.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/d-Rickyy-b/rexamine.svg)](https://pkg.go.dev/github.com/d-Rickyy-b/rexamine)

Rexamine, pronounced as [ɹɛɡˈzæmən], is a lightweight Go library designed for scanning vast volumes of data using regex. This library was created in order to avoid the excessive memory usage encountered when searching within large files, which traditionally required loading the entire file into memory.

This tool aims to fix that issue by either passing a reader (e.g. from an open file) or by streaming data to the `io.Writer` interface of `RegexWriter`.
That makes it to scan streams of data with regex.

To find out more about the project, check out the [blog post](https://blog.rico-j.de/rexamine-golang-stream-regex).

## Usage

Install the library by using
`go get github.com/d-Rickyy-b/rexamine`.

Then you can use it in your project like this.

```Go
package main

import (
    "fmt"
    "os"
    "regexp"
    "strings"

    "github.com/d-Rickyy-b/rexamine/pkg/streamregex"
)

func main() {
    pattern := regexp.MustCompile(`[\w\-+\.%]+@[\w-]+\.[a-zA-Z]{2,24}`)

    targetFile, openErr := os.Open("sample.txt")
    if openErr != nil {
        os.Exit(1)
    }

    newReader := streamregex.NewRegexReader(targetFile, pattern)

    matches, err := newReader.FindAllMatches()
    if err != nil {
        return
    }

    fmt.Println(strings.Join(matches, "\n"))
}
```

## Pitfalls and Drawbacks

This library was created as a proof of concept.
It hasn't been tested excessively, so there might still be bugs.
Use at your own risk.

To search through large files via regex, rexamine caches data in a buffer.
If the amount of characters matched by a given regex exceeds the chosen buffer size, obviously the full match cannot be extracted.
This can easily happen by using unlimited quantifiers like `*`, `+` or `{3,}`.
To prevent issues, make sure to limit the length of a match by either specifically defining the quantity `{5}` or at least by setting an upper bound.

So instead of matching "any amount of matches" with `*`, use `{,10}` to match "any amount of matches up to 10".
And instead of matching "one or more matches" with `+`, use `{1,10}` to match "one or more matches up to 10".

## Benchmark

The benchmarks show that rexamine is about **5-7%** slower than reading all the file's content into memory and searching it with regex all at once.
On the other hand, it uses only about **19 KB** of memory.

### Preparation

For the benchmark, we need to generate some binary and text files (haystack) in order to compare the different approaches.
To do so, we used the following method:

Binary file:
`dd if=/dev/urandom bs=100M count=1 iflag=fullblock of=sample.bin`

Text file:
`dd if=/dev/urandom iflag=fullblock | base64 -w 0 | head -c 100M > sample.txt`

To insert some data (needle) we want to find with rexamine:

```bash
echo -n "MyData" | dd bs=1 seek=1000 of=sample.txt
echo -n "MyData" | dd bs=1 seek=10000 of=sample.txt
echo -n "MyData" | dd bs=1 seek=100000 of=sample.txt
```

After that we only need to compile the four test binaries.

```bash
go build .\cmd\iocopy
go build .\cmd\iocreadall
go build .\cmd\rexamine
go build .\cmd\rexaminewriter
```

### hyperfine

With the generated files in place we can now run rexamine on these files and compare different approaches.
To do this efficiently, we can utilize [hyperfine](https://github.com/sharkdp/hyperfine).

```bash
rexamine> hyperfine -w 2 -r 6 'iocopy.exe -file 500mb.txt -regex "..."' 'ioreadall.exe -file 500mb.txt -regex "..."' 'rexamine.exe -file 500mb.txt -regex "..."' 'rexaminewriter.exe -file 500mb.txt -regex "..."'
Benchmark 1: iocopy.exe -file 500mb.txt -regex [\w\-+\.%]+@[\w-]+\.[a-zA-Z]{2,24}
  Time (mean ± σ):      8.045 s ±  0.279 s    [User: 2.852 s, System: 0.049 s]
  Range (min … max):    7.887 s …  8.612 s    6 runs

Benchmark 2: ioreadall.exe -file 500mb.txt -regex [\w\-+\.%]+@[\w-]+\.[a-zA-Z]{2,24}
  Time (mean ± σ):      8.085 s ±  0.042 s    [User: 3.263 s, System: 0.042 s]
  Range (min … max):    8.023 s …  8.135 s    6 runs

Benchmark 3: rexamine.exe -file 500mb.txt -regex [\w\-+\.%]+@[\w-]+\.[a-zA-Z]{2,24}
  Time (mean ± σ):      8.630 s ±  0.083 s    [User: 3.729 s, System: 0.104 s]
  Range (min … max):    8.572 s …  8.753 s    6 runs

Benchmark 4: rexaminewriter.exe -file 500mb.txt -regex [\w\-+\.%]+@[\w-]+\.[a-zA-Z]{2,24}
  Time (mean ± σ):      8.601 s ±  0.041 s    [User: 1.391 s, System: 0.062 s]
  Range (min … max):    8.551 s …  8.669 s    6 runs

Summary
  iocopy.exe -file 500mb.txt -regex [\w\-+\.%]+@[\w-]+\.[a-zA-Z]{2,24} ran
    1.00 ± 0.04 times faster than ioreadall.exe -file 500mb.txt -regex [\w\-+\.%]+@[\w-]+\.[a-zA-Z]{2,24}
    1.07 ± 0.04 times faster than rexamine.exe -file 500mb.txt -regex [\w\-+\.%]+@[\w-]+\.[a-zA-Z]{2,24}
    1.07 ± 0.04 times faster than rexaminewriter.exe -file 500mb.txt -regex [\w\-+\.%]+@[\w-]+\.[a-zA-Z]{2,24}
```

We can see that rexamine is about 7% slower than reading the full file into memory and processing it via regex.

### Memory footprint

Since rexamine was specifically developed to decrease the memory footprint, the used memory is much more important than execution speed.
We can use Go's benchmarking tooling to get data on memory usage.

```bash
rexamine> go test -bench=.\pkg\streamregex -benchmem -run=^$ -bench ^Benchmark.+$ -count 5
cpu: AMD Ryzen 9 7900 12-Core Processor
BenchmarkIOCopy-24                     1        1618671100 ns/op        268449152 B/op        76 allocs/op
BenchmarkIOReadAll-24                  1        1570844100 ns/op        615242440 B/op       106 allocs/op
BenchmarkRexamine-24                   1        1751799700 ns/op           19464 B/op         58 allocs/op
BenchmarkRexamineWriter-24             1        1735913500 ns/op           53440 B/op         68 allocs/op
```

`io.ReadAll` needs by far the most memory allocations. For a 100 MB file, it allocates (and frees) more than 600 MB.
`io.Copy` requires less than that, but still around 270 MB.
rexamine completely crushes it with only 19 KB of allocated memory.

```text
cpu: AMD Ryzen 9 7900 12-Core Processor
                  │   sec/op    │
IOCopy-24            1.572 ± 4%
IOReadAll-24         1.594 ± 1%
Rexamine-24          1.761 ± 1%
RexamineWriter-24    1.760 ± 3%
geomean              1.669

                  │     B/op      │
IOCopy-24           256.0Mi ±  0%
IOReadAll-24        586.7Mi ±  0%
Rexamine-24         18.95Ki ±  0%
RexamineWriter-24   51.37Ki ± 11%
geomean             3.436Mi

                  │  allocs/op  │
IOCopy-24           69.00 ± 10%
IOReadAll-24        94.50 ±  4%
Rexamine-24         55.00 ±  4%
RexamineWriter-24   62.00 ± 52%
geomean             68.67
```
