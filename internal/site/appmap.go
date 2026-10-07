package site

import (
	"sort"

	"carrybee/handbook/internal/content"
)

func (s *site) writeApps() error {
	for _, app := range s.cfg.Apps {
		sc := s.scans[app.ID]
		termLinks := map[string][][2]string{}
		for _, t := range s.terms {
			for _, m := range t.Where[app.ID] {
				termLinks[m] = append(termLinks[m], [2]string{t.ID, t.Name})
			}
		}
		for m := range termLinks {
			sort.Slice(termLinks[m], func(i, j int) bool { return termLinks[m][i][1] < termLinks[m][j][1] })
		}
		story := s.data.Stories[app.ID]
		if story == nil {
			story = []content.StoryPhase{}
		}
		payload := map[string]any{
			"id": app.ID, "label": app.Label, "audience": app.Audience, "commit": sc.Commit, "error": sc.Error,
			"mods": sc.Modules, "routes": sc.Routes, "cycles": sc.Cycles,
			"desc": s.data.Modules[app.ID], "groups": s.data.Groups[app.ID],
			"story": story, "storySub": s.data.StorySub[app.ID], "termLinks": termLinks,
		}
		b, err := jsData("app", payload)
		if err != nil {
			return err
		}
		if err := s.write("data/"+app.ID+".js", b); err != nil {
			return err
		}
		out := appOut(app.ID)
		v := s.view(out, app.Label)
		v.Section, v.Wide = "apps", true
		v.Extra = map[string]any{"App": app, "Scan": sc, "Data": Rel(out, "data/"+app.ID+".js")}
		page, err := renderTemplate("app.html", v)
		if err != nil {
			return err
		}
		if err := s.write(out, page); err != nil {
			return err
		}
	}
	return nil
}
