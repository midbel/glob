package glob

import (
	"fmt"
	"strings"
)

type Matcher interface {
	Match(string) bool
	fmt.Stringer
}

type not struct {
	Matcher
}

func (n not) Match(str string) bool {
	return !n.Matcher.Match(str)
}

func (n not) String() string {
	return fmt.Sprintf("not(%s)", n.Matcher.String())
}

type simple string

func (i simple) Match(str string) bool {
	return string(i) == str
}

func (i simple) String() string {
	return fmt.Sprintf("simple(%s)", string(i))
}

type group struct {
	ms []Matcher
}

func (g group) Match(str string) bool {
	for i := range g.ms {
		if g.ms[i].Match(str) {
			return true
		}
	}
	return false
}

func (g group) String() string {
	var buf strings.Builder

	buf.WriteString("group(")
	for i, m := range g.ms {
		if i > 0 {
			buf.WriteRune(pipe)
		}
		switch m := m.(type) {
		case group, multiple, list, simple, not:
			buf.WriteString(m.String())
		case nil:
			buf.WriteString("<nil>")
		default:
			buf.WriteString("<unknown>")
		}
	}
	buf.WriteRune(rparen)
	return buf.String()
}

type list []Matcher

func (ms list) Match(str string) bool {
	for i := range ms {
		if !ms[i].Match(str) {
			return false
		}
	}
	return true
}

func (ms list) String() string {
	var buf strings.Builder
	buf.WriteString("list(")
	for i, m := range ms {
		if i > 0 {
			buf.WriteRune(slash)
		}
		buf.WriteString(m.String())
	}
	buf.WriteRune(rparen)
	return buf.String()
}

type multiple []Matcher

func (ms multiple) Match(str string) bool {
	for i := range ms {
		if !ms[i].Match(str) {
			return false
		}
	}
	return true
}

func (ms multiple) String() string {
	var buf strings.Builder
	buf.WriteString("multiple(")
	for _, m := range ms {
		buf.WriteString(m.String())
	}
	buf.WriteRune(rparen)
	return buf.String()
}

// *?+()
type any struct {
	min int
	max int
	Matcher
}

func (a any) Match(str string) bool {
	var i int
	for {
		ok := a.Matcher.Match(str)
		if ok {
			i++
		} else {
			break
		}
		if a.max > 0 && i == a.max {
			break
		}
	}
	return i >= a.min && i <= a.max
}

func (a any) String() string {
	return fmt.Sprintf("any(%s)", a.Matcher)
}

func Debug(m Matcher, level int) {
	indent := strings.Repeat(" ", (level * 2))
	switch m := m.(type) {
	case list:
		fmt.Printf("%spath(\n", indent)
		for i := range m {
			Debug(m[i], level+1)
		}
		fmt.Printf("%s)\n", indent)
	case group:
		fmt.Printf("%sgroup(\n", indent)
		for i := range m.ms {
			Debug(m.ms[i], level+1)
		}
		fmt.Printf("%s)\n", indent)
	case simple:
		fmt.Printf("%s%s\n", indent, m.String())
	case not:
		fmt.Printf("%snot(\n", indent)
		Debug(m.Matcher, level+1)
		fmt.Printf("%s)\n", indent)
	case multiple:
		fmt.Printf("%smultiple(\n", indent)
		for i := range m {
			Debug(m[i], level+1)
		}
		fmt.Printf("%s)\n", indent)
	case any:
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
		Debug(m.Matcher, level+1)
		fmt.Printf("%s)\n", indent)
	}
}
