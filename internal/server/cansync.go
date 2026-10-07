package server

import (
	"strings"

	"carrybee/handbook/internal/config"
	"carrybee/handbook/internal/site"
)

// askForCopy ends the reason shown when Sync is off.
const askForCopy = "Ask the frontend team for a new copy of the handbook."

// syncReason explains that Sync needs the code of every configured app.
func syncReason(apps []config.App) string {
	ids := make([]string, len(apps))
	for i, a := range apps {
		ids[i] = a.ID
	}
	return "Sync needs the " + joinAnd(ids) + " code next to this folder. " + askForCopy
}

// joinAnd joins items as "a, b and c".
func joinAnd(items []string) string {
	if len(items) < 2 {
		return strings.Join(items, "")
	}
	last := len(items) - 1
	return strings.Join(items[:last], ", ") + " and " + items[last]
}

// canSync reports whether every app's code folder is present, and the reason when it is not.
func (s *Server) canSync() (bool, string) {
	for _, a := range s.cfg.Apps {
		if !site.AppPresent(a) {
			return false, syncReason(s.cfg.Apps)
		}
	}
	return true, ""
}
