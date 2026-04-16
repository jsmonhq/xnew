// xnew: append only lines that are not already in a file.
// Optimized for very large files with low memory usage:
// - stream existing file (no full file load)
// - store only 64-bit hashes in an open-addressed set
// - stream stdin and append new lines in one pass
//
// Usage:
//   xnew existing.txt                 # read stdin, append new lines to existing.txt
//   xnew existing.txt -o out.txt      # write new uniques to out.txt
//   cat new.txt | xnew existing.txt
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"

	"github.com/zeebo/xxh3"
)

const (
	defaultReadBuf  = 8 << 20 // 8 MiB scanner line limit
	defaultWriteBuf = 4 << 20 // 4 MiB output buffer
)

func main() {
	outputPath := flag.String("o", "", "output file (default: append to existing file)")
	trimSpace := flag.Bool("trim", false, "trim spaces when comparing")
	quiet := flag.Bool("q", false, "quiet: no output (only exit code)")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: xnew <existing-file> [-o out] [-trim] [-q]\n")
		os.Exit(1)
	}
	existingPath := args[0]
	if *outputPath == "" {
		*outputPath = existingPath
	}

	_, err := run(existingPath, *outputPath, *trimSpace, *quiet)
	if err != nil {
		fmt.Fprintf(os.Stderr, "xnew: %v\n", err)
		os.Exit(1)
	}
}

func run(existingPath, outputPath string, trim, quiet bool) (int64, error) {
	showInserted := !quiet
	set := newHashSet(1024)

	if err := loadExistingHashes(existingPath, trim, set); err != nil {
		return 0, err
	}

	out, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return 0, err
	}
	defer out.Close()

	w := bufio.NewWriterSize(out, defaultWriteBuf)
	defer func() { _ = w.Flush() }()

	sc := bufio.NewScanner(os.Stdin)
	sc.Buffer(make([]byte, 0, 64<<10), defaultReadBuf)
	var written int64

	for sc.Scan() {
		line := sc.Bytes()
		if trim {
			line = bytes.TrimSpace(line)
		}
		if len(line) == 0 {
			continue
		}
		if set.AddHash(hashLine(line)) {
			if showInserted {
				// stdout so `xnew f >> other` captures inserted lines via redirection
				_, _ = os.Stdout.Write(line)
				_, _ = os.Stdout.Write([]byte{'\n'})
			}
			_, _ = w.Write(line)
			_ = w.WriteByte('\n')
			written++
		}
	}

	if err := sc.Err(); err != nil {
		return written, err
	}
	if err := w.Flush(); err != nil {
		return written, err
	}
	return written, nil
}

func loadExistingHashes(path string, trim bool, set *hashSet) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64<<10), defaultReadBuf)
	for sc.Scan() {
		line := sc.Bytes()
		if trim {
			line = bytes.TrimSpace(line)
		}
		if len(line) == 0 {
			continue
		}
		set.AddHash(hashLine(line))
	}
	if err := sc.Err(); err != nil {
		return err
	}
	return nil
}

func hashLine(line []byte) uint64 {
	h := xxh3.Hash(line)
	if h == 0 {
		return 1 // keep zero as empty-slot sentinel
	}
	return h
}

type hashSet struct {
	keys []uint64 // 0 == empty slot
	used int
}

func newHashSet(minCap int) *hashSet {
	if minCap < 16 {
		minCap = 16
	}
	size := nextPow2(minCap)
	return &hashSet{keys: make([]uint64, size)}
}

func (s *hashSet) AddHash(h uint64) bool {
	// keep max load ~70% for speed and short probe chains
	if (s.used+1)*10 >= len(s.keys)*7 {
		s.grow()
	}

	mask := uint64(len(s.keys) - 1)
	i := h & mask
	for {
		k := s.keys[i]
		if k == 0 {
			s.keys[i] = h
			s.used++
			return true
		}
		if k == h {
			return false
		}
		i = (i + 1) & mask
	}
}

func (s *hashSet) grow() {
	old := s.keys
	s.keys = make([]uint64, len(old)*2)
	s.used = 0
	mask := uint64(len(s.keys) - 1)

	for _, h := range old {
		if h == 0 {
			continue
		}
		i := h & mask
		for s.keys[i] != 0 {
			i = (i + 1) & mask
		}
		s.keys[i] = h
		s.used++
	}
}

func nextPow2(n int) int {
	x := 1
	for x < n {
		x <<= 1
	}
	return x
}