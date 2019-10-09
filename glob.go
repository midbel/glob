package glob

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const branch = "**"

const (
	slash     = '/'
	backslash = '\\'

	star    = '*'
	mark    = '?'
	lsquare = '['
	rsquare = ']'
	caret   = '^'
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
		ps := cleanPattern(strings.FieldsFunc(pattern, splitPattern))
		glob(queue, dir, ps)
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
		file := filepath.Join(dir, i.Name())
		if i.IsDir() {
			glob(queue, file, pattern[1:])
		}
		if len(pattern) <= 1 {
			queue <- file
		}
	}
}

func globAny(queue chan<- string, dir string, pattern []string) {
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

func empty() {
	fmt.Println()
}

func match(str, pat string) bool {
	// shortcut: pat is only one star or pat and str are identicals
	if pat == string(star) || (len(str) == len(pat) && str == pat) {
		return true
	}
	var i, j int
	for ; i < len(pat); i++ {
		switch char := pat[i]; char {
		case star:
			// multiple stars is the same as one star
			for i = i + 1; i < len(pat) && pat[i] == char; i++ {
			}
			// trailing star matchs rest of text - star matchs also empty string
			if i >= len(pat) || str == "" {
				return true
			}
			for j < len(str) {
				if ok := match(str[j:], pat[i:]); ok {
					return ok
				}
				j++
			}
		case mark:
			// match a single character
		case lsquare:
			i++
			reverse := pat[i] == caret
			if reverse {
				i++
			}
			var ok bool
			for pat[i] != rsquare {
				ok = str[j] == pat[i]
				i++
				if ok {
					break
				}
			}
			for pat[i] != rsquare {
				i++
			}
			if reverse {
				ok = !ok
			}
			if !ok {
				return false
			}
		default:
			if j >= len(str) || pat[i] != str[j] {
				return false
			}
		}
		if j >= len(str) {
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
				i := is[i]
				if set := i.Mode() & os.ModeSymlink; set != 0 {
					f, err := filepath.EvalSymlinks(filepath.Join(dir, i.Name()))
					if err != nil {
						continue
					}
					if i, err = os.Stat(f); err != nil {
						continue
					}
				}
				queue <- i
			}
			if len(is) == 0 || err == io.EOF {
				break
			}
		}
	}()
	return queue
}

func cleanPattern(pattern []string) []string {
	// just remove consecutive **
	for i := 0; ; i++ {
		if i >= len(pattern) {
			break
		}
		if j := i - 1; j >= 0 && pattern[i] == branch && pattern[j] == branch {
			pattern, i = append(pattern[:i], pattern[i+1:]...), j
		}
	}
	return pattern
}

func splitPattern(r rune) bool {
	return r == slash || r == backslash
}
