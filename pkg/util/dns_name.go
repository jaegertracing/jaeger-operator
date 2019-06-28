package util

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var regex = regexp.MustCompile(`[a-z0-9]`)

// DNSName returns a dns-safe string for the given name.
// Any char that is not [a-z0-9] is replaced by "-".
// If the final name starts with "-", "a" is added as prefix. Similarly, if it ends with "-", "z" is added.
func DNSName(name string) string {
	var d []rune

	first := true
	for _, x := range strings.ToLower(name) {
		if regex.Match([]byte(string(x))) {
			d = append(d, x)
		} else {
			if first {
				d = append(d, 'a')
			}
			d = append(d, '-')

			if len(d) == utf8.RuneCountInString(name) {
				// we had to replace the last char, so, it's "-". DNS names can't end with dash.
				d = append(d, 'z')
			}
		}

		first = false
	}

	return string(d)
}
