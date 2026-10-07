// Package config loads handbook.json.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type App struct {
	ID           string   `json:"id"`
	Path         string   `json:"path"`
	Label        string   `json:"label"`
	Audience     string   `json:"audience"`
	NestedAPI    []string `json:"nestedAPI"`
	SkipSegments []string `json:"skipSegments"`
}

type Config struct {
	SiteTitle string `json:"siteTitle"`
	Apps      []App  `json:"apps"`
	Dir       string `json:"-"` // directory holding handbook.json
}

// Load reads the config. appsRoot, when set, replaces the parent directory of every app path.
func Load(path, appsRoot string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var c Config
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	c.Dir = filepath.Dir(abs)
	seen := map[string]bool{}
	for i := range c.Apps {
		a := &c.Apps[i]
		if a.ID == "" {
			return nil, fmt.Errorf("app %d in %s has no id", i+1, path)
		}
		if seen[a.ID] {
			return nil, fmt.Errorf("duplicate app id %q", a.ID)
		}
		seen[a.ID] = true
		if a.Label == "" {
			a.Label = a.ID
		}
		if appsRoot != "" {
			a.Path = filepath.Join(appsRoot, filepath.Base(a.Path))
		} else if !filepath.IsAbs(a.Path) {
			a.Path = filepath.Clean(filepath.Join(c.Dir, a.Path))
		}
	}
	if c.SiteTitle == "" {
		c.SiteTitle = "Carrybee Handbook"
	}
	return &c, nil
}
