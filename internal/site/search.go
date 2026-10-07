package site

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"carrybee/handbook/internal/content"
	"carrybee/handbook/internal/scan"
)

// SearchEntry is one row of the ⌘K search index.
type SearchEntry struct {
	Type  string `json:"type"` // page | term | module | route | status | api
	Title string `json:"title"`
	Sub   string `json:"sub"`  // short context line
	Href  string `json:"href"` // root-relative output path (JS prefixes with HB.root)
	Text  string `json:"text"` // extra words to match
}

// pageOut maps a content page to its output path.
func pageOut(p content.Page) string {
	if p.Path == "home" {
		return "index.html"
	}
	return p.Path + ".html"
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func appEntries(a string, s *scan.AppScan, data *content.Data) []SearchEntry {
	var out []SearchEntry
	for _, m := range sortedKeys(s.Modules) {
		d := data.Modules[a][m]
		out = append(out, SearchEntry{"module", m, a + " · " + d.Title, "apps/" + a + ".html#mod." + m, d.Text})
	}
	for _, r := range sortedKeys(s.Routes) {
		out = append(out, SearchEntry{"route", r, a + " page", "apps/" + a + ".html", ""})
	}
	for _, d := range sortedKeys(s.APIDomains) {
		mods := s.APIDomains[d]
		out = append(out, SearchEntry{"api", d, a + " · used by " + strconv.Itoa(len(mods)) + " modules",
			"code/api-index.html#api." + a + "." + d, strings.Join(mods, " ")})
	}
	return out
}

func searchIndex(pages []content.Page, terms []content.Term, data *content.Data, scans map[string]*scan.AppScan) []SearchEntry {
	var out []SearchEntry
	for _, p := range pages {
		out = append(out, SearchEntry{"page", p.Title, p.Summary, pageOut(p), p.Section})
	}
	for _, t := range terms {
		out = append(out, SearchEntry{"term", t.Name, t.Def, "terms.html#t." + t.ID, strings.Join(t.Aka, " ")})
	}
	seenStatus := map[int]bool{}
	for _, a := range sortedKeys(scans) {
		s := scans[a]
		out = append(out, appEntries(a, s, data)...)
		for _, st := range s.Statuses {
			if seenStatus[st.ID] {
				continue
			}
			seenStatus[st.ID] = true
			id := strconv.Itoa(st.ID)
			out = append(out, SearchEntry{"status", id + " " + st.Name, "order status", "business/statuses.html#s." + id, ""})
		}
	}
	return out
}

func jsData(key string, v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return []byte("window.HB = window.HB || {}; HB." + key + " = " + string(b) + ";\n"), nil
}
