//go:build !js || !wasm

package adapter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWsConfig_Load_ValidFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	content := []byte(`name: my-workspace
description: Build a distributed cache
repos:
  - name: cache-core
    description: Core caching library
  - name: cache-proxy
    description: Proxy layer
metaPlans:
  - cache-migration
  - perf-tuning
`)
	if err := os.WriteFile(filepath.Join(dir, "forge-workspace.yaml"), content, 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewWsConfigLoader()
	cfg, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Name != "my-workspace" {
		t.Errorf("Name = %q, want %q", cfg.Name, "my-workspace")
	}
	if cfg.Description != "Build a distributed cache" {
		t.Errorf("Description = %q, want %q", cfg.Description, "Build a distributed cache")
	}
	if len(cfg.Repos) != 2 {
		t.Fatalf("len(Repos) = %d, want 2", len(cfg.Repos))
	}
	if cfg.Repos[0].Name != "cache-core" {
		t.Errorf("Repos[0].Name = %q, want %q", cfg.Repos[0].Name, "cache-core")
	}
	if cfg.Repos[0].Description != "Core caching library" {
		t.Errorf("Repos[0].Description = %q, want %q", cfg.Repos[0].Description, "Core caching library")
	}
	if cfg.Repos[1].Name != "cache-proxy" {
		t.Errorf("Repos[1].Name = %q, want %q", cfg.Repos[1].Name, "cache-proxy")
	}
	if len(cfg.MetaPlans) != 2 {
		t.Fatalf("len(MetaPlans) = %d, want 2", len(cfg.MetaPlans))
	}
	if cfg.MetaPlans[0] != "cache-migration" {
		t.Errorf("MetaPlans[0] = %q, want %q", cfg.MetaPlans[0], "cache-migration")
	}
}

func TestWsConfig_Load_MissingFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	loader := NewWsConfigLoader()
	cfg, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Name != "" {
		t.Errorf("Name = %q, want empty", cfg.Name)
	}
	if len(cfg.Repos) != 0 {
		t.Errorf("len(Repos) = %d, want 0", len(cfg.Repos))
	}
}

func TestWsConfig_Load_MalformedYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	content := []byte(`name: [invalid yaml
  description: broken
`)
	if err := os.WriteFile(filepath.Join(dir, "forge-workspace.yaml"), content, 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewWsConfigLoader()
	_, err := loader.Load(dir)
	if err == nil {
		t.Fatal("Load() expected error for malformed YAML, got nil")
	}
}

func TestWsConfig_Load_EmptyFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "forge-workspace.yaml"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewWsConfigLoader()
	cfg, err := loader.Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Name != "" {
		t.Errorf("Name = %q, want empty", cfg.Name)
	}
	if len(cfg.Repos) != 0 {
		t.Errorf("len(Repos) = %d, want 0", len(cfg.Repos))
	}
}
