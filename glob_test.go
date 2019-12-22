package glob

import (
	"sort"
	"strings"
	"testing"
)

var files = []entry{
	{Name: "src/github.com/midbel/glob/glob.go"},
	{Name: "src/github.com/midbel/glob/glob_test.go"},
	{Name: "src/github.com/midbel/glob/parse.go"},
	{Name: "src/github.com/midbel/glob/parse_test.go"},
	{Name: "src/github.com/midbel/glob/match.go"},
	{Name: "src/github.com/midbel/glob/match_test.go"},
	{Name: "src/github.com/midbel/glob/README.md"},
	{Name: "src/github.com/midbel/glob/LICENCE"},
	{Name: "src/github.com/midbel/toml/README.md"},
	{Name: "src/github.com/midbel/toml/LICENCE"},
	{Name: "bin/testglob-linux64"},
	{Name: "bin/testglob-win64.exe"},
}

func init() {
	scan = scanmap
}

type GlobCase struct {
	Pattern string
	Base    string
	Files   []string
}

func TestGlob(t *testing.T) {
	data := []GlobCase{
		{
			Pattern: "bin/*exe",
			Files:   []string{"bin/testglob-win64.exe"},
		},
		{
			Pattern: "src/**/glob/*",
			Files: []string{
				"src/github.com/midbel/glob/glob.go",
				"src/github.com/midbel/glob/glob_test.go",
				"src/github.com/midbel/glob/parse.go",
				"src/github.com/midbel/glob/parse_test.go",
				"src/github.com/midbel/glob/match.go",
				"src/github.com/midbel/glob/match_test.go",
				"src/github.com/midbel/glob/README.md",
				"src/github.com/midbel/glob/LICENCE",
			},
		},
		{
			Pattern: "src/github.com/**/*.!(go)",
			Files: []string{
				"src/github.com/midbel/glob/README.md",
				"src/github.com/midbel/toml/README.md",
			},
		},
	}
	for i, d := range data {
		testGlobCase(t, d, i)
	}
}

func testGlobCase(t *testing.T, d GlobCase, i int) {
	g, err := New(d.Pattern, d.Base)
	if err != nil {
		t.Errorf("%d) invalid pattern %s: %v", i, d.Pattern, err)
		return
	}

	sort.Strings(d.Files)

	var j int
	for f := g.Glob(); f != ""; f = g.Glob() {
		if j >= len(d.Files) {
			t.Errorf("%d) too many files found (want: %d, got %d)", i, len(d.Files), j)
			return
		}
		f = strings.ReplaceAll(f, "\\", "/")
		ix := sort.SearchStrings(d.Files, f)
		if ix >= len(d.Files) {
			t.Errorf("%d) unexpected file %s", i, f)
			return
		}
		if d.Files[ix] != f {
			t.Errorf("%d) unexpected file %s", i, f)
			return
		}
		j++
	}
	if j < len(d.Files) {
		t.Errorf("%d) not enough files found (want: %d, got: %d)", i, len(d.Files), j)
	}
}

func scanmap(dir string) (<-chan entry, error) {
	queue := make(chan entry)
	go func() {
		defer close(queue)
		seen := make(map[string]struct{})

		dir = strings.ReplaceAll(dir, "\\", "/") + "/"
		for _, f := range files {
			if dir != "/" && !strings.HasPrefix(f.Name, dir) {
				continue
			}
			name := strings.TrimPrefix(f.Name, dir)
			if name == "" {
				continue
			}
			parts := strings.SplitN(name, "/", 2)
			if len(parts) < 1 || parts[0] == "" {
				continue
			}
			f.Name, f.Dir = parts[0], len(parts) != 1
			if _, ok := seen[f.Name]; !ok {
				queue <- f
				seen[f.Name] = struct{}{}
			}
		}
	}()
	return queue, nil
}
