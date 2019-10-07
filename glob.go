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

func Match(dir, pattern string) bool {
	return match(dir, pattern)
}

func match(str, pat string) bool {
	// shortcut: pat is only one star or pat and str are identicals
	if pat == string(star) || (len(str) == len(pat) && str == pat) {
		return true
	}
	var i, j int
	// for ; i < len(pat) && j < len(str); i++ {
	for ; i < len(pat); i++ {
		switch char := pat[i]; {
		case char == star:
			// multiple stars is the same as one star
			for i = i + 1; i < len(pat) && pat[i] == char; i++ {
			}
			// trailing star matchs rest of text
			if i >= len(pat) {
				return true
			}
			for j < len(str) {
				if ok := match(str[j:], pat[i:]); ok {
					return ok
				}
				j++
			}
		case char == mark || (j < len(str) && char == str[j]):
			// default
		default:
			return false
		}
		j++
	}
	// match when all characters of pattern and text have been read
	return i == len(pat) && j == len(str)
}
