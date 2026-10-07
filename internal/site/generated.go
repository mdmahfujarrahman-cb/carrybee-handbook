package site

import (
	"fmt"
	"html/template"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"carrybee/handbook/internal/content"
)

const statusesPath = "business/statuses"

func statusPhase(id int) string {
	switch {
	case id >= 1 && id <= 7:
		return "First mile"
	case (id >= 8 && id <= 13) || id == 35 || id == 38 || id == 39:
		return "Middle mile"
	case id == 14 || id == 16:
		return "Last mile"
	case id == 15 || id == 18 || id == 19 || id == 20 || id == 21 || id == 22 || id == 32 || id == 33:
		return "Final"
	case (id >= 23 && id <= 31) || id == 34 || id == 40 || id == 41 || id == 17:
		return "Return"
	case id >= 42 && id <= 45:
		return "Reverse pickup"
	case id == 36 || id == 37:
		return "Recovery"
	case id == 53:
		return "Change request"
	}
	return "Other"
}

func (s *site) writeTerms() error {
	v := s.view("terms.html", "Courier terms")
	v.Section, v.Wide = "reference", true
	b, err := renderTemplate("terms.html", v)
	if err != nil {
		return err
	}
	return s.write("terms.html", b)
}

type apiRow struct {
	Domain, Anchor string
	Modules        []string
	Pages          []string
}

type appTable struct {
	App  string
	Rows []apiRow
}

func (s *site) apiTable(appID string) appTable {
	sc := s.scans[appID]
	t := appTable{App: appID}
	for d, mods := range sc.APIDomains {
		pages := map[string]bool{}
		for _, m := range mods {
			if mod := sc.Modules[m]; mod != nil {
				for r := range mod.UsedByRoutes {
					pages[r] = true
				}
			}
		}
		row := apiRow{Domain: d, Anchor: "api." + appID + "." + d, Modules: mods}
		for p := range pages {
			row.Pages = append(row.Pages, p)
		}
		sort.Strings(row.Pages)
		t.Rows = append(t.Rows, row)
	}
	sort.Slice(t.Rows, func(i, j int) bool { return t.Rows[i].Domain < t.Rows[j].Domain })
	return t
}

func (s *site) writeGenerated() error {
	var tables []appTable
	for _, app := range s.cfg.Apps {
		if sc := s.scans[app.ID]; sc != nil && sc.Error == "" {
			tables = append(tables, s.apiTable(app.ID))
		}
	}
	const out = "code/api-index.html"
	v := s.view(out, "API domain index")
	v.Section, v.Wide = sectionCode, true
	v.Summary = "Every backend API domain each app calls, the modules that call it and the pages that load those modules."
	v.Extra = map[string]any{"Tables": tables, "AppsRoot": Rel(out, "apps/")}
	b, err := renderTemplate("apiindex.html", v)
	if err != nil {
		return err
	}
	if err := s.write(out, b); err != nil {
		return err
	}
	return s.appendStatusTable()
}

var statusTermRE = regexp.MustCompile(`^(\d+)(?:\s*[–-]\s*(\d+))?\s`)

// statusTerms maps each status number to the "Order statuses" term that names it.
// A term for one number ("15 Delivered") wins over a range term ("23–34 Return chain").
func statusTerms(terms []content.Term) map[int]content.Term {
	exact, ranged := map[int]content.Term{}, map[int]content.Term{}
	for _, t := range terms {
		m := statusTermRE.FindStringSubmatch(t.Name)
		if t.Cat != "status" || m == nil {
			continue
		}
		lo, _ := strconv.Atoi(m[1])
		if m[2] == "" {
			exact[lo] = t
			continue
		}
		hi, _ := strconv.Atoi(m[2])
		for id := lo; id <= hi; id++ {
			ranged[id] = t
		}
	}
	for id, t := range ranged {
		if _, ok := exact[id]; !ok {
			exact[id] = t
		}
	}
	return exact
}

func (s *site) statusRows(out string) string {
	seen := map[int]string{}
	for _, sc := range s.scans {
		for _, st := range sc.Statuses {
			seen[st.ID] = st.Name
		}
	}
	ids := make([]int, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	terms := statusTerms(s.terms)
	var rows strings.Builder
	rows.WriteString(`<h2 id="all-statuses">Every status in code</h2><p>Read from <code>enum TransferStatus</code> at build time. Each row links to its entry in the courier terms.</p>`)
	rows.WriteString(`<div class="table-wrap"><table class="status-table"><thead><tr><th>#</th><th>Status</th><th>Name in code</th><th>Phase</th><th>Term</th></tr></thead><tbody>`)
	for _, id := range ids {
		term := "–"
		if t, ok := terms[id]; ok {
			term = fmt.Sprintf(`<a href="%s">%s</a>`, template.HTMLEscapeString(Rel(out, "terms.html#t."+t.ID)), template.HTMLEscapeString(t.Name))
		}
		fmt.Fprintf(&rows, `<tr id="s.%d"><td class="num">%d</td><td>%s</td><td><code>%s</code></td><td>%s</td><td>%s</td></tr>`,
			id, id, template.HTMLEscapeString(content.HumanStatus(seen[id])), template.HTMLEscapeString(seen[id]), statusPhase(id), term)
	}
	rows.WriteString(`</tbody></table></div>`)
	return rows.String()
}

// appendStatusTable re-renders business/statuses.html with a generated table after the hand-written body.
func (s *site) appendStatusTable() error {
	var page *content.Page
	var siblings []content.Page
	for i := range s.pages {
		if s.pages[i].Path == statusesPath {
			page = &s.pages[i]
		}
	}
	if page == nil {
		return fmt.Errorf("content/%s.md is required", statusesPath)
	}
	for _, p := range s.pages {
		if p.Section == page.Section {
			siblings = append(siblings, p)
		}
	}
	out := statusesPath + ".html"
	body, err := content.Render(page.Body, s.resolver(out))
	if err != nil {
		return err
	}
	v := s.view(out, page.Title)
	v.Section, v.Level, v.Summary = page.Section, page.Level, page.Summary
	v.Body = template.HTML(body + s.statusRows(out))
	prevNext(&v, *page, siblings)
	b, err := renderTemplate("article.html", v)
	if err != nil {
		return err
	}
	return s.write(out, b)
}
