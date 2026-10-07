package scan

import (
	"regexp"
	"strconv"
)

var (
	transferEnumRE = regexp.MustCompile(`(?s)enum\s+TransferStatus\s*\{(.*?)\}`)
	enumMemberRE   = regexp.MustCompile(`(?m)^\s*([A-Za-z][A-Za-z0-9]*)\s*=\s*(\d+)`)
)

type Status struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// parseStatuses reads members of `enum TransferStatus` from comment-stripped source.
func parseStatuses(src string) []Status {
	m := transferEnumRE.FindStringSubmatch(src)
	if m == nil {
		return nil
	}
	var out []Status
	for _, mm := range enumMemberRE.FindAllStringSubmatch(m[1], -1) {
		id, _ := strconv.Atoi(mm[2])
		out = append(out, Status{ID: id, Name: mm[1]})
	}
	return out
}
