package adapter

import (
	"testing"
)

func TestDemoDataSource_ListPortfolios(t *testing.T) {
	ds := NewDemoDataSource()
	data, err := ds.ListPortfolios("time")
	if err != nil {
		t.Fatalf("ListPortfolios: %v", err)
	}

	if len(data.Portfolios) != 2 {
		t.Fatalf("expected 2 portfolios, got %d", len(data.Portfolios))
	}

	if data.Stats.TotalPortfolios != 2 {
		t.Errorf("expected TotalPortfolios=2, got %d", data.Stats.TotalPortfolios)
	}

	if data.SortMode != "time" {
		t.Errorf("expected SortMode='time', got %q", data.SortMode)
	}
}

func TestDemoDataSource_GetPortfolio(t *testing.T) {
	ds := NewDemoDataSource()

	data, err := ds.GetPortfolio("infrastructure", "name")
	if err != nil {
		t.Fatalf("GetPortfolio: %v", err)
	}

	if data.Name != "infrastructure" {
		t.Errorf("expected Name='infrastructure', got %q", data.Name)
	}

	if len(data.Workspaces) != 2 {
		t.Errorf("expected 2 workspaces, got %d", len(data.Workspaces))
	}
}

func TestDemoDataSource_GetPortfolio_NotFound(t *testing.T) {
	ds := NewDemoDataSource()

	data, err := ds.GetPortfolio("nonexistent", "time")
	if err != nil {
		t.Fatalf("GetPortfolio: %v", err)
	}

	if data.Name != "" {
		t.Errorf("expected empty Name for nonexistent portfolio, got %q", data.Name)
	}
}

func TestDemoDataSource_GetWorkspace(t *testing.T) {
	ds := NewDemoDataSource()

	data, err := ds.GetWorkspace("infrastructure", "platform", "time")
	if err != nil {
		t.Fatalf("GetWorkspace: %v", err)
	}

	if data.Name != "platform" {
		t.Errorf("expected Name='platform', got %q", data.Name)
	}

	if len(data.Repos) != 3 {
		t.Errorf("expected 3 repos, got %d", len(data.Repos))
	}
}

func TestDemoDataSource_GetWorkspace_NotFound(t *testing.T) {
	ds := NewDemoDataSource()

	data, err := ds.GetWorkspace("infrastructure", "nonexistent", "time")
	if err != nil {
		t.Fatalf("GetWorkspace: %v", err)
	}

	if data.Name != "" {
		t.Errorf("expected empty Name for nonexistent workspace, got %q", data.Name)
	}
}

func TestDemoDataSource_GetForge(t *testing.T) {
	ds := NewDemoDataSource()

	data, err := ds.GetForge("infrastructure", "platform", "forge")
	if err != nil {
		t.Fatalf("GetForge: %v", err)
	}

	if data.RepoName != "forge" {
		t.Errorf("expected RepoName='forge', got %q", data.RepoName)
	}

	if data.Spec.Name != "forge" {
		t.Errorf("expected Spec.Name='forge', got %q", data.Spec.Name)
	}

	if len(data.TestReports) != 3 {
		t.Errorf("expected 3 test reports, got %d", len(data.TestReports))
	}
}

func TestDemoDataSource_GetForge_NotFound(t *testing.T) {
	ds := NewDemoDataSource()

	data, err := ds.GetForge("infrastructure", "platform", "nonexistent")
	if err != nil {
		t.Fatalf("GetForge: %v", err)
	}

	if data.RepoName != "" {
		t.Errorf("expected empty RepoName for nonexistent repo, got %q", data.RepoName)
	}
}
