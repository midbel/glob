package glob

import (
	"testing"
)

type MatchCase struct {
	Pattern string
	Input   string
	Match   bool
}

func TestMatch(t *testing.T) {
	t.Run("simple", testMatchSimple)
	t.Run("extended", testMatchExtended)
}

func testMatchExtended(t *testing.T) {
	data := []MatchCase{
		{Input: "foo", Pattern: "(foo|bar)", Match: true},
		{Input: "bar", Pattern: "(foo|bar)", Match: true},
		{Input: "foo", Pattern: "+(foo)", Match: true},
		{Input: "foobar", Pattern: "+(foo|bar)", Match: true},
		{Input: "foo", Pattern: "*(foo)", Match: true},
		{Input: "foofoo", Pattern: "*(foo)", Match: true},
		{Input: "", Pattern: "*(foo)", Match: true},
		{Input: "foo", Pattern: "?(foo)", Match: true},
		{Input: "", Pattern: "?(foo)", Match: true},
		{Input: "foobar", Pattern: "?(foo|bar)", Match: false},
		{Input: "foobar", Pattern: "!(foo|bar)", Match: false},
		{Input: "github.com", Pattern: "g*.(com|org)", Match: true},
		{Input: "golang.org", Pattern: "g*.(com|org)", Match: true},
	}
	testMatchCases(t, data)
}

func testMatchSimple(t *testing.T) {
	data := []MatchCase{
		{Input: "", Pattern: "*", Match: true},
		{Input: "foo", Pattern: "*", Match: true},
		{Input: "foo", Pattern: "*foo", Match: true},
		{Input: "foo", Pattern: "foo*", Match: true},
		{Input: "foo", Pattern: "*foo*", Match: true},
		{Input: "foobar", Pattern: "*foo*", Match: true},
		{Input: "foo", Pattern: "???", Match: true},
		{Input: "foo", Pattern: "f?o", Match: true},
		{Input: "foobar", Pattern: "???", Match: false},
		{Input: "foobar", Pattern: "???bar", Match: true},
		{Input: "foobar", Pattern: "*.*", Match: false},
		{Input: "foo.bar", Pattern: "*.*", Match: true},
		{Input: "foo.bar", Pattern: "f*.bar", Match: true},
		{Input: "src/github.com/midbel/glob/glob.go", Pattern: "src/**/midbel/*/*go", Match: true},
		{Input: "src/github.com/midbel/glob/glob.go", Pattern: "**/*go", Match: true},
		{Input: "src/github.com/midbel/glob/glob.md", Pattern: "src/**/midbel/*/*go", Match: false},
		{Input: "src/github.com/midbel/glob/glob.md", Pattern: "**/*go", Match: false},
		{Input: "foobar", Pattern: "foo[abc]?r", Match: true},
		{Input: "foobar", Pattern: "foo[!abc]?r", Match: false},
		{Input: "foobar", Pattern: "foo[!xyz]?r", Match: true},
		{Input: "foobar", Pattern: "[a-z]oobar", Match: true},
		{Input: "foobar", Pattern: "[A-Z]oobar", Match: false},
		{Input: "foobar", Pattern: "f[az-]obar", Match: false},
		{Input: "foo-bar", Pattern: "foo[a-z-]bar", Match: true},
		{Input: "foo-bar", Pattern: "foo[!a-z-]bar", Match: false},
		{Input: "GMT287/S_FOO_BAR_19_287_00_43", Pattern: "GMT???/S_*[0-9][0-9]", Match: true},
		{Input: "GMT287/S_FOO_BAR_19_287_00_43.DAT", Pattern: "GMT???/S_*[0-9][0-9]", Match: false},
		{Input: "foo*bar", Pattern: `foo\*bar`, Match: true},
	}
	testMatchCases(t, data)
}

func testMatchCases(t *testing.T, data []MatchCase) {
	t.Helper()
	for i, d := range data {
		err := Match(d.Input, d.Pattern)
		if d.Match && err != nil {
			t.Errorf("%d) match failed: %s (%s)", i, d.Input, d.Pattern)
		}
	}
}
