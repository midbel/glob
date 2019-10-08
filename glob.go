package glob

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

const branch = "**"

const (
	slash     = '/'
	backslash = '\\'

	star = '*'
	mark = '?'
)

func Match(dir, pattern string) bool {
	return match(dir, pattern)
}

func Glob(pattern string) []string {
	var (
		files = make([]string, 0, 100)
		glob  = New("", pattern)
	)
	for f := glob.Glob(); f != ""; f = glob.Glob() {
		files = append(files, f)
	}
	return files
}

type Globber struct {
	queue <-chan string

	keepDir  bool // keep directories when they match the given pattern
	keepLink bool // follows symlinks
}

func New(dir, pattern string) *Globber {
	queue := make(chan string)
	go func() {
		defer close(queue)
		if pattern == "" {
			return
		}
		if dir == "" {
			dir = "."
		}
		glob(queue, dir, strings.FieldsFunc(pattern, splitPattern))
	}()
	g := Globber{queue: queue}
	return &g
}

func (g *Globber) Glob() string {
	s, ok := <-g.queue
	if !ok {
		s = ""
	}
	return s
}

func glob(queue chan<- string, dir string, pattern []string) {
	if len(pattern) == 0 {
		return
	}
	if pattern[0] == branch {
		if len(pattern) > 1 {
			globAny(queue, dir, pattern)
		} else {
			for i := range readDir(dir) {
				if file := filepath.Join(dir, i.Name()); i.IsDir() {
					glob(queue, file, pattern)
				} else {
					queue <- file
				}
			}
		}
		return
	}
	for i := range readDir(dir) {
		if ok := match(i.Name(), pattern[0]); !ok {
			continue
		}
		if file := filepath.Join(dir, i.Name()); i.IsDir() {
			glob(queue, file, pattern[1:])
		} else {
			queue <- file
		}
	}
}

func globAny(queue chan<- string, dir string, pattern []string) {
	if pattern[1] == branch {
		globAny(queue, dir, pattern[1:])
		return
	}
	for i := range readDir(dir) {
		var ix int
		switch ok, idir := match(i.Name(), pattern[1]), i.IsDir(); {
		case ok && idir:
			ix++
			ix++
		case ok && !idir:
			queue <- filepath.Join(dir, i.Name())
			continue
		default:
		}
		glob(queue, filepath.Join(dir, i.Name()), pattern[ix:])
	}
}

func match(str, pat string) bool {
	// shortcut: pat is only one star or pat and str are identicals
	if pat == string(star) || (len(str) == len(pat) && str == pat) {
		return true
	}
	var i, j int
	// for ; i < len(pat) && j < len(str); i++ {
	for ; i < len(pat); i++ {
		switch char := pat[i]; {
		case char == star:
			// multiple stars is the same as one star
			for i = i + 1; i < len(pat) && pat[i] == char; i++ {
			}
			// trailing star matchs rest of text
			if i >= len(pat) {
				return true
			}
			for j < len(str) {
				if ok := match(str[j:], pat[i:]); ok {
					return ok
				}
				j++
			}
		case char == mark || (j < len(str) && char == str[j]):
			// default
		default:
			return false
		}
		j++
	}
	// match when all characters of pattern and text have been read
	return i == len(pat) && j == len(str)
}

func readDir(dir string) <-chan os.FileInfo {
	r, err := os.Open(dir)
	if err != nil {
		return nil
	}
	queue := make(chan os.FileInfo)
	go func() {
		defer func() {
			close(queue)
			r.Close()
		}()
		for {
			is, err := r.Readdir(100)
			for i := range is {
				queue <- is[i]
			}
			if len(is) == 0 || err == io.EOF {
				break
			}
		}
	}()
	return queue
}

func splitPattern(r rune) bool {
	return r == slash || r == backslash
}
