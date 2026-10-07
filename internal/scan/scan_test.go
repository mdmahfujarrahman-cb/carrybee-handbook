package scan

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"carrybee/handbook/internal/config"
)

func TestRouteNamerHive(t *testing.T) {
	name := RouteNamer(config.App{ID: "hive"})
	cases := map[string]string{
		"(main)/(protected)/(dashboard_layout)/(oms)/orders/page.tsx":        "(oms)/orders",
		"(main)/(protected)/(dashboard_layout)/run-not-closed/page.tsx":      "run-not-closed",
		"(main)/(protected)/(dashboard_layout)/dashboard/@payments/page.tsx": "dashboard",
		"page.tsx": "/",
	}
	for in, want := range cases {
		if got := name(split(in)); got != want {
			t.Errorf("%s: got %q want %q", in, got, want)
		}
	}
}

func TestRouteNamerAladdinTwoLevels(t *testing.T) {
	name := RouteNamer(config.App{ID: "aladdin", SkipSegments: []string{"(business)", "(merchant)"}})
	cases := map[string]string{
		"(protected)/(business)/businesses/[businessId]/orders/(create-forms)/create/page.tsx": "orders/create",
		"(protected)/(business)/businesses/(global_layout)/choose/page.tsx":                    "businesses/choose",
		"(protected)/(business)/businesses/[businessId]/stores/(list)/page.tsx":                "stores",
	}
	for in, want := range cases {
		if got := name(split(in)); got != want {
			t.Errorf("%s: got %q want %q", in, got, want)
		}
	}
}

func TestAPIDomains(t *testing.T) {
	text := "API_ROUTES.orders.list; API_ROUTES.business.stores.create; API_ROUTES.auth"
	got := apiDomains(text, []string{"business"})
	want := map[string]int{"orders": 1, "business.stores": 1, "auth": 1}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
}

func TestStatusesIgnoreCommentedMembers(t *testing.T) {
	src := StripComments("export enum TransferStatus {\n Pending = 1,\n // Old = 38,\n Delivered = 15,\n}\nexport enum Other { X = 9 }")
	got := parseStatuses(src)
	want := []Status{{1, "Pending"}, {15, "Delivered"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
}

func TestCycles(t *testing.T) {
	deps := map[string]map[string]int{
		"a": {"b": 1}, "b": {"c": 1}, "c": {"a": 1},
		"x": {"y": 1}, "y": {"x": 1}, "z": {"a": 1},
	}
	got := Cycles(deps)
	want := [][]string{{"a", "b", "c"}, {"x", "y"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
}

func TestScanAppFixture(t *testing.T) {
	s := ScanApp(fixtureApp(t))
	if s.Error != "" {
		t.Fatal(s.Error)
	}
	if len(s.Modules) != 3 || s.Routes["(oms)/orders"] == nil || s.Routes["run-not-closed"] == nil {
		t.Fatalf("modules=%d routes=%v", len(s.Modules), s.Routes)
	}
	if s.Modules["orders"].UsedByRoutes["(oms)/orders"] != 1 {
		t.Fatalf("usedByRoutes = %v", s.Modules["orders"].UsedByRoutes)
	}
	if got := s.APIDomains["orders"]; !reflect.DeepEqual(got, []string{"orders"}) {
		t.Fatalf("api domains = %v", s.APIDomains)
	}
	if len(s.Statuses) != 3 {
		t.Fatalf("statuses = %v", s.Statuses)
	}
	if !reflect.DeepEqual(s.Cycles, [][]string{{"basket", "orders"}}) {
		t.Fatalf("cycles = %v", s.Cycles)
	}
}

func TestScanAllMissingApp(t *testing.T) {
	cfg := &config.Config{Apps: []config.App{fixtureApp(t), {ID: "ghost", Path: filepath.Join(os.TempDir(), "no-such-app")}}}
	out := ScanAll(cfg)
	if out["fixture"].Error != "" {
		t.Fatalf("fixture failed: %s", out["fixture"].Error)
	}
	if out["ghost"] == nil || out["ghost"].Error == "" {
		t.Fatal("missing app must be returned with Error set")
	}
}

// Parity with the prototype on the real repos; skipped when they are not next to the handbook.
func TestParityWithPrototype(t *testing.T) {
	want := map[string][3]int{"hive": {76, 82, 4}, "aladdin": {18, 54, 1}, "accounts": {10, 10, 0}}
	cfg, err := config.Load("../../handbook.json", "")
	if err != nil {
		t.Skip("no handbook.json")
	}
	for _, app := range cfg.Apps {
		if _, err := os.Stat(app.Path); err != nil {
			t.Skipf("%s not present", app.ID)
		}
		s := ScanApp(app)
		got := [3]int{len(s.Modules), len(pageRoutes(s.Routes)), len(s.Cycles)}
		if got != want[app.ID] {
			t.Errorf("%s: modules/pages/cycles = %v, want %v", app.ID, got, want[app.ID])
		}
	}
}
