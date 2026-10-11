package content

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDataFromRepo(t *testing.T) {
	d, err := LoadData("../../data")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Terms) != 186|| len(d.Categories) != 12 {
		t.Fatalf("terms=%d cats=%d", len(d.Terms), len(d.Categories))
	}
	if d.Modules["hive"]["delivery-run"].Title != "Delivery run" {
		t.Fatalf("desc = %+v", d.Modules["hive"]["delivery-run"])
	}
	if len(d.Stories["hive"]) != 14 || d.StorySub["aladdin"] == "" {
		t.Fatalf("stories hive=%d", len(d.Stories["hive"]))
	}
	for _, term := range d.Terms {
		if term.ID == "" {
			t.Fatalf("term %q has no id", term.Name)
		}
	}
	if d.Pins["G-Mode"]["hive"][0] != "g-mode" {
		t.Fatalf("pins = %v", d.Pins["G-Mode"])
	}
}

func TestLoadDataRejectsDuplicateTermIDs(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"modules.json": `{}`, "groups.json": `{}`, "term-pins.json": `{}`,
		"stories.json": `{"intro":{},"apps":{}}`,
		// "Hub ledger" slugs to hub-ledger, the same id as the explicit one.
		"terms.json": `{"categories":[],"terms":[{"id":"hub-ledger","t":"Ledger","def":"x"},{"t":"Hub ledger","def":"y"}]}`,
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	_, err := LoadData(dir)
	if err == nil || !strings.Contains(err.Error(), `duplicate term id "hub-ledger"`) {
		t.Fatalf("err = %v, want duplicate term id error", err)
	}
}
