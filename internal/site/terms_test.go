package site

import (
	"reflect"
	"testing"

	"carrybee/handbook/internal/content"
	"carrybee/handbook/internal/scan"
)

func TestLinkTerms(t *testing.T) {
	scans := map[string]*scan.AppScan{"hive": {Files: map[string]string{
		"modules/hub-payments/a.ts": "const cashInHand = 1; cashInHand++",
		"modules/hub-ledger/b.ts":   "type X = { cash_in_hand: number }",
		"modules/geonix/c.ts":       "const cashInHandle = 2",
	}, Modules: map[string]*scan.Module{"hub-payments": {}, "hub-ledger": {}, "geonix": {}}}}
	terms := []content.Term{
		{Name: "Cash in hand", Aka: []string{"cash_in_hand", "cashInHand"}, Apps: []string{"hive"}},
		{Name: "Pinned", Aka: []string{"zzzz"}, Apps: []string{"hive"}},
	}
	pins := map[string]map[string][]string{"Pinned": {"hive": {"geonix"}}}
	got := LinkTerms(terms, scans, pins)
	if !reflect.DeepEqual(got[0].Where["hive"], []string{"hub-payments", "hub-ledger"}) {
		t.Fatalf("where = %v (word boundaries must reject cashInHandle)", got[0].Where)
	}
	if !reflect.DeepEqual(got[1].Where["hive"], []string{"geonix"}) {
		t.Fatalf("pin not applied: %v", got[1].Where)
	}
}
