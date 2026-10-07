package main

import (
	"reflect"
	"testing"

	"carrybee/handbook/internal/site"
)

func TestCheckProblemsRejectsSnapshotUnlessAllowed(t *testing.T) {
	res := &site.Result{FromSnapshot: []string{"hive"}, Warnings: []string{"w"}, Missing: []string{"m"}}
	want := []string{"w", "m", "hive: not scanned live (loaded from snapshot/scan.json; pass -allow-snapshot to accept)"}
	if got := checkProblems(res, false); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
	if got := checkProblems(res, true); !reflect.DeepEqual(got, []string{"w", "m"}) {
		t.Fatalf("allowed: got %q", got)
	}
}

func TestParseAllowSnapshot(t *testing.T) {
	cmd, o, err := parse([]string{"check", "-allow-snapshot"})
	if err != nil || cmd != "check" || !o.allowSnapshot {
		t.Fatalf("cmd %q opts %+v err %v", cmd, o, err)
	}
}
