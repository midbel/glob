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

	dash    = '-'
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
		globAny(queue, dir, pattern[1:])
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
	if len(pattern) == 1 {
		for i := range readDir(dir) {
			file := filepath.Join(dir, i.Name())
			if i.IsDir() {
				glob(queue, file, pattern)
			}
			queue <- file
		}
		return
	}
	for i := range readDir(dir) {
		ok := match(i.Name(), pattern[0])
		if i.Mode().IsRegular() && ok && len(pattern) == 1 {
			queue <- filepath.Join(dir, i.Name())
			continue
		}
		if i.IsDir() {
			if file := filepath.Join(dir, i.Name()); ok {
				glob(queue, file, pattern[1:])
				if len(pattern) <= 1 {
					queue <- file
				}
			} else {
				globAny(queue, file, pattern)
			}
		}
	}
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
			ni, nj, ok := starMatch(str[j:], pat[i:])
			if ok {
				return ok
			}
			i += ni
			j += nj
		case mark:
			// match a single character
		case lsquare:
			n, ok := charsetMatch(str[j], pat[i+1:])
			if !ok {
				return false
			}
			i += n + 1
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

func starMatch(str, pat string) (int, int, bool) {
	// multiple stars is the same as one star
	var (
		i, j int
		ok   bool
	)
	for i = 1; i < len(pat) && pat[i] == star; i++ {
	}
	// trailing star matchs rest of text
	// star matchs also empty string
	if i >= len(pat) || str == "" {
		return i, len(str) + 1, true
	}
	for j < len(str) {
		if ok = match(str[j:], pat[i:]); ok {
			break
		}
		j++
	}
	return i, j, ok
}

func charsetMatch(char byte, pat string) (int, bool) {
	var (
		i     int
		match bool
		neg   = pat[0] == caret
	)
	if neg {
		i++
	}
	for ; pat[i] != rsquare; i++ {
		if pat[i] == dash {
			if p, n := pat[i-1], pat[i+1]; isRange(p, n) && char >= p && char <= n {
				match = true
				break
			}
		}
		if match = char == pat[i]; match {
			break
		}
	}
	for ; pat[i] != rsquare; i++ {
	}
	if neg {
		match = !match
	}
	return i, match
}

func isRange(prev, next byte) bool {
	return prev < next && acceptRange(prev) && acceptRange(next)
}

func acceptRange(b byte) bool {
	return (b >= 'a' || b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
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
