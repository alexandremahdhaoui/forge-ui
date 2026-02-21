package render

import (
	"strings"
	"testing"
)

func TestExecute_Portfolios(t *testing.T) {
	html, err := Execute("/portfolios?sort=time")
	if err != nil {
		t.Fatalf("Execute: %v", err)
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

func TestExecute_Portfolio(t *testing.T) {
	html, err := Execute("/portfolios/infrastructure?sort=name")
	if err != nil {
		t.Fatalf("Execute: %v", err)
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

func TestExecute_Workspace(t *testing.T) {
	html, err := Execute("/portfolios/infrastructure/workspaces/platform?sort=time")
	if err != nil {
		t.Fatalf("Execute: %v", err)
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

func TestExecute_Forge(t *testing.T) {
	html, err := Execute("/portfolios/infrastructure/workspaces/platform/repos/forge")
	if err != nil {
		t.Fatalf("Execute: %v", err)
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

func TestExecute_UnknownPortfolio_FallsBackToPortfolios(t *testing.T) {
	html, err := Execute("/portfolios/nonexistent")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// Should fall back to portfolios list
	if !strings.Contains(html, "Portfolios") {
		t.Error("expected fallback to portfolios page")
	}
}

func TestExecute_DefaultRoute(t *testing.T) {
	html, err := Execute("")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if !strings.Contains(html, "Portfolios") {
		t.Error("expected default route to render portfolios")
	}
}

func TestExecute_EmptyPortfolios_SortButtons(t *testing.T) {
	html, err := Execute("/portfolios?sort=time")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// Default sort should be "time" and the Recent button should be active
	if !strings.Contains(html, "segmented-btn--active\">Recent") {
		t.Error("expected Recent button to be active by default")
	}
}

func TestExecute_HashRoute(t *testing.T) {
	html, err := Execute("#/portfolios/infrastructure")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if !strings.Contains(html, "infrastructure") {
		t.Error("expected hash route to resolve correctly")
	}
}

func TestParseInput_Defaults(t *testing.T) {
	input := ParseInput("")
	if input.Route != "/" {
		t.Errorf("expected route '/', got %q", input.Route)
	}
	if input.Sort != "time" {
		t.Errorf("expected sort 'time', got %q", input.Sort)
	}
	if input.Theme != "light" {
		t.Errorf("expected theme 'light', got %q", input.Theme)
	}
}

func TestParseInput_WithParams(t *testing.T) {
	input := ParseInput("/portfolios/infra?sort=name&theme=dark")
	if input.Route != "/portfolios/infra" {
		t.Errorf("expected route '/portfolios/infra', got %q", input.Route)
	}
	if input.Sort != "name" {
		t.Errorf("expected sort 'name', got %q", input.Sort)
	}
	if input.Theme != "dark" {
		t.Errorf("expected theme 'dark', got %q", input.Theme)
	}
}

func TestParseInput_HashPrefix(t *testing.T) {
	input := ParseInput("#/portfolios")
	if input.Route != "/portfolios" {
		t.Errorf("expected route '/portfolios', got %q", input.Route)
	}
}
