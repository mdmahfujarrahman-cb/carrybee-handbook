package content

import (
	"fmt"
	"strconv"
)

func resolves(ref Ref, r Resolver) bool {
	switch ref.Kind {
	case kindTerm:
		_, ok := r.TermName(ref.ID)
		return ok
	case kindModule:
		_, ok := r.ModuleDesc(ref.App, ref.ID)
		return ok
	case kindStatus:
		id, err := strconv.Atoi(ref.ID)
		if err != nil {
			return false
		}
		_, ok := r.StatusName(id)
		return ok
	}
	return false
}

// Check reports references in pages that do not resolve.
func Check(pages []Page, r Resolver) []string {
	var problems []string
	for _, p := range pages {
		for _, ref := range p.Refs {
			if resolves(ref, r) {
				continue
			}
			name := ref.ID
			if ref.App != "" {
				name = ref.App + "/" + ref.ID
			}
			problems = append(problems, fmt.Sprintf("content/%s.md: unknown %s %q", p.Path, ref.Kind, name))
		}
	}
	return problems
}
