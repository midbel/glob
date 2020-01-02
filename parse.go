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
		case arobase:
			if z, _, _ := r.ReadRune(); z != lparen {
				buf.WriteRune(k)
				r.UnreadRune()
				continue
			}
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
			r.UnreadRune()
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
		fmt.Printf("%selement(\n", indent)
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
