package httpdriver

import (
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/alexandremahdhaoui/forge-ui/internal/types"
	"github.com/alexandremahdhaoui/forge-ui/internal/util/mocks/mockcontroller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestHandler creates a Handler with minimal templates and mock services.
func newTestHandler(ps *mockcontroller.PortfolioService, ws *mockcontroller.WorkspaceService, fs *mockcontroller.ForgeService) *Handler {
	tmpl := template.Must(template.New("layout").Parse(`{{define "layout"}}ok{{end}}`))
	return &Handler{
		BaseDir: "/base",
		Templates: map[string]*template.Template{
			"portfolios": tmpl,
			"portfolio":  tmpl,
			"workspace":  tmpl,
			"forge":      tmpl,
		},
		HomeURL:          "/portfolios",
		PortfolioService: ps,
		WorkspaceService: ws,
		ForgeService:     fs,
	}
}

// --- HandlePortfolios ---

func TestHandlePortfolios_Success(t *testing.T) {
	t.Parallel()

	ps := new(mockcontroller.PortfolioService)
	h := newTestHandler(ps, nil, nil)

	ps.On("ListPortfolios", "/base", "time").Return(types.PortfoliosPageData{
		Portfolios: []types.PortfolioSummary{{Name: "p1"}},
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/portfolios", nil)
	w := httptest.NewRecorder()

	h.HandlePortfolios(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
	ps.AssertExpectations(t)
}

func TestHandlePortfolios_SortParam(t *testing.T) {
	t.Parallel()

	ps := new(mockcontroller.PortfolioService)
	h := newTestHandler(ps, nil, nil)

	ps.On("ListPortfolios", "/base", "name").Return(types.PortfoliosPageData{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/portfolios?sort=name", nil)
	w := httptest.NewRecorder()

	h.HandlePortfolios(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	ps.AssertExpectations(t)
}

func TestHandlePortfolios_Error(t *testing.T) {
	t.Parallel()

	ps := new(mockcontroller.PortfolioService)
	h := newTestHandler(ps, nil, nil)

	ps.On("ListPortfolios", "/base", "time").Return(types.PortfoliosPageData{}, errors.New("disk error"))

	req := httptest.NewRequest(http.MethodGet, "/portfolios", nil)
	w := httptest.NewRecorder()

	h.HandlePortfolios(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "disk error")
	ps.AssertExpectations(t)
}

// --- HandlePortfolio ---

func TestHandlePortfolio_Success(t *testing.T) {
	t.Parallel()

	ps := new(mockcontroller.PortfolioService)
	h := newTestHandler(ps, nil, nil)

	ps.On("GetPortfolio", "/base", "myp", "time").Return(types.PortfolioPageData{
		Name: "myp",
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/portfolios/myp", nil)
	req.SetPathValue("name", "myp")
	w := httptest.NewRecorder()

	h.HandlePortfolio(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	ps.AssertExpectations(t)
}

func TestHandlePortfolio_MissingName(t *testing.T) {
	t.Parallel()

	h := newTestHandler(nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/portfolios/", nil)
	// No path value set
	w := httptest.NewRecorder()

	h.HandlePortfolio(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "missing portfolio name")
}

func TestHandlePortfolio_Error(t *testing.T) {
	t.Parallel()

	ps := new(mockcontroller.PortfolioService)
	h := newTestHandler(ps, nil, nil)

	ps.On("GetPortfolio", "/base", "bad", "time").Return(types.PortfolioPageData{}, errors.New("not found"))

	req := httptest.NewRequest(http.MethodGet, "/portfolios/bad", nil)
	req.SetPathValue("name", "bad")
	w := httptest.NewRecorder()

	h.HandlePortfolio(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "not found")
	ps.AssertExpectations(t)
}

// --- HandleWorkspace ---

func TestHandleWorkspace_Success(t *testing.T) {
	t.Parallel()

	wsSvc := new(mockcontroller.WorkspaceService)
	h := newTestHandler(nil, wsSvc, nil)

	wsSvc.On("GetWorkspace", "/base", "p1", "ws1", "time").Return(types.WorkspacePageData{
		Name: "ws1",
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/portfolios/p1/workspaces/ws1", nil)
	req.SetPathValue("p", "p1")
	req.SetPathValue("w", "ws1")
	w := httptest.NewRecorder()

	h.HandleWorkspace(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	wsSvc.AssertExpectations(t)
}

func TestHandleWorkspace_MissingParams(t *testing.T) {
	t.Parallel()

	h := newTestHandler(nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/portfolios//workspaces/", nil)
	w := httptest.NewRecorder()

	h.HandleWorkspace(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "portfolio and workspace name required")
}

func TestHandleWorkspace_Error(t *testing.T) {
	t.Parallel()

	wsSvc := new(mockcontroller.WorkspaceService)
	h := newTestHandler(nil, wsSvc, nil)

	wsSvc.On("GetWorkspace", "/base", "p1", "missing", "time").Return(types.WorkspacePageData{}, errors.New("not found"))

	req := httptest.NewRequest(http.MethodGet, "/portfolios/p1/workspaces/missing", nil)
	req.SetPathValue("p", "p1")
	req.SetPathValue("w", "missing")
	w := httptest.NewRecorder()

	h.HandleWorkspace(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "not found")
	wsSvc.AssertExpectations(t)
}

// --- HandleForge ---

func TestHandleForge_Success(t *testing.T) {
	t.Parallel()

	fsSvc := new(mockcontroller.ForgeService)
	h := newTestHandler(nil, nil, fsSvc)

	fsSvc.On("GetForge", "/base", "p1", "ws1", "repo-a").Return(types.ForgePageData{
		RepoName: "repo-a",
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/portfolios/p1/workspaces/ws1/repos/repo-a", nil)
	req.SetPathValue("p", "p1")
	req.SetPathValue("w", "ws1")
	req.SetPathValue("r", "repo-a")
	w := httptest.NewRecorder()

	h.HandleForge(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	fsSvc.AssertExpectations(t)
}

func TestHandleForge_MissingParams(t *testing.T) {
	t.Parallel()

	h := newTestHandler(nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/portfolios/p1/workspaces/ws1/repos/", nil)
	req.SetPathValue("p", "p1")
	req.SetPathValue("w", "ws1")
	// No "r" path value
	w := httptest.NewRecorder()

	h.HandleForge(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "portfolio, workspace and repo name required")
}

func TestHandleForge_Error(t *testing.T) {
	t.Parallel()

	fsSvc := new(mockcontroller.ForgeService)
	h := newTestHandler(nil, nil, fsSvc)

	fsSvc.On("GetForge", "/base", "p1", "ws1", "bad-repo").Return(types.ForgePageData{}, errors.New("no forge.yaml"))

	req := httptest.NewRequest(http.MethodGet, "/portfolios/p1/workspaces/ws1/repos/bad-repo", nil)
	req.SetPathValue("p", "p1")
	req.SetPathValue("w", "ws1")
	req.SetPathValue("r", "bad-repo")
	w := httptest.NewRecorder()

	h.HandleForge(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "no forge.yaml")
	fsSvc.AssertExpectations(t)
}

// --- HandleRedirect ---

func TestHandleRedirect(t *testing.T) {
	t.Parallel()

	h := newTestHandler(nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	h.HandleRedirect(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/portfolios", w.Header().Get("Location"))
}

// --- HandleToggleTheme ---

func TestHandleToggleTheme_LightToDark(t *testing.T) {
	t.Parallel()

	h := newTestHandler(nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/theme/toggle", nil)
	req.Header.Set("Referer", "/portfolios")
	w := httptest.NewRecorder()

	h.HandleToggleTheme(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/portfolios", w.Header().Get("Location"))
	cookies := w.Result().Cookies()
	var themeCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "theme" {
			themeCookie = c
			break
		}
	}
	assert.NotNil(t, themeCookie)
	assert.Equal(t, "dark", themeCookie.Value)
}

func TestHandleToggleTheme_DarkToLight(t *testing.T) {
	t.Parallel()

	h := newTestHandler(nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/theme/toggle", nil)
	req.AddCookie(&http.Cookie{Name: "theme", Value: "dark"})
	req.Header.Set("Referer", "/portfolios/p1")
	w := httptest.NewRecorder()

	h.HandleToggleTheme(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	cookies := w.Result().Cookies()
	var themeCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "theme" {
			themeCookie = c
			break
		}
	}
	assert.NotNil(t, themeCookie)
	assert.Equal(t, "light", themeCookie.Value)
}

// --- isDarkMode ---

func TestIsDarkMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cookie   *http.Cookie
		expected bool
	}{
		{"no cookie", nil, false},
		{"light", &http.Cookie{Name: "theme", Value: "light"}, false},
		{"dark", &http.Cookie{Name: "theme", Value: "dark"}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.cookie != nil {
				req.AddCookie(tc.cookie)
			}
			got := isDarkMode(req)
			assert.Equal(t, tc.expected, got)
		})
	}
}

// --- HandleToggleTheme (no Referer) ---

func TestHandleToggleTheme_NoReferer(t *testing.T) {
	t.Parallel()

	h := newTestHandler(nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/theme/toggle", nil)
	// No Referer header set.
	w := httptest.NewRecorder()

	h.HandleToggleTheme(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/portfolios", w.Header().Get("Location"))
}

// --- HandleSetLightPalette ---

func TestHandleSetLightPalette_ValidValues(t *testing.T) {
	t.Parallel()

	for _, n := range []string{"1", "2", "3", "4"} {
		t.Run("palette_"+n, func(t *testing.T) {
			h := newTestHandler(nil, nil, nil)

			req := httptest.NewRequest(http.MethodGet, "/palette/set/"+n, nil)
			req.SetPathValue("n", n)
			req.Header.Set("Referer", "/portfolios/p1")
			w := httptest.NewRecorder()

			h.HandleSetLightPalette(w, req)

			assert.Equal(t, http.StatusSeeOther, w.Code)
			assert.Equal(t, "/portfolios/p1", w.Header().Get("Location"))

			cookies := w.Result().Cookies()
			var paletteCookie *http.Cookie
			for _, c := range cookies {
				if c.Name == "light-palette" {
					paletteCookie = c
					break
				}
			}
			require.NotNil(t, paletteCookie)
			assert.Equal(t, n, paletteCookie.Value)
		})
	}
}

func TestHandleSetLightPalette_InvalidValue(t *testing.T) {
	t.Parallel()

	h := newTestHandler(nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/palette/set/99", nil)
	req.SetPathValue("n", "99")
	req.Header.Set("Referer", "/portfolios")
	w := httptest.NewRecorder()

	h.HandleSetLightPalette(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)

	cookies := w.Result().Cookies()
	var paletteCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "light-palette" {
			paletteCookie = c
			break
		}
	}
	require.NotNil(t, paletteCookie)
	assert.Equal(t, "1", paletteCookie.Value)
}

func TestHandleSetLightPalette_NoReferer(t *testing.T) {
	t.Parallel()

	h := newTestHandler(nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/palette/set/2", nil)
	req.SetPathValue("n", "2")
	// No Referer header set.
	w := httptest.NewRecorder()

	h.HandleSetLightPalette(w, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/portfolios", w.Header().Get("Location"))
}

// --- lightPalette ---

func TestLightPalette(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cookies  []*http.Cookie
		expected string
	}{
		{
			"no cookie returns 1",
			nil,
			"1",
		},
		{
			"dark mode returns empty",
			[]*http.Cookie{{Name: "theme", Value: "dark"}},
			"",
		},
		{
			"palette cookie 1",
			[]*http.Cookie{{Name: "light-palette", Value: "1"}},
			"1",
		},
		{
			"palette cookie 2",
			[]*http.Cookie{{Name: "light-palette", Value: "2"}},
			"2",
		},
		{
			"palette cookie 3",
			[]*http.Cookie{{Name: "light-palette", Value: "3"}},
			"3",
		},
		{
			"palette cookie 4",
			[]*http.Cookie{{Name: "light-palette", Value: "4"}},
			"4",
		},
		{
			"invalid palette value falls back to 1",
			[]*http.Cookie{{Name: "light-palette", Value: "invalid"}},
			"1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			for _, c := range tc.cookies {
				req.AddCookie(c)
			}
			got := lightPalette(req)
			assert.Equal(t, tc.expected, got)
		})
	}
}

// --- New ---

func TestNew_Success(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	templateFiles := map[string]string{
		"layout.html":     `{{define "layout"}}{{template "content" .}}{{end}}`,
		"portfolios.html": `{{define "content"}}portfolios{{end}}`,
		"portfolio.html":  `{{define "content"}}portfolio{{end}}`,
		"workspace.html":  `{{define "content"}}workspace{{end}}`,
		"forge.html":      `{{define "content"}}forge{{end}}`,
	}
	for name, content := range templateFiles {
		err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0o644)
		require.NoError(t, err)
	}

	ps := new(mockcontroller.PortfolioService)
	ws := new(mockcontroller.WorkspaceService)
	fs := new(mockcontroller.ForgeService)

	h, err := New("/base", tmpDir, ps, ws, fs)

	require.NoError(t, err)
	assert.NotNil(t, h)
	assert.Equal(t, "/base", h.BaseDir)
	assert.Equal(t, "/portfolios", h.HomeURL)
	assert.Len(t, h.Templates, 4)
	for _, page := range []string{"portfolios", "portfolio", "workspace", "forge"} {
		assert.Contains(t, h.Templates, page)
	}
}

func TestNew_MissingTemplate(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Only write layout.html, omit page templates.
	err := os.WriteFile(filepath.Join(tmpDir, "layout.html"), []byte(`{{define "layout"}}{{end}}`), 0o644)
	require.NoError(t, err)

	ps := new(mockcontroller.PortfolioService)
	ws := new(mockcontroller.WorkspaceService)
	fs := new(mockcontroller.ForgeService)

	h, err := New("/base", tmpDir, ps, ws, fs)

	assert.Error(t, err)
	assert.Nil(t, h)
	assert.Contains(t, err.Error(), "parsing template")
}

func TestNew_PercentFuncMap(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	templateFiles := map[string]string{
		"layout.html":     `{{define "layout"}}{{template "content" .}}{{end}}`,
		"portfolios.html": `{{define "content"}}{{percent 3 10}}{{end}}`,
		"portfolio.html":  `{{define "content"}}portfolio{{end}}`,
		"workspace.html":  `{{define "content"}}workspace{{end}}`,
		"forge.html":      `{{define "content"}}forge{{end}}`,
	}
	for name, content := range templateFiles {
		err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0o644)
		require.NoError(t, err)
	}

	h, err := New("/base", tmpDir, nil, nil, nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	h.render(w, "portfolios", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "30")
}

func TestNew_PercentFuncMap_ZeroTotal(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	templateFiles := map[string]string{
		"layout.html":     `{{define "layout"}}{{template "content" .}}{{end}}`,
		"portfolios.html": `{{define "content"}}{{percent 5 0}}{{end}}`,
		"portfolio.html":  `{{define "content"}}portfolio{{end}}`,
		"workspace.html":  `{{define "content"}}workspace{{end}}`,
		"forge.html":      `{{define "content"}}forge{{end}}`,
	}
	for name, content := range templateFiles {
		err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0o644)
		require.NoError(t, err)
	}

	h, err := New("/base", tmpDir, nil, nil, nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	h.render(w, "portfolios", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "0")
}

// --- render ---

func TestRender_UnknownTemplate(t *testing.T) {
	t.Parallel()

	h := newTestHandler(nil, nil, nil)

	w := httptest.NewRecorder()

	h.render(w, "nonexistent", nil)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "template not found")
}

func TestRender_ExecutionError(t *testing.T) {
	t.Parallel()

	// Create a template that references a missing sub-template to trigger execution error.
	tmpl := template.Must(template.New("layout").Parse(`{{define "layout"}}{{template "missing" .}}{{end}}`))
	h := &Handler{
		BaseDir: "/base",
		Templates: map[string]*template.Template{
			"portfolios": tmpl,
		},
		HomeURL: "/portfolios",
	}

	w := httptest.NewRecorder()
	h.render(w, "portfolios", nil)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
