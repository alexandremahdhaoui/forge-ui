package controller

import (
	"strings"
	"testing"

	"github.com/alexandremahdhaoui/forge-ui/internal/adapter"
)

func newTestRenderer() PageRenderer {
	ds := adapter.NewDemoDataSource()
	return NewPageRenderer(ds)
}

func TestRender_Portfolios(t *testing.T) {
	r := newTestRenderer()
	html, err := r.Render("/portfolios?sort=time")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	checks := []string{
		"Portfolios",
		"infrastructure",
		"default",
		"catch-all",
		"platform",
		"/home/user/workspaces/infrastructure/platform",
		"cell-passed",
		"segmented-btn--active",
	}

	for _, check := range checks {
		if !strings.Contains(html, check) {
			t.Errorf("expected HTML to contain %q", check)
		}
	}
}

func TestRender_Portfolio(t *testing.T) {
	r := newTestRenderer()
	html, err := r.Render("/portfolios/infrastructure?sort=name")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	checks := []string{
		"infrastructure",
		"Portfolios",
		"platform",
		"cell-passed",
	}

	for _, check := range checks {
		if !strings.Contains(html, check) {
			t.Errorf("expected HTML to contain %q", check)
		}
	}
}

func TestRender_Workspace(t *testing.T) {
	r := newTestRenderer()
	html, err := r.Render("/portfolios/infrastructure/workspaces/platform?sort=time")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	checks := []string{
		"platform",
		"infrastructure",
		"forge",
		"2 changed",
		"cmd/forge-ui-wasm/main.go",
		"cell-passed",
		"cell-failed",
		"a1b2c3d",
		"feat: add generic-builder engine",
		"card-elevated",
	}

	for _, check := range checks {
		if !strings.Contains(html, check) {
			t.Errorf("expected HTML to contain %q", check)
		}
	}
}

func TestRender_Forge(t *testing.T) {
	r := newTestRenderer()
	html, err := r.Render("/portfolios/infrastructure/workspaces/platform/repos/forge")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	checks := []string{
		"forge",
		"infrastructure",
		"platform",
		"go://go-build",
		"go://go-lint",
		"go://go-test",
		"binary",
		"a1b2c3d",
		"cell-passed",
		"78.4%",
		"Build Targets",
		"Test Reports",
		"Artifacts",
	}

	for _, check := range checks {
		if !strings.Contains(html, check) {
			t.Errorf("expected HTML to contain %q", check)
		}
	}
}

func TestRender_UnknownPortfolio_FallsBackToPortfolios(t *testing.T) {
	r := newTestRenderer()
	html, err := r.Render("/portfolios/nonexistent")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	if !strings.Contains(html, "Portfolios") {
		t.Error("expected fallback to portfolios page")
	}
}

func TestRender_DefaultRoute(t *testing.T) {
	r := newTestRenderer()
	html, err := r.Render("")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	if !strings.Contains(html, "Portfolios") {
		t.Error("expected default route to render portfolios")
	}
}

func TestRender_SortButtons(t *testing.T) {
	r := newTestRenderer()
	html, err := r.Render("/portfolios?sort=time")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	if !strings.Contains(html, "segmented-btn--active\">Recent") {
		t.Error("expected Recent button to be active by default")
	}
}

func TestRender_HashRoute(t *testing.T) {
	r := newTestRenderer()
	html, err := r.Render("#/portfolios/infrastructure")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	if !strings.Contains(html, "infrastructure") {
		t.Error("expected hash route to resolve correctly")
	}
}

func TestParseInput_Defaults(t *testing.T) {
	in := parseInput("")
	if in.Route != "/" {
		t.Errorf("expected route '/', got %q", in.Route)
	}
	if in.Sort != "time" {
		t.Errorf("expected sort 'time', got %q", in.Sort)
	}
	if in.Theme != "light" {
		t.Errorf("expected theme 'light', got %q", in.Theme)
	}
}

func TestParseInput_WithParams(t *testing.T) {
	in := parseInput("/portfolios/infra?sort=name&theme=dark")
	if in.Route != "/portfolios/infra" {
		t.Errorf("expected route '/portfolios/infra', got %q", in.Route)
	}
	if in.Sort != "name" {
		t.Errorf("expected sort 'name', got %q", in.Sort)
	}
	if in.Theme != "dark" {
		t.Errorf("expected theme 'dark', got %q", in.Theme)
	}
}

func TestParseInput_HashPrefix(t *testing.T) {
	in := parseInput("#/portfolios")
	if in.Route != "/portfolios" {
		t.Errorf("expected route '/portfolios', got %q", in.Route)
	}
}
