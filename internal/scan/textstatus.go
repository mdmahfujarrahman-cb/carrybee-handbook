package scan

import (
	"regexp"
	"sort"
)

// primaryTextEnum lists its values first; other order status enums add the values it lacks.
const primaryTextEnum = "orderStatusSchema"

var (
	// textEnumRE matches order status enums such as orderStatusSchema or deliveryOrderStatusSchema.
	// Non-order status enums (hub, zone, payment: active/disabled, true/false, paid/unpaid) are not parcel statuses.
	textEnumRE  = regexp.MustCompile(`(?s)const\s+(\w*[Oo]rder\w*StatusSchema)\s*=\s*z\s*\.\s*enum\(\s*\[(.*?)\]`)
	textValueRE = regexp.MustCompile(`['"]([A-Za-z0-9_]+)['"]`)
)

type textEnum struct {
	Name   string
	Values []string
}

// parseTextStatuses reads `z.enum([...])` order status schemas from comment-stripped source.
func parseTextStatuses(src string) []textEnum {
	var out []textEnum
	for _, m := range textEnumRE.FindAllStringSubmatch(src, -1) {
		e := textEnum{Name: m[1]}
		for _, v := range textValueRE.FindAllStringSubmatch(m[2], -1) {
			e.Values = append(e.Values, v[1])
		}
		out = append(out, e)
	}
	return out
}

// textStatuses merges the order status enums of all files: the primary enum's values in source
// order, then values found only in other enums, in file path order. Never nil.
func textStatuses(files map[string]string) []string {
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	var primary, others []textEnum
	for _, p := range paths {
		for _, e := range parseTextStatuses(files[p]) {
			if e.Name == primaryTextEnum {
				primary = append(primary, e)
			} else {
				others = append(others, e)
			}
		}
	}
	out := []string{}
	seen := map[string]bool{}
	for _, e := range append(primary, others...) {
		for _, v := range e.Values {
			if !seen[v] {
				seen[v] = true
				out = append(out, v)
			}
		}
	}
	return out
}
