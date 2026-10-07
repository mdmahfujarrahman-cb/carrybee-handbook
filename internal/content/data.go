// Package content loads hand-written Markdown and curated JSON data.
package content

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Desc struct{ Title, Text string }

func (d *Desc) UnmarshalJSON(b []byte) error {
	var pair []string
	if err := json.Unmarshal(b, &pair); err != nil || len(pair) != 2 {
		return fmt.Errorf("description must be [title, text]: %s", b)
	}
	d.Title, d.Text = pair[0], pair[1]
	return nil
}

func (d Desc) MarshalJSON() ([]byte, error) { return json.Marshal([]string{d.Title, d.Text}) }

type Category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Term struct {
	ID       string              `json:"id"`
	Name     string              `json:"t"`
	Cat      string              `json:"cat"`
	Def      string              `json:"def"`
	Aka      []string            `json:"aka"`
	Apps     []string            `json:"apps"`
	Inferred bool                `json:"inferred,omitempty"`
	Where    map[string][]string `json:"where,omitempty"`
}

type StoryStep struct {
	Title  string   `json:"title"`
	Actor  string   `json:"actor"`
	Text   string   `json:"text"`
	Status string   `json:"status,omitempty"`
	Mods   []string `json:"mods"`
}

type StoryPhase struct {
	Title string      `json:"title"`
	When  string      `json:"when"`
	Steps []StoryStep `json:"steps"`
}

type Data struct {
	Modules    map[string]map[string]Desc
	Groups     map[string]map[string]Desc
	Stories    map[string][]StoryPhase
	StorySub   map[string]string
	Categories []Category
	Terms      []Term
	Pins       map[string]map[string][]string
}

var slugRE = regexp.MustCompile(`[^a-z0-9]+`)

// Slug turns a display name into an id usable in URLs and anchors.
func Slug(s string) string {
	return strings.Trim(slugRE.ReplaceAllString(strings.ToLower(s), "-"), "-")
}

func readJSON(dir, name string, v any) error {
	raw, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, v); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func LoadData(dir string) (*Data, error) {
	d := &Data{}
	var stories struct {
		Intro map[string]string       `json:"intro"`
		Apps  map[string][]StoryPhase `json:"apps"`
	}
	var terms struct {
		Categories []Category `json:"categories"`
		Terms      []Term     `json:"terms"`
	}
	for _, f := range []struct {
		name string
		v    any
	}{{"modules.json", &d.Modules}, {"groups.json", &d.Groups}, {"stories.json", &stories},
		{"terms.json", &terms}, {"term-pins.json", &d.Pins}} {
		if err := readJSON(dir, f.name, f.v); err != nil {
			return nil, err
		}
	}
	d.Stories, d.StorySub = stories.Apps, stories.Intro
	d.Categories, d.Terms = terms.Categories, terms.Terms
	for i := range d.Terms {
		if d.Terms[i].ID == "" {
			d.Terms[i].ID = Slug(d.Terms[i].Name)
		}
	}
	return d, nil
}
