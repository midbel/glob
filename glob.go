package glob

const (
	slash     = '/'
	backslash = '\\'
)

const (
	star = '*'
	mark = '?'
	any  = "**"
)

func Walk(pattern string) ([]string, error) {
	return nil, nil
}

func Glob(dir, pattern string) ([]string, error) {
	parts := strings.FieldsFunc(pattern, func(r rune) bool { return r == slash || r == backslash })
	return glob(dir, parts)
}

func Match(dir, pattern string) bool {
	return false
}

func glob(dir, parts []string) ([]string, error) {
	return nil, nil
}
