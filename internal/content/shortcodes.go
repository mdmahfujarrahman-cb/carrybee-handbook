package content

import (
	"bytes"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"
)

type Resolver struct {
	TermName   func(id string) (string, bool)
	ModuleDesc func(app, id string) (Desc, bool)
	StatusName func(id int) (string, bool)
	ModuleHref func(app, id string) string
	// Diagram returns the inline SVG markup of a named diagram.
	Diagram func(name string) (string, bool)
}

// md renders GFM with raw HTML allowed and an id on every heading, so links such as
// how-carrybee-works.html#money-flow resolve without JavaScript.
var md = goldmark.New(goldmark.WithExtensions(extension.GFM),
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	goldmark.WithRendererOptions(gmhtml.WithUnsafe()))

var calloutRE = regexp.MustCompile(`(?ms)^:::(note|carrybee|warning)\s*\n(.*?)\n:::\s*$`)

var goodStatus = map[int]bool{15: true, 18: true, 22: true}
var badStatus = map[int]bool{17: true, 19: true, 20: true, 32: true}

func statusClass(id int) string {
	switch {
	case goodStatus[id]:
		return "ok"
	case badStatus[id]:
		return "bad"
	}
	return "go"
}

func expandTerm(args []string, r Resolver) string {
	name, ok := r.TermName(args[0])
	if !ok {
		name = args[0]
	}
	if len(args) > 1 && args[1] != "" {
		name = args[1]
	}
	return fmt.Sprintf(`<span class="term" data-term="%s">%s</span>`, html.EscapeString(args[0]), html.EscapeString(name))
}

func expandModule(args []string, r Resolver) string {
	app, id := args[0], args[1]
	title := ""
	if d, ok := r.ModuleDesc(app, id); ok {
		title = fmt.Sprintf(` title="%s"`, html.EscapeString(d.Title+": "+d.Text))
	}
	return fmt.Sprintf(`<a class="where %s" href="%s"%s><b>%s</b><span>%s</span></a>`,
		html.EscapeString(app), html.EscapeString(r.ModuleHref(app, id)), title, html.EscapeString(app), html.EscapeString(id))
}

func expandStatus(args []string, r Resolver) string {
	id, _ := strconv.Atoi(args[0])
	name, _ := r.StatusName(id)
	return fmt.Sprintf(`<span class="pill st %s">%d %s</span>`, statusClass(id), id, html.EscapeString(name))
}

// expandDiagram embeds an inline SVG as one HTML block: blank lines would end the block in Markdown.
func expandDiagram(args []string, r Resolver) string {
	svg, ok := r.Diagram(args[0])
	if !ok {
		return fmt.Sprintf(`<p class="callout warning">Missing diagram %s</p>`, html.EscapeString(args[0]))
	}
	var lines []string
	for _, l := range strings.Split(strings.TrimSpace(svg), "\n") {
		if strings.TrimSpace(l) != "" {
			lines = append(lines, l)
		}
	}
	return `<figure class="diagram">` + strings.Join(lines, "\n") + `</figure>`
}

func expandOne(m string, r Resolver) string {
	parts := shortcodeRE.FindStringSubmatch(m)
	args := shortcodeArgs(parts[2])
	switch {
	case parts[1] == kindTerm && len(args) >= 1:
		return expandTerm(args, r)
	case parts[1] == kindModule && len(args) == 2:
		return expandModule(args, r)
	case parts[1] == kindStatus && len(args) >= 1:
		return expandStatus(args, r)
	case parts[1] == kindDiagram && len(args) >= 1:
		return expandDiagram(args, r)
	}
	return m
}

func expand(src string, r Resolver) string {
	return shortcodeRE.ReplaceAllStringFunc(src, func(m string) string { return expandOne(m, r) })
}

func renderCallouts(src string) (string, error) {
	var renderErr error
	out := calloutRE.ReplaceAllStringFunc(src, func(m string) string {
		parts := calloutRE.FindStringSubmatch(m)
		var inner bytes.Buffer
		if err := md.Convert([]byte(parts[2]), &inner); err != nil {
			renderErr = err
		}
		return fmt.Sprintf("<aside class=\"callout %s\">\n%s\n</aside>\n", parts[1], strings.TrimSpace(inner.String()))
	})
	return out, renderErr
}

// Render turns handbook Markdown into HTML.
func Render(src string, r Resolver) (string, error) {
	src, err := renderCallouts(expand(src, r))
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	if err := md.Convert([]byte(src), &out); err != nil {
		return "", err
	}
	return out.String(), nil
}
