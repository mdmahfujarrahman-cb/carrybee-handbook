package site

import "html/template"

// Placeholders: Task 8 replaces writeTerms and writeGenerated.

func (s *site) placeholder(out, title string) error {
	v := s.view(out, title)
	v.Body = template.HTML("<p>Generated in a later task.</p>")
	b, err := renderTemplate("article.html", v)
	if err != nil {
		return err
	}
	return s.write(out, b)
}

func (s *site) writeTerms() error { return s.placeholder("terms.html", "Courier terms") }

func (s *site) writeGenerated() error {
	return s.placeholder("code/api-index.html", "API domain index")
}
