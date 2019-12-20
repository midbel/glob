package glob

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

func Compile(pattern string) (Matcher, error) {
	pattern = strings.TrimSpace(pattern)
	if len(pattern) == 0 {
		return nil, fmt.Errorf("empty pattern")
	}
	pattern = strings.ReplaceAll(pattern, "\r\n", "\n")
	return parseReader(strings.NewReader(pattern))
}

func Debug(m Matcher) {
	debug(m, 0)
}

const (
	slash     = '/'
	backslash = '\\'
	dash      = '-'
	star      = '*'
	mark      = '?'
	lsquare   = '['
	rsquare   = ']'
	bang      = '!'
	caret     = '^'
	lparen    = '('
	rparen    = ')'
	pipe      = '|'
	arobase   = '@'
	plus      = '+'
	newline   = '\n'
	tab       = '\t'
	space     = ' '
)

type simple struct {
	pattern string
}

func (s *simple) String() string {
	return fmt.Sprintf("Simple(%s)", s.pattern)
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
	buf.WriteString("Group(")
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
	for _, m := range g.ms {
		x, err := m.Match(str)
		if err == nil || errors.Is(err, ErrMatch) {
			return x, nil
		}
	}
	return nil, ErrPattern
}

func (g *group) is(_ string) bool {
	return false
}

type multiple struct {
	ms []Matcher
}

func (m *multiple) String() string {
	var buf strings.Builder
	buf.WriteString("Multi(")
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
		)
		for i := len(str); i > offset; i-- {
			if _, err := m.Match(str[offset:i]); err == nil {
				if mok && mr.more() {
					multi = true
					continue
				}
				match, offset = true, i
				break
			}
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
	return fmt.Sprintf("Any(%s)", a.inner.String())
}

func (a *any) Match(str string) (Matcher, error) {
	var (
		match  int
		offset int
	)
	for {
		if offset >= len(str) {
			return nil, ErrPattern
		}
		base := offset
		offset++
		for offset <= len(str) {
			if _, err := a.inner.Match(str[base:offset]); err == nil {
				match++
				break
			}
			offset++
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
	return fmt.Sprintf("Not(%s)", n.inner.String())
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
	if e.head == nil {
		return "Nil()"
	}
	return fmt.Sprintf("Element(%s)", e.head)
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

func parseReader(r *strings.Reader) (Matcher, error) {
	var (
		buf strings.Builder
		ms  []Matcher
		cs  []Matcher
	)

	for {
		k, _, err := r.ReadRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		if k == pipe || k == rparen {
			r.UnreadRune()
			break
		}
		switch k {
		case backslash:
			if k, _, _ = r.ReadRune(); k != newline {
				buf.WriteRune(backslash)
			} else {
				for {
					k, _, _ = r.ReadRune()
					if k != space && k != tab && k != newline {
						break
					}
				}
			}
			r.UnreadRune()
		case lparen:
			if buf.Len() > 0 {
				cs = append(cs, &simple{pattern: buf.String()})
				buf.Reset()
			}
			g, err := parseGroup(r)
			if err != nil {
				return nil, err
			}
			cs = append(cs, g)
		case bang:
      if z, _, _ := r.ReadRune(); z != lparen {
				buf.WriteRune(k)
				r.UnreadRune()
				continue
			}
			if buf.Len() > 0 {
				cs = append(cs, &simple{pattern: buf.String()})
				buf.Reset()
			}
			n, err := parseNot(r)
			if err != nil {
				return nil, err
			}
			cs = append(cs, n)
		case plus, star, mark:
			if z, _, _ := r.ReadRune(); z != lparen {
				buf.WriteRune(k)
				r.UnreadRune()
				continue
			}
			if buf.Len() > 0 {
				cs = append(cs, &simple{pattern: buf.String()})
				buf.Reset()
			}
			a, err := parseAny(r, k)
			if err != nil {
				return nil, err
			}
			cs = append(cs, a)
		case slash:
			if buf.Len() > 0 {
				cs = append(cs, &simple{pattern: buf.String()})
				buf.Reset()
			}
			if m := mergeMatchers(cs); m != nil {
				ms = append(ms, m)
			}
			cs = cs[:0]
		default:
			buf.WriteRune(k)
		}
	}
	if buf.Len() > 0 {
		cs = append(cs, &simple{pattern: buf.String()})
	}
	if m := mergeMatchers(cs); m != nil {
		ms = append(ms, m)
	}
	return linkMatchers(ms), nil
}

func parseAny(r *strings.Reader, k rune) (Matcher, error) {
	m, err := parseGroup(r)
	if err != nil {
		return nil, err
	}
	var a any

	a.inner = m
	switch k {
	default:
		a.min, a.max = 0, 0
	case mark:
		a.min, a.max = 0, 1
	case plus:
		a.min, a.max = 1, 0
	}
	return &a, nil
}

func parseNot(r *strings.Reader) (Matcher, error) {
	k, _, err := r.ReadRune()
	if err != nil {
		return nil, err
	}
	if k != lparen {
		return nil, fmt.Errorf("expecting (, got %c (position: %d)", k, int(r.Size())-r.Len())
	}
	m, err := parseGroup(r)
	if err != nil {
		return nil, err
	}
	return &not{inner: m}, nil
}

func parseGroup(r *strings.Reader) (Matcher, error) {
	var grp group

Loop:
	for {
		m, err := parseReader(r)
		if err != nil {
			return nil, err
		}

		grp.ms = append(grp.ms, m)
		k, _, err := r.ReadRune()
		if err != nil {
			return nil, err
		}
		switch k {
		case pipe:
		case rparen:
			break Loop
		default:
			return nil, fmt.Errorf("unexpected character %c", k)
		}
	}
	return &grp, nil
}

func linkMatchers(ms []Matcher) Matcher {
	n := len(ms)
	if n == 0 {
		return nil
	}
	e := &element{
		head: ms[n-1],
	}
	n--
	for i := n - 1; i >= 0; i-- {
		e = &element{
			head: ms[i],
			next: e,
		}
	}
	return e
}

func mergeMatchers(cs []Matcher) Matcher {
	var m Matcher
	if len(cs) == 1 {
		m = cs[0]
	} else if len(cs) > 1 {
		xs := make([]Matcher, len(cs))
		copy(xs, cs)
		m = &multiple{ms: xs}
	}
	return m
}

func debug(m Matcher, level int) {
	indent := strings.Repeat(" ", level*2)
	switch m := m.(type) {
	case *element:
		if m == nil {
			return
		}
		fmt.Printf("%selement[%v](\n", indent, m.next)
		debug(m.head, level+1)
		fmt.Printf("%s)\n", indent)
		if m.next != nil {
			debug(m.next, level)
		}
	case *simple:
		fmt.Printf("%ssimple(pattern=%s)\n", indent, m.pattern)
	case *multiple:
		fmt.Printf("%smultiple(\n", indent)
		for i := range m.ms {
			debug(m.ms[i], level+1)
		}
		fmt.Printf("%s)\n", indent)
	case *not:
		fmt.Printf("%snot(\n", indent)
		debug(m.inner, level+1)
		fmt.Printf("%s)\n", indent)
	case *group:
		fmt.Printf("%sgroup(\n", indent)
		for i, m := range m.ms {
			fmt.Printf("%s  item[%d](\n", indent, i+1)
			debug(m, level+2)
			fmt.Printf("%s  )\n", indent)
		}
		fmt.Printf("%s)\n", indent)
	case *any:
		var k rune = bang
		switch {
		case m.min == 0 && m.max == 0:
			k = star
		case m.min == 0 && m.max == 1:
			k = mark
		case m.min == 1 && m.max == 0:
			k = plus
		}
		fmt.Printf("%sany%c(\n", indent, k)
		debug(m.inner, level+1)
		fmt.Printf("%s)\n", indent)
	default:
		fmt.Printf("%sunknown(%T)\n", indent, m)
	}
}
