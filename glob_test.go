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
	}
	for i, d := range data {
		got := Match(d.Line, d.Pattern)
		if got != d.Want {
			t.Errorf("%d) match failed: %s (%s)", i, d.Line, d.Pattern)
		}
	}
}
