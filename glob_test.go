package glob

import (
	"testing"
)

func TestMatch(t *testing.T) {
	data := []struct {
		Pattern string
		Line    string
		Want    bool
	}{
		{Line: "", Pattern: "*", Want: true},
		{Line: "foo", Pattern: "*", Want: true},
		{Line: "foo", Pattern: "*foo", Want: true},
		{Line: "foo", Pattern: "foo*", Want: true},
		{Line: "foo", Pattern: "*foo*", Want: true},
		{Line: "foobar", Pattern: "*foo*", Want: true},
		{Line: "foo", Pattern: "???", Want: true},
		{Line: "foo", Pattern: "f?o", Want: true},
		{Line: "foobar", Pattern: "???", Want: false},
		{Line: "foobar", Pattern: "???bar", Want: true},
		{Line: "foobar", Pattern: "*.*", Want: false},
		{Line: "foo.bar", Pattern: "*.*", Want: true},
		{Line: "src/github.com/midbel/glob/glob.go", Pattern: "src/**/midbel/*/*go", Want: true},
		{Line: "src/github.com/midbel/glob/glob.go", Pattern: "**/*go", Want: true},
		{Line: "src/github.com/midbel/glob/glob.md", Pattern: "src/**/midbel/*/*go", Want: false},
		{Line: "src/github.com/midbel/glob/glob.md", Pattern: "**/*go", Want: false},
		{Line: "foobar", Pattern: "foo[abc]?r", Want: true},
		{Line: "foobar", Pattern: "foo[^abc]?r", Want: false},
		{Line: "foobar", Pattern: "foo[^xyz]?r", Want: true},
		{Line: "foobar", Pattern: "[a-z]oobar", Want: true},
		{Line: "foobar", Pattern: "[A-Z]oobar", Want: false},
		{Line: "foobar", Pattern: "f[az-]obar", Want: false},
		{Line: "foo-bar", Pattern: "foo[a-z-]bar", Want: true},
		{Line: "foo-bar", Pattern: "foo[^a-z-]bar", Want: false},
		{Line: "GMT287/S_FOO_BAR_19_287_00_43", Pattern: "GMT???/S_*[0-9][0-9]", Want: true},
		{Line: "GMT287/S_FOO_BAR_19_287_00_43.DAT", Pattern: "GMT???/S_*[0-9][0-9]", Want: false},
		{Line: "foo*bar", Pattern: `foo\*bar`, Want: true},
	}
	for i, d := range data {
		got := Match(d.Line, d.Pattern)
		if got != d.Want {
			t.Errorf("%d) match failed: %s (%s)", i, d.Line, d.Pattern)
		}
	}
}
