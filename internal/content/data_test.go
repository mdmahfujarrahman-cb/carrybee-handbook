package content

import "testing"

func TestLoadDataFromRepo(t *testing.T) {
	d, err := LoadData("../../data")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Terms) != 185 || len(d.Categories) != 12 {
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
