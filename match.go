package glob

import (
	"errors"
	"strings"
)

var (
	ErrMatch   = errors.New("match")
	ErrPattern = errors.New("mismatch")
)

type Matcher interface {
	Match(string) (Matcher, error)

	is(string) bool
}

func Match(str, pattern string) error {
	m, err := Compile(pattern)
	if err != nil {
		return err
	}
	parts := strings.Split(strings.Trim(str, "/"), "/")
	for i := 0; i < len(parts); i++ {
		if m == nil {
			return ErrPattern
		}
		m, err = m.Match(parts[i])
		if err != nil && !errors.Is(err, ErrMatch) {
			return err
		}
	}
	if m != nil {
		err = ErrPattern
	}
	return err
}

func match(str, pat string) (int, bool) {
	// shortcut: pat is only one star or pat and str are identicals
	if pat == string(star) || (len(str) == len(pat) && str == pat) {
		return len(str), true
	}
	var i, j int
	for ; i < len(pat); i++ {
		if j >= len(str) && pat[i] != star {
			break
		}
		switch char := pat[i]; char {
		case star:
			ni, nj, ok := starMatch(str[j:], pat[i:])
			if ok {
				return len(str), ok
			}
			i += ni
			j += nj
		case mark:
			// match a single character
		case lsquare:
			n, ok := charsetMatch(str[j], pat[i+1:])
			if !ok {
				return j, false
			}
			i += n + 1
		default:
			if char == backslash {
				i++
			}
			if pat[i] != str[j] {
				return j, false
			}
		}
		if j >= len(str) {
			break
		}
		j++
	}
	// we have a match when all characters of pattern and text have been read
	return j, i == len(pat) && j >= len(str)
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
	if i >= len(pat) && str == "" {
		return i, len(str) + 1, true
	}
	for j < len(str) {
		if _, ok = match(str[j:], pat[i:]); ok {
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
		neg   = pat[0] == bang || pat[0] == caret
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
