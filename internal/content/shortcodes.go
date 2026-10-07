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
	gmhtml "github.com/yuin/goldmark/renderer/html"
)

type Resolver struct {
	TermName   func(id string) (string, bool)
	ModuleDesc func(app, id string) (Desc, bool)
	StatusName func(id int) (string, bool)
	ModuleHref func(app, id string) string
	TermHref   func(id string) string
}

var md = goldmark.New(goldmark.WithExtensions(extension.GFM), goldmark.WithRendererOptions(gmhtml.WithUnsafe()))

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
