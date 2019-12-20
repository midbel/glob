package glob

import (
	"testing"
)

func TestCompile(t *testing.T) {
	data := []struct {
		Pattern string
		Fail    bool
	}{}
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
