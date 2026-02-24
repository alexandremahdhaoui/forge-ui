//go:build !js || !wasm

package adapter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPortfolioConfig_Load_ValidFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	content := []byte(`name: my-portfolio
description: Ship the next-generation platform
trackerPaths:
  - .forge-ai/plan
  - .forge-ai/tracker
`)
	if err := os.WriteFile(filepath.Join(dir, "forge-portfolio.yaml"), content, 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewPortfolioConfigLoader()
	cfg, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Name != "my-portfolio" {
		t.Errorf("Name = %q, want %q", cfg.Name, "my-portfolio")
	}
	if cfg.Description != "Ship the next-generation platform" {
		t.Errorf("Description = %q, want %q", cfg.Description, "Ship the next-generation platform")
	}
	if len(cfg.TrackerPaths) != 2 {
		t.Fatalf("len(TrackerPaths) = %d, want 2", len(cfg.TrackerPaths))
	}
	if cfg.TrackerPaths[0] != ".forge-ai/plan" {
		t.Errorf("TrackerPaths[0] = %q, want %q", cfg.TrackerPaths[0], ".forge-ai/plan")
	}
	if cfg.TrackerPaths[1] != ".forge-ai/tracker" {
		t.Errorf("TrackerPaths[1] = %q, want %q", cfg.TrackerPaths[1], ".forge-ai/tracker")
	}
}

func TestPortfolioConfig_Load_MissingFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	loader := NewPortfolioConfigLoader()
	cfg, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Name != "" {
		t.Errorf("Name = %q, want empty", cfg.Name)
	}
	if cfg.Description != "" {
		t.Errorf("Description = %q, want empty", cfg.Description)
	}
	if len(cfg.TrackerPaths) != 0 {
		t.Errorf("len(TrackerPaths) = %d, want 0", len(cfg.TrackerPaths))
	}
}

func TestPortfolioConfig_Load_MalformedYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	content := []byte(`name: [invalid yaml
  description: broken
`)
	if err := os.WriteFile(filepath.Join(dir, "forge-portfolio.yaml"), content, 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewPortfolioConfigLoader()
	_, err := loader.Load(dir)
	if err == nil {
		t.Fatal("Load() expected error for malformed YAML, got nil")
	}
}

func TestPortfolioConfig_Load_EmptyFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "forge-portfolio.yaml"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewPortfolioConfigLoader()
	cfg, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Name != "" {
		t.Errorf("Name = %q, want empty", cfg.Name)
	}
	if cfg.Description != "" {
		t.Errorf("Description = %q, want empty", cfg.Description)
	}
	if len(cfg.TrackerPaths) != 0 {
		t.Errorf("len(TrackerPaths) = %d, want 0", len(cfg.TrackerPaths))
	}
}
