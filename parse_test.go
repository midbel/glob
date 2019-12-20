package glob

import (
	"testing"
)

func TestCompile(t *testing.T) {
	data := []struct {
		Pattern string
		Fail    bool
	}{
		{Pattern: "", Fail: true},
		{Pattern: "(github|golang).(org|com)", Fail: false},
		{Pattern: "(github|golang).!(org|com)", Fail: false},
		{Pattern: "g*.(org|com)", Fail: false},
		{Pattern: "+(ab|cd)", Fail: false},
		{Pattern: "*(ab|cd)", Fail: false},
		{Pattern: "?(ab|cd)", Fail: false},
		{Pattern: "github.com/(midbel/glob|midbel/cbor)/**/*.!(go)", Fail: false},
		{Pattern: "git(hub|lab).(com|org)/(midbel|busoc)", Fail: false},
	}
	for i, d := range data {
		_, err := Compile(d.Pattern)
		if d.Fail == false && err != nil {
			t.Errorf("%d) compile fail %q: %v", i, d.Pattern, err)
		}
		if d.Fail == true && err == nil {
			t.Errorf("%d) compile fail %q", i, d.Pattern)
		}
	}
}
