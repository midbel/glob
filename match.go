package glob

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

var (
	ErrMatch   = errors.New("match")
	ErrPattern = errors.New("mismatch")
)

type Matcher interface {
	fmt.Stringer

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

type simple struct {
	pattern string
}

func (s *simple) String() string {
	return fmt.Sprintf("simple(%s)", s.pattern)
}

func (s *simple) Match(str string) (Matcher, error) {
	var (
		err   error
		_, ok = match(str, s.pattern)
	)
	if !ok {
		err = ErrPattern
	}
	return nil, err
}

func (s *simple) is(str string) bool {
	return str == s.pattern
}

type group struct {
	ms []Matcher
}

func (g *group) String() string {
	var buf strings.Builder
	buf.WriteString("group(")
	for i, m := range g.ms {
		if i > 0 {
			buf.WriteRune(pipe)
		}
		buf.WriteString(m.String())
	}
	buf.WriteRune(rparen)
	return buf.String()
}

func (g *group) Match(str string) (Matcher, error) {
	var (
		match bool
		next  = make([]Matcher, 0, len(g.ms))
	)
	for _, m := range g.ms {
		x, err := m.Match(str)
		if err == nil || errors.Is(err, ErrMatch) {
			match = true
			if x != nil {
				next = append(next, x)
			}
		}
	}
	if !match {
		return nil, ErrPattern
	}
	var x Matcher
	switch len(next) {
	case 0:
	case 1:
		x = next[0]
	default:
		x = &group{ms: next}
	}
	return x, nil
}

func (g *group) is(_ string) bool {
	return false
}

type multiple struct {
	ms []Matcher
}

func (m *multiple) String() string {
	var buf strings.Builder
	buf.WriteString("group(")
	for _, m := range m.ms {
		buf.WriteString(m.String())
	}
	buf.WriteRune(rparen)
	return buf.String()
}

func (m *multiple) Match(str string) (Matcher, error) {
	var offset int
	for _, m := range m.ms {
		if offset >= len(str) {
			return nil, ErrPattern
		}

		mr, mok := m.(interface{ more() bool })

		var (
			multi bool
			match bool
			limit = len(str)
		)
		for {
			if _, err := m.Match(str[offset:limit]); err == nil {
				if mok && mr.more() {
					multi = true
					continue
				}
				match, offset = true, limit
				break
			}
			k, n := utf8.DecodeLastRuneInString(str[offset:limit])
			if k == utf8.RuneError {
				break
			}
			limit -= n
		}
		if multi {
			match = multi
		}
		if !match {
			offset = 0
			break
		}
	}
	if offset == len(str) {
		return nil, nil
	}
	return nil, ErrPattern
}

func (m *multiple) is(_ string) bool {
	return false
}

type any struct {
	min   int
	max   int
	inner Matcher

	matched int
}

func (a *any) String() string {
	return fmt.Sprintf("any(%s)", a.inner)
}

func (a *any) Match(str string) (Matcher, error) {
	var (
		match  int
		offset int
	)
	for {
		if offset >= len(str) {
			break
		}
		base := offset
		_, n := utf8.DecodeRuneInString(str[base:])
		if n <= 0 {
			break
		}
		offset += n
		for offset <= len(str) {
			if _, err := a.inner.Match(str[base:offset]); err == nil {
				match++
				break
			}
			k, n := utf8.DecodeRuneInString(str[base:])
			if k == utf8.RuneError {
				break
			}
			offset += n
		}
		if a.max > 0 && match >= a.max {
			break
		}
	}
	ok := offset <= len(str) && match >= a.min
	if ok && a.max > 0 {
		ok = match <= a.max
	}
	if ok {
		a.matched++
		return nil, nil
	}
	return nil, ErrPattern
}

func (a *any) more() bool {
	return a.matched < a.min || a.max == 0 || a.matched < a.max
}

func (a *any) is(_ string) bool {
	return false
}

type not struct {
	inner Matcher
}

func (n *not) String() string {
	return fmt.Sprintf("not(%s)", n.inner)
}

func (n *not) Match(str string) (Matcher, error) {
	_, err := n.inner.Match(str)
	if err == nil {
		err = ErrPattern
	} else {
		err = nil
	}
	return nil, err
}

func (n *not) is(_ string) bool {
	return false
}

type element struct {
	head Matcher
	next Matcher
}

func (e *element) String() string {
	return fmt.Sprintf("element(%s)", e.head)
}

func (e *element) Match(str string) (Matcher, error) {
	if e == nil || e.head == nil {
		return nil, ErrPattern
	}
	if e.head.is("**") && e.next != nil {
		if m, err := e.next.Match(str); err == nil {
			return m, nil
		}
		return e, nil
	}
	m, err := e.head.Match(str)
	if err == nil || errors.Is(err, ErrMatch) {
		if m == nil {
			m = e.next
		} else {
			m = &element{
				head: m,
				next: e.next,
			}
		}
	}
	return m, err
}

func (e *element) is(str string) bool {
	return e.head.is(str)
}

func match(str, pat string) (int, bool) {
	// shortcut: pat is only one star or pat and str are identicals
	if pat == string(star) || (len(str) == len(pat) && str == pat) {
		return len(str), true
	}
	var i, j int
	for i < len(pat) {
		k, n := utf8.DecodeRuneInString(pat[i:])
		if k == utf8.RuneError {
			if n == 0 {
				break
			}
			if n < 0 {
				return -1, false
			}
		}
		if j >= len(str) && k != star {
			break
		}
		i += n
		switch k {
		case star:
			ni, nj, ok := starMatch(str[j:], pat[i:])
			if ok {
				return len(str), ok
			}
			i += ni
			j += nj
		case mark:
			// match a single character
			_, n := utf8.DecodeRuneInString(str[j:])
			if n > 0 {
				j += n
			}
		case lsquare:
			char, nj := utf8.DecodeRuneInString(str[j:])
			ni, ok := charsetMatch(char, pat[i:])
			if !ok {
				return j, false
			}
			j += nj
			i += ni
		default:
			if k == backslash {
				k, n = utf8.DecodeRuneInString(pat[i:])
				i += n
			}
			char, n := utf8.DecodeRuneInString(str[j:])
			if k != char {
				return j, false
			}
			j += n
		}
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
	for {
		k, n := utf8.DecodeRuneInString(pat[i:])
		if k != star {
			break
		}
		i += n
	}
	// trailing star matchs rest of text - star matchs also empty string
	if i >= len(pat) && str == "" {
		return i, len(str) + 1, true
	}
	for j < len(str) {
		if _, ok = match(str[j:], pat[i:]); ok {
			break
		}
		k, n := utf8.DecodeRuneInString(str[j:])
		if k == utf8.RuneError {
			break
		}
		j += n
	}
	return i, j, ok
}

func charsetMatch(char rune, pat string) (int, bool) {
	next, n := utf8.DecodeRuneInString(pat)
	var (
		i      int
		match  bool
		prev   rune
		negate = next == bang || next == caret
	)
	if negate {
		i += n
	}
	for {
		k, n := utf8.DecodeRuneInString(pat[i:])
		i += n
		if k == rsquare || k == utf8.RuneError {
			prev = k
			break
		}
		if k == dash {
			next, n = utf8.DecodeRuneInString(pat[i:])
			if isRange(prev, next) && char >= prev && char <= next {
				match = true
				break
			}
		}
		if match = char == k; match {
			break
		}
		prev = k
	}
	if prev != rsquare && prev != utf8.RuneError {
		for {
			k, n := utf8.DecodeRuneInString(pat[i:])
			i += n
			if k == rsquare || k == utf8.RuneError {
				break
			}
		}
	}
	if negate {
		match = !match
	}
	return i, match
}

func isRange(prev, next rune) bool {
	return prev < next && acceptRange(prev) && acceptRange(next)
}

func acceptRange(b rune) bool {
	return (b >= 'a' || b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}
