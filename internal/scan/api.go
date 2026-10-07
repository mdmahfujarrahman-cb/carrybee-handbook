package scan

import "regexp"

var apiRE = regexp.MustCompile(`API_ROUTES\.(\w+)(?:\.(\w+))?`)

func apiDomains(text string, nested []string) map[string]int {
	nest := map[string]bool{}
	for _, n := range nested {
		nest[n] = true
	}
	out := map[string]int{}
	for _, m := range apiRE.FindAllStringSubmatch(text, -1) {
		d := m[1]
		if nest[d] && m[2] != "" {
			d += "." + m[2]
		}
		out[d]++
	}
	return out
}
