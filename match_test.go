package glob

import (
	"testing"
)

func TestMatch(t *testing.T) {
	data := []struct {
		Pattern string
		Line    string
		Match   bool
	}{
		{Line: "", Pattern: "*", Match: true},
		{Line: "foo", Pattern: "*", Match: true},
		{Line: "foo", Pattern: "*foo", Match: true},
		{Line: "foo", Pattern: "foo*", Match: true},
		{Line: "foo", Pattern: "*foo*", Match: true},
		{Line: "foobar", Pattern: "*foo*", Match: true},
		{Line: "foo", Pattern: "???", Match: true},
		{Line: "foo", Pattern: "f?o", Match: true},
		{Line: "foobar", Pattern: "???", Match: false},
		{Line: "foobar", Pattern: "???bar", Match: true},
		{Line: "foobar", Pattern: "*.*", Match: false},
		{Line: "foo.bar", Pattern: "*.*", Match: true},
		{Line: "foo.bar", Pattern: "f*.bar", Match: true},
		{Line: "src/github.com/midbel/glob/glob.go", Pattern: "src/**/midbel/*/*go", Match: true},
		{Line: "src/github.com/midbel/glob/glob.go", Pattern: "**/*go", Match: true},
		{Line: "src/github.com/midbel/glob/glob.md", Pattern: "src/**/midbel/*/*go", Match: false},
		{Line: "src/github.com/midbel/glob/glob.md", Pattern: "**/*go", Match: false},
		{Line: "foobar", Pattern: "foo[abc]?r", Match: true},
		{Line: "foobar", Pattern: "foo[!abc]?r", Match: false},
		{Line: "foobar", Pattern: "foo[!xyz]?r", Match: true},
		{Line: "foobar", Pattern: "[a-z]oobar", Match: true},
		{Line: "foobar", Pattern: "[A-Z]oobar", Match: false},
		{Line: "foobar", Pattern: "f[az-]obar", Match: false},
		{Line: "foo-bar", Pattern: "foo[a-z-]bar", Match: true},
		{Line: "foo-bar", Pattern: "foo[!a-z-]bar", Match: false},
		{Line: "GMT287/S_FOO_BAR_19_287_00_43", Pattern: "GMT???/S_*[0-9][0-9]", Match: true},
		{Line: "GMT287/S_FOO_BAR_19_287_00_43.DAT", Pattern: "GMT???/S_*[0-9][0-9]", Match: false},
		{Line: "foo*bar", Pattern: `foo\*bar`, Match: true},
	}
	for i, d := range data {
		err := Match(d.Line, d.Pattern)
		if d.Match && err != nil {
			t.Errorf("%d) match failed: %s (%s)", i, d.Line, d.Pattern)
		}
	}
}
